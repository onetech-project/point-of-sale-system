package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// EventOutbox represents an event waiting to be published to Kafka
// Implements the Transactional Outbox Pattern for reliable event publishing
type EventOutbox struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`    // e.g., "offline_order.created"
	EventKey     string          `json:"event_key"`     // Kafka partition key (e.g., order_id)
	EventPayload json.RawMessage `json:"event_payload"` // Full event payload as JSON
	Topic        string          `json:"topic"`         // Target Kafka topic
	CreatedAt    time.Time       `json:"created_at"`
	PublishedAt  *time.Time      `json:"published_at,omitempty"` // NULL = pending, NOT NULL = published
	RetryCount   int             `json:"retry_count"`
	LastError    *string         `json:"last_error,omitempty"`
}

// EventPayload represents the structure of event_payload JSONB
type EventPayload map[string]interface{}

// Scan implements sql.Scanner for EventPayload (JSONB)
func (ep *EventPayload) Scan(value interface{}) error {
	if value == nil {
		*ep = EventPayload{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, ep)
}

// Value implements driver.Valuer for EventPayload (JSONB)
func (ep EventPayload) Value() (driver.Value, error) {
	if ep == nil {
		return nil, nil
	}
	return json.Marshal(ep)
}

// CreateEventOutboxRequest represents the request to create a new event in the outbox
type CreateEventOutboxRequest struct {
	EventType    string          `json:"event_type" validate:"required,max=100"`
	EventKey     string          `json:"event_key" validate:"required,max=255"`
	EventPayload json.RawMessage `json:"event_payload" validate:"required"`
	Topic        string          `json:"topic" validate:"required,max=100"`
}

// IsPending checks if the event is still pending (not yet published)
func (eo *EventOutbox) IsPending() bool {
	return eo.PublishedAt == nil
}

// IsPublished checks if the event has been successfully published
func (eo *EventOutbox) IsPublished() bool {
	return eo.PublishedAt != nil
}

// HasErrors checks if there were errors during publishing attempts
func (eo *EventOutbox) HasErrors() bool {
	return eo.LastError != nil && *eo.LastError != ""
}

// MarkAsPublished sets the published timestamp to now
func (eo *EventOutbox) MarkAsPublished() {
	now := time.Now()
	eo.PublishedAt = &now
}

// RecordError increments retry count and stores the error message
func (eo *EventOutbox) RecordError(errMsg string) {
	eo.RetryCount++
	eo.LastError = &errMsg
}

// ShouldRetry checks if the event should be retried based on retry count
func (eo *EventOutbox) ShouldRetry(maxRetries int) bool {
	return eo.RetryCount < maxRetries && eo.IsPending()
}
