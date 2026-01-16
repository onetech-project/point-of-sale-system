package events

import (
	"time"
)

// ConsentGrantedEvent represents a consent granted event published to Kafka
// This event is published AFTER user/order creation to ensure proper subject_id
// Only granted optional consents are included in the event
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

// ConsentRevokedEvent represents a consent revoked event published to Kafka
// Published when user revokes optional consent (analytics, advertising)
type ConsentRevokedEvent struct {
	EventID      string    `json:"event_id"`      // Idempotency key (UUID)
	EventType    string    `json:"event_type"`    // "consent.revoked"
	TenantID     string    `json:"tenant_id"`     // Tenant UUID
	SubjectType  string    `json:"subject_type"`  // "tenant" or "guest"
	SubjectID    string    `json:"subject_id"`    // user_id OR order_id
	PurposeCode  string    `json:"purpose_code"`  // Revoked consent code (e.g., "analytics")
	PurposeName  string    `json:"purpose_name"`  // Human-readable purpose name
	RevokedAt    time.Time `json:"revoked_at"`    // Timestamp of revocation
	IPAddress    string    `json:"ip_address"`    // From request (encrypted)
	UserAgent    string    `json:"user_agent"`    // From request headers
	Timestamp    time.Time `json:"timestamp"`     // Event creation time
	ComplianceTag string   `json:"compliance_tag"` // "UU_PDP_Article_21" (right to revoke consent)
}
