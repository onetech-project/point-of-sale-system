package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"

	"github.com/pos/audit-service/src/events"
	"github.com/pos/audit-service/src/models"
	"github.com/pos/audit-service/src/repository"
	"github.com/pos/audit-service/src/utils"
)

// ConsentConsumer consumes consent events from Kafka and persists to database
type ConsentConsumer struct {
	reader      *kafka.Reader
	consentRepo *repository.ConsentRepository
	encryptor   utils.Encryptor
	dlqProducer *kafka.Writer
}

// NewConsentConsumer creates a new Kafka consumer for consent events
func NewConsentConsumer(config KafkaConsumerConfig, consentRepo *repository.ConsentRepository, encryptor utils.Encryptor) *ConsentConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{config.Brokers},
		Topic:          config.Topic, // Consent events topic (configurable)
		GroupID:        "audit-service-consent-consumer",
		StartOffset:    kafka.FirstOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 1 * time.Second,
	})

	// Create DLQ producer for failed messages
	dlqProducer := &kafka.Writer{
		Addr:     kafka.TCP(config.Brokers),
		Topic:    config.Topic + "-dlq",
		Balancer: &kafka.LeastBytes{},
	}

	return &ConsentConsumer{
		reader:      reader,
		consentRepo: consentRepo,
		encryptor:   encryptor,
		dlqProducer: dlqProducer,
	}
}

// Start begins consuming consent events from Kafka
func (c *ConsentConsumer) Start(ctx context.Context) {
	log.Info().Str("topic", c.reader.Config().Topic).Msg("Consent consumer started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Consent consumer shutting down")
			if err := c.reader.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close Kafka reader")
			}
			if err := c.dlqProducer.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close DLQ producer")
			}
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Error().Err(err).Msg("Failed to fetch Kafka message")
				time.Sleep(1 * time.Second)
				continue
			}

			if err := c.processMessageWithRetry(ctx, msg, 5); err != nil {
				log.Error().
					Err(err).
					Str("partition", fmt.Sprintf("%d", msg.Partition)).
					Str("offset", fmt.Sprintf("%d", msg.Offset)).
					Msg("Failed to process consent event after retries")

				// Send to DLQ after max retries
				c.sendToDLQ(msg, "processing_error", err)
			}

			// Commit offset after processing (success or DLQ)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("Failed to commit Kafka offset")
			}
		}
	}
}

