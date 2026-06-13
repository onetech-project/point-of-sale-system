package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderItemCostSnapshot struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	OrderID                uuid.UUID       `json:"order_id" db:"order_id"`
	OrderItemID            uuid.UUID       `json:"order_item_id" db:"order_item_id"`
	ItemType               string          `json:"item_type" db:"item_type"`
	ProductID              *uuid.UUID      `json:"product_id,omitempty" db:"product_id"`
	BundleID               *uuid.UUID      `json:"bundle_id,omitempty" db:"bundle_id"`
	RecipeID               *uuid.UUID      `json:"recipe_id,omitempty" db:"recipe_id"`
	RecipeVersion          int             `json:"recipe_version" db:"recipe_version"`
	Quantity               decimal.Decimal `json:"-" db:"quantity"`
	MaterialCost           decimal.Decimal `json:"-" db:"material_cost"`
	FixedOverheadCost      decimal.Decimal `json:"-" db:"fixed_overhead_cost"`
	PercentageOverheadCost decimal.Decimal `json:"-" db:"percentage_overhead_cost"`
	TotalCOGS              decimal.Decimal `json:"-" db:"total_cogs"`
	SellingPrice           decimal.Decimal `json:"-" db:"selling_price"`
	GrossProfit            decimal.Decimal `json:"-" db:"gross_profit"`
	GrossMarginPercentage  decimal.Decimal `json:"-" db:"gross_margin_percentage"`
	DiscountAmount         decimal.Decimal `json:"-" db:"discount_amount"`
	PricingSnapshot        json.RawMessage `json:"pricing_snapshot" db:"pricing_snapshot"`
	ComponentCosts         json.RawMessage `json:"component_costs" db:"component_costs"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
}

type OrderItemCostSnapshotResponse struct {
	ID                     uuid.UUID       `json:"id"`
	TenantID               uuid.UUID       `json:"tenant_id"`
	OrderID                uuid.UUID       `json:"order_id"`
	OrderItemID            uuid.UUID       `json:"order_item_id"`
	ItemType               string          `json:"item_type"`
	ProductID              *uuid.UUID      `json:"product_id,omitempty"`
	BundleID               *uuid.UUID      `json:"bundle_id,omitempty"`
	RecipeID               *uuid.UUID      `json:"recipe_id,omitempty"`
	RecipeVersion          int             `json:"recipe_version"`
	Quantity               Number          `json:"quantity"`
	MaterialCost           Number          `json:"material_cost"`
	FixedOverheadCost      Number          `json:"fixed_overhead_cost"`
	PercentageOverheadCost Number          `json:"percentage_overhead_cost"`
	TotalCOGS              Number          `json:"total_cogs"`
	SellingPrice           Number          `json:"selling_price"`
	GrossProfit            Number          `json:"gross_profit"`
	GrossMarginPercentage  Number          `json:"gross_margin_percentage"`
	DiscountAmount         Number          `json:"discount_amount"`
	PricingSnapshot        json.RawMessage `json:"pricing_snapshot"`
	ComponentCosts         json.RawMessage `json:"component_costs"`
	CreatedAt              time.Time       `json:"created_at"`
}

type OrderProfitabilityResponse struct {
	OrderID               uuid.UUID                       `json:"order_id"`
	TotalRevenue          Number                          `json:"total_revenue"`
	TotalCOGS             Number                          `json:"total_cogs"`
	GrossProfit           Number                          `json:"gross_profit"`
	GrossMarginPercentage Number                          `json:"gross_margin_percentage"`
	Items                 []OrderItemCostSnapshotResponse `json:"items"`
}

func NewOrderItemCostSnapshotResponse(snapshot *OrderItemCostSnapshot) OrderItemCostSnapshotResponse {
	pricingSnapshot := snapshot.PricingSnapshot
	if len(pricingSnapshot) == 0 {
		pricingSnapshot = json.RawMessage(`{}`)
	}
	componentCosts := snapshot.ComponentCosts
	if len(componentCosts) == 0 {
		componentCosts = json.RawMessage(`[]`)
	}
	return OrderItemCostSnapshotResponse{
		ID:                     snapshot.ID,
		TenantID:               snapshot.TenantID,
		OrderID:                snapshot.OrderID,
		OrderItemID:            snapshot.OrderItemID,
		ItemType:               snapshot.ItemType,
		ProductID:              snapshot.ProductID,
		BundleID:               snapshot.BundleID,
		RecipeID:               snapshot.RecipeID,
		RecipeVersion:          snapshot.RecipeVersion,
		Quantity:               NewNumber(snapshot.Quantity),
		MaterialCost:           NewNumber(snapshot.MaterialCost),
		FixedOverheadCost:      NewNumber(snapshot.FixedOverheadCost),
		PercentageOverheadCost: NewNumber(snapshot.PercentageOverheadCost),
		TotalCOGS:              NewNumber(snapshot.TotalCOGS),
		SellingPrice:           NewNumber(snapshot.SellingPrice),
		GrossProfit:            NewNumber(snapshot.GrossProfit),
		GrossMarginPercentage:  NewNumber(snapshot.GrossMarginPercentage),
		DiscountAmount:         NewNumber(snapshot.DiscountAmount),
		PricingSnapshot:        pricingSnapshot,
		ComponentCosts:         componentCosts,
		CreatedAt:              snapshot.CreatedAt,
	}
}
