package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/segmentio/kafka-go"
)

// EventPublisher handles publishing events from the outbox to Kafka
// Implements the Transactional Outbox Pattern for reliable event delivery
type EventPublisher struct {
	outboxRepo     *repository.OutboxRepository
	kafkaWriter    *kafka.Writer
	maxRetries     int
	isInitialized  bool
}

// EventPublisherConfig holds configuration for the event publisher
type EventPublisherConfig struct {
	KafkaBrokers []string
	MaxRetries   int // Maximum retry attempts before marking event as failed
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(db *sql.DB, config EventPublisherConfig) *EventPublisher {
	outboxRepo := repository.NewOutboxRepository(db)

	// Note: Writer will be initialized later with topic from each event
	// This is a placeholder writer that won't be used directly
	kafkaWriter := &kafka.Writer{
		Addr:                   kafka.TCP(config.KafkaBrokers...),
		Balancer:               &kafka.LeastBytes{},
		MaxAttempts:            3,
		RequiredAcks:           kafka.RequireOne,
		Async:                  false,
		Compression:            kafka.Snappy,
		AllowAutoTopicCreation: true,
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5 // Default max retries
	}

	return &EventPublisher{
		outboxRepo:    outboxRepo,
		kafkaWriter:   kafkaWriter,
		maxRetries:    maxRetries,
		isInitialized: true,
	}
}

// CreateEvent inserts an event into the outbox within a transaction
// This ensures atomic writes: business operation + event creation
func (ep *EventPublisher) CreateEvent(ctx context.Context, tx *sql.Tx, req *models.CreateEventOutboxRequest) error {
	event := &models.EventOutbox{
		EventType:    req.EventType,
		EventKey:     req.EventKey,
		EventPayload: req.EventPayload,
		Topic:        req.Topic,
	}

	return ep.outboxRepo.Create(ctx, tx, event)
}

// PublishPendingEvents polls the outbox and publishes pending events to Kafka
// Called by the background worker on a regular interval
func (ep *EventPublisher) PublishPendingEvents(ctx context.Context, batchSize int) (int, int, error) {
	if !ep.isInitialized {
		return 0, 0, fmt.Errorf("event publisher not initialized")
	}

	// Fetch pending events from outbox
	events, err := ep.outboxRepo.GetPendingEvents(ctx, batchSize)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch pending events: %w", err)
	}

	if len(events) == 0 {
		return 0, 0, nil // No events to publish
	}

	successCount := 0
	failureCount := 0

	for _, event := range events {
		// Check if event has exceeded retry limit
		if event.RetryCount >= ep.maxRetries {
			log.Printf("[EventPublisher] Event %s exceeded max retries (%d), skipping", event.ID, ep.maxRetries)
			failureCount++
			continue
		}

		// Publish to Kafka
		err := ep.publishEventToKafka(ctx, &event)
		if err != nil {
			// Record error in outbox
			errMsg := err.Error()
			if recordErr := ep.outboxRepo.RecordError(ctx, event.ID, errMsg); recordErr != nil {
				log.Printf("[EventPublisher] Failed to record error for event %s: %v", event.ID, recordErr)
			}
			log.Printf("[EventPublisher] Failed to publish event %s to Kafka: %v", event.ID, err)
			failureCount++
			continue
		}

		// Mark event as published
		if markErr := ep.outboxRepo.MarkAsPublished(ctx, event.ID); markErr != nil {
			log.Printf("[EventPublisher] Failed to mark event %s as published: %v", event.ID, markErr)
			failureCount++
			continue
		}

		log.Printf("[EventPublisher] Successfully published event %s (type: %s) to topic: %s", 
			event.ID, event.EventType, event.Topic)
		successCount++
	}

	return successCount, failureCount, nil
}

// publishEventToKafka sends a single event to the specified Kafka topic
func (ep *EventPublisher) publishEventToKafka(ctx context.Context, event *models.EventOutbox) error {
	// Create a topic-specific writer for this event
	// Each event can target a different topic
	writer := &kafka.Writer{
		Addr:                   ep.kafkaWriter.Addr,
		Topic:                  event.Topic,
		Balancer:               ep.kafkaWriter.Balancer,
		MaxAttempts:            ep.kafkaWriter.MaxAttempts,
		RequiredAcks:           ep.kafkaWriter.RequiredAcks,
		Async:                  ep.kafkaWriter.Async,
		Compression:            ep.kafkaWriter.Compression,
		AllowAutoTopicCreation: ep.kafkaWriter.AllowAutoTopicCreation,
	}
	defer writer.Close()

	// Parse event payload to ensure it's valid JSON
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(event.EventPayload, &payloadMap); err != nil {
		return fmt.Errorf("invalid event payload JSON: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(event.EventKey),
		Value: event.EventPayload,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.EventType)},
			{Key: "event-id", Value: []byte(event.ID)},
		},
	}

	// Write message to Kafka
	if err := writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

// GetFailedEvents retrieves events that have exceeded max retry attempts
// Used for monitoring and manual intervention
func (ep *EventPublisher) GetFailedEvents(ctx context.Context) ([]models.EventOutbox, error) {
	return ep.outboxRepo.GetFailedEvents(ctx, ep.maxRetries)
}

// Close closes the Kafka writer connection
func (ep *EventPublisher) Close() error {
	if ep.kafkaWriter != nil {
		return ep.kafkaWriter.Close()
	}
	return nil
}
