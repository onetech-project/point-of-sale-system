package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type IngredientUOMConversion struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	TenantID     *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	IngredientID *uuid.UUID      `json:"ingredient_id,omitempty" db:"ingredient_id"`
	FromUOMID    uuid.UUID       `json:"from_uom_id" db:"from_uom_id"`
	ToUOMID      uuid.UUID       `json:"to_uom_id" db:"to_uom_id"`
	Multiplier   decimal.Decimal `json:"-" db:"multiplier"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

type ConversionResponse struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	IngredientID *uuid.UUID `json:"ingredient_id,omitempty"`
	FromUOMID    uuid.UUID  `json:"from_uom_id"`
	ToUOMID      uuid.UUID  `json:"to_uom_id"`
	Multiplier   Number     `json:"multiplier"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func NewConversionResponse(c *IngredientUOMConversion) ConversionResponse {
	return ConversionResponse{
		ID:           c.ID,
		TenantID:     c.TenantID,
		IngredientID: c.IngredientID,
		FromUOMID:    c.FromUOMID,
		ToUOMID:      c.ToUOMID,
		Multiplier:   NewNumber(c.Multiplier),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

type CreateConversionRequest struct {
	FromUOMID  uuid.UUID `json:"from_uom_id"`
	ToUOMID    uuid.UUID `json:"to_uom_id"`
	Multiplier Number    `json:"multiplier"`
}

type UpdateConversionRequest struct {
	FromUOMID  *uuid.UUID `json:"from_uom_id,omitempty"`
	ToUOMID    *uuid.UUID `json:"to_uom_id,omitempty"`
	Multiplier *Number    `json:"multiplier,omitempty"`
}
