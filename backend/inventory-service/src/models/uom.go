package models

import (
	"time"

	"github.com/google/uuid"
)

type UOM struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Code       string     `json:"code" db:"code"`
	Name       string     `json:"name" db:"name"`
	Category   string     `json:"category" db:"category"`
	IsBaseUnit bool       `json:"is_base_unit" db:"is_base_unit"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

type CreateUOMRequest struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	IsBaseUnit bool   `json:"is_base_unit"`
	IsActive   *bool  `json:"is_active,omitempty"`
}

type UpdateUOMRequest struct {
	Name       *string `json:"name,omitempty"`
	Category   *string `json:"category,omitempty"`
	IsBaseUnit *bool   `json:"is_base_unit,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}