// processMessageWithRetry attempts to process message with exponential backoff
func (c *ConsentConsumer) processMessageWithRetry(ctx context.Context, msg kafka.Message, maxRetries int) error {
	var lastErr error
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := c.processMessage(ctx, msg)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_retries", maxRetries).
			Msg("Consent processing failed, retrying")

		if attempt < maxRetries {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// processMessage deserializes and persists consent event
func (c *ConsentConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event events.ConsentGrantedEvent

	// Deserialize JSON message
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		// Malformed event → DLQ immediately
		c.sendToDLQ(msg, "unmarshal_error", err)
		return nil // Don't retry unmarshaling errors
	}

	log.Info().
		Str("event_id", event.EventID).
		Str("tenant_id", event.TenantID).
		Str("subject_id", event.SubjectID).
		Str("subject_type", event.SubjectType).
		Interface("optional_consents", event.Consents).
		Interface("required_consents", event.RequiredConsents).
		Str("ip_address", event.Metadata.IPAddress).
		Str("user_agent", event.Metadata.UserAgent).
		Msg("Processing consent event")

	// Validate required fields
	if event.TenantID == "" || event.SubjectID == "" || event.SubjectType == "" {
		err := fmt.Errorf("missing required fields: tenant_id=%s, subject_id=%s, subject_type=%s",
			event.TenantID, event.SubjectID, event.SubjectType)
		c.sendToDLQ(msg, "validation_error", err)
		return nil // Don't retry validation errors
	}

	// Check idempotency - has this event been processed already?
	processed, err := c.consentRepo.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check event processing status: %w", err)
	}
	if processed {
		log.Info().
			Str("event_id", event.EventID).
			Str("subject_id", event.SubjectID).
			Msg("Event already processed, skipping")
		return nil
	}

	// Encrypt IP address via Vault
	var encryptedIP *string
	if event.Metadata.IPAddress != "" {
		encrypted, err := c.encryptor.EncryptWithContext(ctx, event.Metadata.IPAddress, "consent:ip")
		if err != nil {
			return fmt.Errorf("failed to encrypt IP address: %w", err)
		}
		encryptedIP = &encrypted
	}

	// Parse subject_id as UUID
	subjectUUID, err := uuid.Parse(event.SubjectID)
	if err != nil {
		c.sendToDLQ(msg, "invalid_subject_id", err)
		return nil // Don't retry invalid UUIDs
	}

	// Combine required and optional consents for recording
	allConsents := append(event.RequiredConsents, event.Consents...)

	log.Info().
		Str("event_id", event.EventID).
		Int("total_consents", len(allConsents)).
		Interface("all_consents", allConsents).
		Msg("Combined consents for recording")

	// Insert consent records for all granted consents
	for _, purposeCode := range allConsents {
		subjectIDStr := subjectUUID.String()
		record := &models.ConsentRecord{
			RecordID:      uuid.New(),
			TenantID:      event.TenantID,
			SubjectType:   event.SubjectType,
			SubjectID:     &subjectIDStr, // SubjectID is used for both user and guest
			PurposeCode:   purposeCode,
			Granted:       true, // All consents in event are granted
			PolicyVersion: event.PolicyVersion,
			ConsentMethod: event.ConsentMethod,
			IPAddress:     encryptedIP,
			UserAgent:     &event.Metadata.UserAgent,
		}

		log.Info().
			Str("event_id", event.EventID).
			Str("purpose_code", purposeCode).
			Str("tenant_id", event.TenantID).
			Msg("Creating consent record")

		if err := c.consentRepo.CreateConsentRecord(ctx, record); err != nil {
			log.Error().
				Err(err).
				Str("purpose_code", purposeCode).
				Msg("Failed to create consent record")
			return fmt.Errorf("failed to create consent record for purpose %s: %w", purposeCode, err)
		}

		log.Info().
			Str("event_id", event.EventID).
			Str("purpose_code", purposeCode).
			Msg("Consent record created successfully")
	}

	// Mark event as processed (idempotency)
	if err := c.consentRepo.MarkEventProcessed(ctx, event.EventID, event.TenantID, event.SubjectType, subjectUUID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	log.Info().
		Str("event_id", event.EventID).
		Str("subject_type", event.SubjectType).
		Str("subject_id", event.SubjectID).
		Int("consents_count", len(allConsents)).
		Msg("Consent event processed successfully")

	return nil
}

// sendToDLQ sends failed message to Dead Letter Queue
func (c *ConsentConsumer) sendToDLQ(msg kafka.Message, reason string, err error) {
	dlqMsg := kafka.Message{
		Key:   msg.Key,
		Value: msg.Value,
		Headers: []kafka.Header{
			{Key: "error_reason", Value: []byte(reason)},
			{Key: "error_message", Value: []byte(err.Error())},
			{Key: "original_topic", Value: []byte("consents")},
			{Key: "original_partition", Value: []byte(fmt.Sprintf("%d", msg.Partition))},
			{Key: "original_offset", Value: []byte(fmt.Sprintf("%d", msg.Offset))},
			{Key: "failed_at", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	if err := c.dlqProducer.WriteMessages(context.Background(), dlqMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send message to DLQ")
	} else {
		log.Warn().
			Str("reason", reason).
			Str("partition", fmt.Sprintf("%d", msg.Partition)).
			Str("offset", fmt.Sprintf("%d", msg.Offset)).
			Msg("Message sent to DLQ")
	}
}
