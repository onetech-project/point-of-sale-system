package models

import (
	"time"
)

// ConsentPurpose represents a legal purpose for collecting/processing personal data
// Maps to consent_purposes table from migration 000028
type ConsentPurpose struct {
	PurposeCode   string    `json:"purpose_code" db:"purpose_code"`       // PRIMARY KEY: service_operation, transactional_communication, etc.
	DisplayNameID string    `json:"display_name_id" db:"display_name_id"` // i18n key for purpose name
	DescriptionID string    `json:"description_id" db:"description_id"`   // i18n key for detailed explanation
	IsRequired    bool      `json:"is_required" db:"is_required"`         // Whether consent is mandatory (UU PDP Article 20)
	DisplayOrder  int       `json:"display_order" db:"display_order"`     // UI ordering
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for ConsentPurpose
func (ConsentPurpose) TableName() string {
	return "consent_purposes"
}
