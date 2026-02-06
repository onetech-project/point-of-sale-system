package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type EventPublisher struct {
	writer        *kafka.Writer
	consentWriter *kafka.Writer // Dedicated writer for consent events
}

func NewEventPublisher(brokers []string, topic string, consentTopic string) *EventPublisher {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
	}

	consentWriter := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  consentTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
	}

	return &EventPublisher{
		writer:        writer,
		consentWriter: consentWriter,
	}
}

type NotificationEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	TenantID  string                 `json:"tenant_id"`
	UserID    string                 `json:"user_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

func (p *EventPublisher) PublishUserRegistered(ctx context.Context, tenantID, userID, email, name, verificationToken string) error {
	event := NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "user.registered",
		TenantID:  tenantID,
		UserID:    userID,
		Data: map[string]interface{}{
			"email":              email,
			"name":               name,
			"verification_token": verificationToken,
		},
		Timestamp: time.Now(),
	}

	return p.publish(ctx, event)
}

// PublishConsentGranted publishes a consent granted event to Kafka
// This should be called AFTER user/order creation to ensure proper subject_id
// Uses dedicated consent-events topic for audit-service consumption
func (p *EventPublisher) PublishConsentGranted(ctx context.Context, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal consent event: %w", err)
	}

	// Extract tenant_id for message key (assumes event has TenantID field)
	eventMap := make(map[string]interface{})
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return fmt.Errorf("failed to unmarshal for key extraction: %w", err)
	}
	
	tenantID, _ := eventMap["tenant_id"].(string)
	
	msg := kafka.Message{
		Key:   []byte(tenantID),
		Value: data,
		Time:  time.Now(),
	}

	// Use dedicated consent writer (consent-events topic)
	if err := p.consentWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write consent event to kafka: %w", err)
	}

	return nil
}

func (p *EventPublisher) publish(ctx context.Context, event NotificationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.TenantID),
		Value: data,
		Time:  event.Timestamp,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return nil
}

func (p *EventPublisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return err
	}
	return p.consentWriter.Close()
}
