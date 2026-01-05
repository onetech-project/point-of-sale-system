package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// AuditEvent represents a single audit log entry
// Maps to audit_events partitioned table from migration 000031
type AuditEvent struct {
	EventID      uuid.UUID `json:"event_id" db:"event_id"`           // PRIMARY KEY
	TenantID     string    `json:"tenant_id" db:"tenant_id"`         // Multi-tenancy (partitioned by tenant_id + timestamp)
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`         // Event occurrence time (monthly partitioning)
	ActorType    string    `json:"actor_type" db:"actor_type"`       // user, admin, guest, system
	ActorID      *string   `json:"actor_id" db:"actor_id"`           // User ID (NULL for guests/system)
	ActorEmail   *string   `json:"actor_email" db:"actor_email"`     // Encrypted - who performed action
	SessionID    *string   `json:"session_id" db:"session_id"`       // Session tracking
	Action       string    `json:"action" db:"action"`               // CREATE, READ, UPDATE, DELETE, etc.
	ResourceType string    `json:"resource_type" db:"resource_type"` // user, order, consent, etc.
	ResourceID   string    `json:"resource_id" db:"resource_id"`     // Affected resource ID
	IPAddress    *string   `json:"ip_address" db:"ip_address"`       // Encrypted - source IP
	UserAgent    *string   `json:"user_agent" db:"user_agent"`       // Browser/device info
	RequestID    *string   `json:"request_id" db:"request_id"`       // Request correlation ID
	Purpose      *string   `json:"purpose" db:"purpose"`             // Legal basis (UU PDP Article 20)
	BeforeValue  JSONB     `json:"before_value" db:"before_value"`   // Encrypted - state before change (for UPDATE/DELETE)
	AfterValue   JSONB     `json:"after_value" db:"after_value"`     // Encrypted - state after change (for CREATE/UPDATE)
	Metadata     JSONB     `json:"metadata" db:"metadata"`           // Additional context (not encrypted)
	CreatedAt    time.Time `json:"created_at" db:"created_at"`       // Insertion timestamp
}

// TableName returns the table name for AuditEvent
func (AuditEvent) TableName() string {
	return "audit_events"
}

// PartitionName returns the monthly partition name for this event
// Format: audit_events_YYYYMM (e.g., audit_events_202501)
func (a *AuditEvent) PartitionName() string {
	return "audit_events_" + a.Timestamp.Format("200601")
}

// IsDataAccess returns true if this is a data access audit event
func (a *AuditEvent) IsDataAccess() bool {
	return a.Action == "READ" || a.Action == "ACCESS" || a.Action == "EXPORT"
}

// IsDataModification returns true if this event modified data
func (a *AuditEvent) IsDataModification() bool {
	return a.Action == "CREATE" || a.Action == "UPDATE" || a.Action == "DELETE"
}
