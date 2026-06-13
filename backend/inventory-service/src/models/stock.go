package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	MovementInitialStock     = "INITIAL_STOCK"
	MovementPurchaseIn       = "PURCHASE_IN"
	MovementManualAdjustment = "MANUAL_ADJUSTMENT"
	MovementSaleConsumption  = "SALE_CONSUMPTION"
	MovementWaste            = "WASTE"
)

type StockMovement struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TenantID       uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	OutletID       *uuid.UUID      `json:"outlet_id,omitempty" db:"outlet_id"`
	IngredientID   uuid.UUID       `json:"ingredient_id" db:"ingredient_id"`
	MovementType   string          `json:"movement_type" db:"movement_type"`
	QuantityBase   decimal.Decimal `json:"-" db:"quantity_base"`
	UnitCost       decimal.Decimal `json:"-" db:"unit_cost"`
	TotalCost      decimal.Decimal `json:"-" db:"total_cost"`
	ReferenceType  *string         `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID    *string         `json:"reference_id,omitempty" db:"reference_id"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty" db:"idempotency_key"`
	Reason         *string         `json:"reason,omitempty" db:"reason"`
	OccurredAt     time.Time       `json:"occurred_at" db:"occurred_at"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

type StockMovementResponse struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	OutletID       *uuid.UUID `json:"outlet_id,omitempty"`
	IngredientID   uuid.UUID  `json:"ingredient_id"`
	MovementType   string     `json:"movement_type"`
	QuantityBase   Number     `json:"quantity_base"`
	UnitCost       Number     `json:"unit_cost"`
	TotalCost      Number     `json:"total_cost"`
	ReferenceType  *string    `json:"reference_type,omitempty"`
	ReferenceID    *string    `json:"reference_id,omitempty"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	OccurredAt     time.Time  `json:"occurred_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func NewStockMovementResponse(m *StockMovement) StockMovementResponse {
	return StockMovementResponse{
		ID:             m.ID,
		TenantID:       m.TenantID,
		OutletID:       m.OutletID,
		IngredientID:   m.IngredientID,
		MovementType:   m.MovementType,
		QuantityBase:   NewNumber(m.QuantityBase),
		UnitCost:       NewNumber(m.UnitCost),
		TotalCost:      NewNumber(m.TotalCost),
		ReferenceType:  m.ReferenceType,
		ReferenceID:    m.ReferenceID,
		IdempotencyKey: m.IdempotencyKey,
		Reason:         m.Reason,
		OccurredAt:     m.OccurredAt,
		CreatedBy:      m.CreatedBy,
		CreatedAt:      m.CreatedAt,
	}
}

type StockMutationRequest struct {
	IngredientID   uuid.UUID  `json:"ingredient_id"`
	OutletID       *uuid.UUID `json:"outlet_id,omitempty"`
	Quantity       Number     `json:"quantity"`
	UOMID          uuid.UUID  `json:"uom_id"`
	UnitCost       Number     `json:"unit_cost"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	ReferenceType  *string    `json:"reference_type,omitempty"`
	ReferenceID    *string    `json:"reference_id,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
}

type ManualAdjustmentRequest struct {
	IngredientID   uuid.UUID  `json:"ingredient_id"`
	OutletID       *uuid.UUID `json:"outlet_id,omitempty"`
	QuantityDelta  Number     `json:"quantity_delta"`
	UOMID          uuid.UUID  `json:"uom_id"`
	UnitCost       Number     `json:"unit_cost"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	Reason         string     `json:"reason"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
}
