package events

import (
	"time"
)

// ConsentGrantedEvent represents a consent granted event published to Kafka
// This event is published AFTER user creation to ensure proper subject_id
// Used for invitation acceptance consent recording
type ConsentGrantedEvent struct {
	EventID          string          `json:"event_id"`          // Idempotency key (UUID)
	EventType        string          `json:"event_type"`        // "consent.granted"
	TenantID         string          `json:"tenant_id"`         // Tenant UUID
	SubjectType      string          `json:"subject_type"`      // "tenant" or "guest"
	SubjectID        string          `json:"subject_id"`        // user_id OR order_id (proper UUID from database)
	ConsentMethod    string          `json:"consent_method"`    // "registration" or "checkout"
	PolicyVersion    string          `json:"policy_version"`    // "1.0.0"
	Consents         []string        `json:"consents"`          // Array of granted consent codes (optional only)
	RequiredConsents []string        `json:"required_consents"` // Array of required consent codes (implicit)
	Metadata         ConsentMetadata `json:"metadata"`          // IP, user agent, timestamp
	Timestamp        time.Time       `json:"timestamp"`         // Event creation time
}

// ConsentMetadata contains request metadata for audit purposes
type ConsentMetadata struct {
	IPAddress string  `json:"ip_address"` // From request
	UserAgent string  `json:"user_agent"` // From request headers
	SessionID *string `json:"session_id"` // Optional
	RequestID string  `json:"request_id"` // Distributed tracing
}
