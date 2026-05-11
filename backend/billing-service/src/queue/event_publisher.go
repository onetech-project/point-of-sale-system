package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// NotificationEvent is the Kafka message envelope.
type NotificationEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	TenantID  string                 `json:"tenant_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventPublisher publishes billing events to Kafka.
type EventPublisher struct {
	writer      *kafka.Writer
	auditWriter *kafka.Writer
}

// NewEventPublisher creates an EventPublisher connected to the given brokers and topic.
func NewEventPublisher(brokers []string, topic string) *EventPublisher {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
	}

	var auditWriter *kafka.Writer
	if auditTopic := os.Getenv("KAFKA_AUDIT_TOPIC"); auditTopic != "" {
		auditWriter = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  auditTopic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
			Async:                  false,
		}
	}

	return &EventPublisher{writer: writer, auditWriter: auditWriter}
}

// Close shuts down the Kafka writer.
func (p *EventPublisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return err
	}
	if p.auditWriter != nil {
		return p.auditWriter.Close()
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

// PublishTrialEnding notifies that a trial is ending soon.
func (p *EventPublisher) PublishTrialEnding(ctx context.Context, tenantID, email, tenantName string, trialEndsAt time.Time, daysRemaining int) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "subscription.trial_ending",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":          email,
			"tenant_name":    tenantName,
			"trial_ends_at":  trialEndsAt.Format(time.RFC3339),
			"days_remaining": daysRemaining,
		},
		Timestamp: time.Now(),
	})
}

// PublishGracePeriodStarted notifies that the grace period has started.
func (p *EventPublisher) PublishGracePeriodStarted(ctx context.Context, tenantID, email, tenantName string, graceEndsAt time.Time) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "subscription.grace_period_started",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":         email,
			"tenant_name":   tenantName,
			"grace_ends_at": graceEndsAt.Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	})
}

// PublishSubscriptionExpired notifies that the subscription has expired.
func (p *EventPublisher) PublishSubscriptionExpired(ctx context.Context, tenantID, email, tenantName string) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "subscription.expired",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":       email,
			"tenant_name": tenantName,
		},
		Timestamp: time.Now(),
	})
}

// PublishInvoiceGenerated notifies that a new invoice has been generated.
func (p *EventPublisher) PublishInvoiceGenerated(ctx context.Context, tenantID, email, tenantName, invoiceNumber string, amountIDR int, billingInterval string, periodStart, periodEnd time.Time, paymentURL string) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "invoice.generated",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":            email,
			"tenant_name":      tenantName,
			"invoice_number":   invoiceNumber,
			"amount_idr":       amountIDR,
			"billing_interval": billingInterval,
			"period_start":     periodStart.Format(time.RFC3339),
			"period_end":       periodEnd.Format(time.RFC3339),
			"payment_url":      paymentURL,
		},
		Timestamp: time.Now(),
	})
}

// PublishInvoicePaid notifies that an invoice has been paid.
func (p *EventPublisher) PublishInvoicePaid(ctx context.Context, tenantID, email, tenantName, invoiceNumber string, amountIDR int, billingInterval string, periodStart, periodEnd time.Time) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "invoice.paid",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":            email,
			"tenant_name":      tenantName,
			"invoice_number":   invoiceNumber,
			"amount_idr":       amountIDR,
			"billing_interval": billingInterval,
			"period_start":     periodStart.Format(time.RFC3339),
			"period_end":       periodEnd.Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	})
}

// PublishInvoicePaymentFailed notifies that a payment attempt failed.
func (p *EventPublisher) PublishInvoicePaymentFailed(ctx context.Context, tenantID, email, tenantName, invoiceNumber, errorMsg string) error {
	return p.publish(ctx, NotificationEvent{
		EventID:   uuid.New().String(),
		EventType: "invoice.payment_failed",
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"email":          email,
			"tenant_name":    tenantName,
			"invoice_number": invoiceNumber,
			"error_msg":      errorMsg,
		},
		Timestamp: time.Now(),
	})
}

// PublishTenantSubscriptionDataAnonymized emits the compliance audit event for retention cleanup.
func (p *EventPublisher) PublishTenantSubscriptionDataAnonymized(ctx context.Context, tenantID string, retentionStartedAt *time.Time, anonymizedAt time.Time) error {
	if p.auditWriter == nil {
		return nil
	}

	now := time.Now().UTC()
	metadata := map[string]interface{}{
		"reason":         "subscription_retention_cleanup",
		"retention_days": 30,
		"anonymized_at":  anonymizedAt.Format(time.RFC3339),
	}
	if retentionStartedAt != nil {
		metadata["retention_started_at"] = retentionStartedAt.Format(time.RFC3339)
	}

	payload := map[string]interface{}{
		"event_id":      uuid.New().String(),
		"tenant_id":     tenantID,
		"timestamp":     now.Format(time.RFC3339),
		"actor_type":    "system",
		"action":        "ANONYMIZE",
		"resource_type": "tenant_subscription_data",
		"resource_id":   tenantID,
		"purpose":       "subscription_retention_cleanup",
		"metadata":      metadata,
		"created_at":    now.Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}
	msg := kafka.Message{
		Key:   []byte(tenantID),
		Value: data,
		Time:  now,
	}
	if err := p.auditWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write audit event to kafka: %w", err)
	}
	return nil
}
