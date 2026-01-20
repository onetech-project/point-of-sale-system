package models

import (
	"time"

	"github.com/google/uuid"
)

// ConsentRecord represents user/guest consent for data processing
// Maps to consent_records table from migration 000030
type ConsentRecord struct {
	RecordID      uuid.UUID  `json:"record_id" db:"record_id"`           // PRIMARY KEY
	TenantID      string     `json:"tenant_id" db:"tenant_id"`           // Multi-tenancy
	SubjectType   string     `json:"subject_type" db:"subject_type"`     // "tenant" (user) or "guest"
	SubjectID     *string    `json:"subject_id" db:"subject_id"`         // User ID or guest order ID
	PurposeCode   string     `json:"purpose_code" db:"purpose_code"`     // FK to consent_purposes
	Granted       bool       `json:"granted" db:"granted"`               // Consent status
	PolicyVersion string     `json:"policy_version" db:"policy_version"` // FK to privacy_policies
	ConsentMethod string     `json:"consent_method" db:"consent_method"` // registration, checkout, settings_update
	IPAddress     *string    `json:"ip_address" db:"ip_address"`         // Encrypted - proof of consent (UU PDP Article 20(3))
	UserAgent     *string    `json:"user_agent" db:"user_agent"`         // Browser/device info
	RevokedAt     *time.Time `json:"revoked_at" db:"revoked_at"`         // NULL if consent is active
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`         // Consent grant timestamp
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for ConsentRecord
func (ConsentRecord) TableName() string {
	return "consent_records"
}

// IsActive returns true if consent is granted and not revoked
func (c *ConsentRecord) IsActive() bool {
	return c.Granted && c.RevokedAt == nil
}
