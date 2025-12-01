package models

import (
	"time"

	"github.com/google/uuid"
)

type StockAdjustment struct {
	ID               uuid.UUID `json:"id" db:"id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ProductID        uuid.UUID `json:"product_id" db:"product_id" validate:"required"`
	UserID           uuid.UUID `json:"user_id" db:"user_id" validate:"required"`
	PreviousQuantity int       `json:"previous_quantity" db:"previous_quantity"`
	NewQuantity      int       `json:"new_quantity" db:"new_quantity" validate:"required"`
	QuantityDelta    int       `json:"quantity_delta" db:"quantity_delta"`
	Reason           string    `json:"reason" db:"reason" validate:"required,oneof=supplier_delivery physical_count shrinkage damage return correction sale"`
	Notes            *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
