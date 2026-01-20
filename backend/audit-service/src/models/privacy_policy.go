package models

import (
	"time"
)

// PrivacyPolicy represents a versioned privacy policy document
// Maps to privacy_policies table from migration 000029
type PrivacyPolicy struct {
	Version       string    `json:"version" db:"version"`               // PRIMARY KEY: v1, v2, etc.
	PolicyTextID  string    `json:"policy_text_id" db:"policy_text_id"` // i18n key for policy content
	EffectiveDate time.Time `json:"effective_date" db:"effective_date"` // When policy takes effect
	IsCurrent     bool      `json:"is_current" db:"is_current"`         // Only one current policy at a time
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for PrivacyPolicy
func (PrivacyPolicy) TableName() string {
	return "privacy_policies"
}
