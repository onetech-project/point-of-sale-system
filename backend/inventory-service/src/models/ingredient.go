package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Ingredient struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	OutletID               *uuid.UUID      `json:"outlet_id,omitempty" db:"outlet_id"`
	SKU                    string          `json:"sku" db:"sku"`
	Name                   string          `json:"name" db:"name"`
	BaseUOMID              uuid.UUID       `json:"base_uom_id" db:"base_uom_id"`
	BaseUOMCode            *string         `json:"base_uom_code,omitempty" db:"base_uom_code"`
	CurrentStockBase       decimal.Decimal `json:"-" db:"current_stock_base"`
	AverageCostPerBaseUnit decimal.Decimal `json:"-" db:"average_cost_per_base_unit"`
	MinimumStockBase       decimal.Decimal `json:"-" db:"minimum_stock_base"`
	TrackStock             bool            `json:"track_stock" db:"track_stock"`
	IsActive               bool            `json:"is_active" db:"is_active"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
}

type IngredientResponse struct {
	ID                     uuid.UUID  `json:"id"`
	TenantID               uuid.UUID  `json:"tenant_id"`
	OutletID               *uuid.UUID `json:"outlet_id,omitempty"`
	SKU                    string     `json:"sku"`
	Name                   string     `json:"name"`
	BaseUOMID              uuid.UUID  `json:"base_uom_id"`
	BaseUOMCode            *string    `json:"base_uom_code,omitempty"`
	CurrentStockBase       Number     `json:"current_stock_base"`
	AverageCostPerBaseUnit Number     `json:"average_cost_per_base_unit"`
	MinimumStockBase       Number     `json:"minimum_stock_base"`
	TrackStock             bool       `json:"track_stock"`
	IsActive               bool       `json:"is_active"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func NewIngredientResponse(i *Ingredient) IngredientResponse {
	return IngredientResponse{
		ID:                     i.ID,
		TenantID:               i.TenantID,
		OutletID:               i.OutletID,
		SKU:                    i.SKU,
		Name:                   i.Name,
		BaseUOMID:              i.BaseUOMID,
		BaseUOMCode:            i.BaseUOMCode,
		CurrentStockBase:       NewNumber(i.CurrentStockBase),
		AverageCostPerBaseUnit: NewNumber(i.AverageCostPerBaseUnit),
		MinimumStockBase:       NewNumber(i.MinimumStockBase),
		TrackStock:             i.TrackStock,
		IsActive:               i.IsActive,
		CreatedAt:              i.CreatedAt,
		UpdatedAt:              i.UpdatedAt,
	}
}

type CreateIngredientRequest struct {
	OutletID               *uuid.UUID `json:"outlet_id,omitempty"`
	SKU                    string     `json:"sku"`
	Name                   string     `json:"name"`
	BaseUOMID              uuid.UUID  `json:"base_uom_id"`
	MinimumStockBase       Number     `json:"minimum_stock_base"`
	AverageCostPerBaseUnit Number     `json:"average_cost_per_base_unit"`
	TrackStock             *bool      `json:"track_stock,omitempty"`
	IsActive               *bool      `json:"is_active,omitempty"`
}

type UpdateIngredientRequest struct {
	OutletID               *uuid.UUID `json:"outlet_id,omitempty"`
	SKU                    *string    `json:"sku,omitempty"`
	Name                   *string    `json:"name,omitempty"`
	BaseUOMID              *uuid.UUID `json:"base_uom_id,omitempty"`
	MinimumStockBase       *Number    `json:"minimum_stock_base,omitempty"`
	AverageCostPerBaseUnit *Number    `json:"average_cost_per_base_unit,omitempty"`
	TrackStock             *bool      `json:"track_stock,omitempty"`
	IsActive               *bool      `json:"is_active,omitempty"`
}
