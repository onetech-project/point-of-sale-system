package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"

	"github.com/pos/audit-service/src/models"
	"github.com/pos/audit-service/src/observability"
	"github.com/pos/audit-service/src/repository"
)

// KafkaConsumerConfig holds configuration for Kafka consumer
type KafkaConsumerConfig struct {
	Brokers     string // Comma-separated list
	Topic       string
	GroupID     string
	StartOffset int64 // -1 for latest, -2 for earliest
}

// AuditConsumer consumes audit events from Kafka and persists to database
type AuditConsumer struct {
	reader    *kafka.Reader
	auditRepo *repository.AuditRepository
}

// NewAuditConsumer creates a new Kafka consumer for audit events
func NewAuditConsumer(config KafkaConsumerConfig, auditRepo *repository.AuditRepository) *AuditConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{config.Brokers},
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		StartOffset:    config.StartOffset,
		MinBytes:       1,    // 1 byte
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 1 * time.Second,
	})

	return &AuditConsumer{
		reader:    reader,
		auditRepo: auditRepo,
	}
}

// Start begins consuming messages from Kafka
func (c *AuditConsumer) Start(ctx context.Context) {
	log.Info().Str("topic", c.reader.Config().Topic).Msg("Audit consumer started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Audit consumer shutting down")
			if err := c.reader.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close Kafka reader")
			}
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Error().Err(err).Msg("Failed to fetch Kafka message")
				time.Sleep(1 * time.Second) // Backoff
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("partition", fmt.Sprintf("%d", msg.Partition)).
					Str("offset", fmt.Sprintf("%d", msg.Offset)).
					Msg("Failed to process audit event")
				// Continue processing next message (at-least-once delivery)
			}

			// Commit offset after successful processing
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("Failed to commit Kafka offset")
			}
		}
	}
}

// processMessage deserializes and persists audit event (T116: with metrics)
func (c *AuditConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	startTime := time.Now()

	var auditEvent models.AuditEvent

	// Deserialize JSON message
	if err := json.Unmarshal(msg.Value, &auditEvent); err != nil {
		observability.AuditEventsPersistErrorsTotal.WithLabelValues("unmarshal_error").Inc()
		log.Error().
			Err(err).
			Str("raw_message", string(msg.Value)).
			Msg("Failed to unmarshal audit event")
		return fmt.Errorf("failed to unmarshal audit event: %w", err)
	}

	// Validate required fields
	if auditEvent.TenantID == "" {
		observability.AuditEventsPersistErrorsTotal.WithLabelValues("validation_error").Inc()
		return fmt.Errorf("audit event missing tenant_id")
	}
	if auditEvent.Action == "" {
		observability.AuditEventsPersistErrorsTotal.WithLabelValues("validation_error").Inc()
		return fmt.Errorf("audit event missing action")
	}
	if auditEvent.ResourceType == "" {
		observability.AuditEventsPersistErrorsTotal.WithLabelValues("validation_error").Inc()
		return fmt.Errorf("audit event missing resource_type")
	}

	// Persist to database (partition-aware insert)
	if err := c.auditRepo.Create(ctx, &auditEvent); err != nil {
		observability.AuditEventsPersistErrorsTotal.WithLabelValues("database_error").Inc()
		observability.AuditEventsPersistedTotal.WithLabelValues(auditEvent.Action, auditEvent.ResourceType, "error").Inc()
		return fmt.Errorf("failed to persist audit event: %w", err)
	}

	// T116: Record successful persistence metrics
	duration := time.Since(startTime).Seconds()
	observability.AuditEventsPersistedTotal.WithLabelValues(auditEvent.Action, auditEvent.ResourceType, "success").Inc()
	observability.AuditEventsProcessingDuration.WithLabelValues(auditEvent.Action, auditEvent.ResourceType).Observe(duration)

	// Update consumer lag metric (T117 alert trigger)
	stats := c.reader.Stats()
	observability.AuditKafkaConsumerLag.Set(float64(stats.Lag))
	observability.AuditKafkaConsumerOffset.Set(float64(stats.Offset))

	log.Debug().
		Str("event_id", auditEvent.EventID.String()).
		Str("tenant_id", auditEvent.TenantID).
		Str("action", auditEvent.Action).
		Str("resource_type", auditEvent.ResourceType).
		Str("partition", auditEvent.PartitionName()).
		Float64("processing_duration_seconds", duration).
		Msg("Audit event persisted")

	return nil
}
