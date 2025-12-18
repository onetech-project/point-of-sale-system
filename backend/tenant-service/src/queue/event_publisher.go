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
	writer *kafka.Writer
}

func NewEventPublisher(brokers []string, topic string) *EventPublisher {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
	}

	return &EventPublisher{writer: writer}
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
	return p.writer.Close()
}
