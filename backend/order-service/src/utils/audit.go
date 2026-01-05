package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/point-of-sale-system/order-service/src/queue"
)

// AuditPublisher publishes audit events to Kafka with idempotency
// Implements FR-027: Immutable audit trail for all data access
type AuditPublisher struct {
	producer    *queue.KafkaProducer
	serviceName string
	mu          sync.Mutex
}

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	EventID      string                 `json:"event_id"`      // Idempotency key
	TenantID     string                 `json:"tenant_id"`     // Tenant isolation
	Timestamp    time.Time              `json:"timestamp"`     // Event timestamp
	ActorType    string                 `json:"actor_type"`    // user, system, guest, admin
	ActorID      *string                `json:"actor_id"`      // User ID (nullable)
	ActorEmail   *string                `json:"actor_email"`   // Email (encrypted)
	SessionID    *string                `json:"session_id"`    // Session ID (nullable)
	Action       string                 `json:"action"`        // CREATE, READ, UPDATE, DELETE, etc.
	ResourceType string                 `json:"resource_type"` // user, order, product, etc.
	ResourceID   string                 `json:"resource_id"`   // Resource identifier
	IPAddress    *string                `json:"ip_address"`    // Client IP
	UserAgent    *string                `json:"user_agent"`    // Browser user agent
	RequestID    *string                `json:"request_id"`    // Distributed tracing ID
	BeforeValue  map[string]interface{} `json:"before_value"`  // State before (encrypted PII)
	AfterValue   map[string]interface{} `json:"after_value"`   // State after (encrypted PII)
	Metadata     map[string]interface{} `json:"metadata"`      // Additional context
	Purpose      *string                `json:"purpose"`       // Legal basis (UU PDP Article 20)
	ConsentID    *string                `json:"consent_id"`    // Linked consent record
	ServiceName  string                 `json:"service_name"`  // Originating service
}

var (
	auditPublisherInstance *AuditPublisher
	auditPublisherOnce     sync.Once
)

// NewAuditPublisher creates a singleton Kafka producer for audit events
func NewAuditPublisher(serviceName string, kafkaBrokers []string, topic string) (*AuditPublisher, error) {
	auditPublisherOnce.Do(func() {
		config := queue.KafkaProducerConfig{
			Brokers:              kafkaBrokers,
			Topic:                topic,
			Balancer:             &kafka.Hash{}, // Partition by event_id for idempotency
			MaxAttempts:          3,
			RequiredAcks:         kafka.RequireAll, // Wait for all replicas
			Async:                false,            // Synchronous writes for reliability
			Compression:          kafka.Snappy,
			AllowAutoTopicCreate: false,
		}

		producer := queue.NewKafkaProducerWithConfig(config)

		auditPublisherInstance = &AuditPublisher{
			producer:    producer,
			serviceName: serviceName,
		}
	})

	return auditPublisherInstance, nil
}

// Publish publishes a single audit event to Kafka
// Event ID is used as Kafka message key for idempotency and partitioning
func (ap *AuditPublisher) Publish(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}

	// Generate event ID if not provided (idempotency key)
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Set service name
	event.ServiceName = ap.serviceName

	// Validate required fields
	if err := ap.validateEvent(event); err != nil {
		return fmt.Errorf("invalid audit event: %w", err)
	}

	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Prepare headers
	headers := []kafka.Header{
		{Key: "event_type", Value: []byte("audit")},
		{Key: "service", Value: []byte(ap.serviceName)},
		{Key: "tenant_id", Value: []byte(event.TenantID)},
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Use queue.KafkaProducer's PublishWithHeaders
	err = ap.producer.PublishWithHeaders(ctx, event.EventID, eventJSON, headers)
	if err != nil {
		return fmt.Errorf("failed to publish audit event to Kafka: %w", err)
	}

	return nil
}

// PublishBatch publishes multiple audit events in a single Kafka batch (performance optimization)
func (ap *AuditPublisher) PublishBatch(ctx context.Context, events []*AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	messages := make([]kafka.Message, len(events))

	for i, event := range events {
		if event == nil {
			return fmt.Errorf("audit event at index %d is nil", i)
		}

		// Generate event ID if not provided
		if event.EventID == "" {
			event.EventID = uuid.New().String()
		}

		// Set timestamp if not provided
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now().UTC()
		}

		// Set service name
		event.ServiceName = ap.serviceName

		// Validate
		if err := ap.validateEvent(event); err != nil {
			return fmt.Errorf("invalid audit event at index %d: %w", i, err)
		}

		// Serialize
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal audit event at index %d: %w", i, err)
		}

		messages[i] = kafka.Message{
			Key:   []byte(event.EventID),
			Value: eventJSON,
			Time:  event.Timestamp,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte("audit")},
				{Key: "service", Value: []byte(ap.serviceName)},
				{Key: "tenant_id", Value: []byte(event.TenantID)},
			},
		}
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Use queue.KafkaProducer's PublishBatch
	err := ap.producer.PublishBatch(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to publish audit event batch to Kafka: %w", err)
	}

	return nil
}

// validateEvent validates required fields per audit_events table schema
func (ap *AuditPublisher) validateEvent(event *AuditEvent) error {
	if event.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	if event.ActorType == "" {
		return fmt.Errorf("actor_type is required")
	}

	validActorTypes := map[string]bool{"user": true, "system": true, "guest": true, "admin": true}
	if !validActorTypes[event.ActorType] {
		return fmt.Errorf("actor_type must be one of: user, system, guest, admin")
	}

	if event.Action == "" {
		return fmt.Errorf("action is required")
	}

	validActions := map[string]bool{
		"CREATE": true, "READ": true, "UPDATE": true, "DELETE": true,
		"ACCESS": true, "EXPORT": true, "ANONYMIZE": true,
	}
	if !validActions[event.Action] {
		return fmt.Errorf("action must be one of: CREATE, READ, UPDATE, DELETE, ACCESS, EXPORT, ANONYMIZE")
	}

	if event.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}

	if event.ResourceID == "" {
		return fmt.Errorf("resource_id is required")
	}

	return nil
}

// Close closes the Kafka writer
func (ap *AuditPublisher) Close() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.producer != nil {
		return ap.producer.Close()
	}
	return nil
}

// Helper functions for common audit event creation

// NewUserEvent creates an audit event for user-related actions
func NewUserEvent(tenantID, userID, action, resourceID string) *AuditEvent {
	return &AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     tenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    "user",
		ActorID:      &userID,
		Action:       action,
		ResourceType: "user",
		ResourceID:   resourceID,
	}
}

// NewSystemEvent creates an audit event for system-initiated actions
func NewSystemEvent(tenantID, action, resourceType, resourceID string) *AuditEvent {
	return &AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     tenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    "system",
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// NewGuestEvent creates an audit event for guest order actions
func NewGuestEvent(tenantID, action, resourceID string) *AuditEvent {
	return &AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     tenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    "guest",
		Action:       action,
		ResourceType: "guest_order",
		ResourceID:   resourceID,
	}
}
