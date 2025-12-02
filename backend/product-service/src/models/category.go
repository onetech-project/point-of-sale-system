package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name         string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
