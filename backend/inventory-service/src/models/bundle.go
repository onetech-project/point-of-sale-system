package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Bundle struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	TenantID     uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	SKU          string          `json:"sku" db:"sku"`
	Name         string          `json:"name" db:"name"`
	Description  *string         `json:"description,omitempty" db:"description"`
	SellingPrice decimal.Decimal `json:"-" db:"selling_price"`
	IsActive     bool            `json:"is_active" db:"is_active"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

type BundleItem struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	BundleID    uuid.UUID       `json:"bundle_id" db:"bundle_id"`
	ProductID   uuid.UUID       `json:"product_id" db:"product_id"`
	ProductName string          `json:"product_name,omitempty"`
	ProductSKU  string          `json:"product_sku,omitempty"`
	Quantity    decimal.Decimal `json:"-" db:"quantity"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

type BundleAggregate struct {
	Bundle Bundle
	Items  []BundleItem
}

type CreateBundleRequest struct {
	SKU          string             `json:"sku"`
	Name         string             `json:"name"`
	Description  *string            `json:"description,omitempty"`
	SellingPrice Number             `json:"selling_price"`
	IsActive     *bool              `json:"is_active,omitempty"`
	Items        []CreateBundleItem `json:"items"`
}

type CreateBundleItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  Number    `json:"quantity"`
}

type UpdateBundleRequest struct {
	SKU          *string             `json:"sku,omitempty"`
	Name         *string             `json:"name,omitempty"`
	Description  *string             `json:"description,omitempty"`
	SellingPrice *Number             `json:"selling_price,omitempty"`
	IsActive     *bool               `json:"is_active,omitempty"`
	Items        *[]CreateBundleItem `json:"items,omitempty"`
}

type BundleItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name,omitempty"`
	ProductSKU  string    `json:"product_sku,omitempty"`
	Quantity    Number    `json:"quantity"`
}

type BundleResponse struct {
	ID           uuid.UUID            `json:"id"`
	TenantID     uuid.UUID            `json:"tenant_id"`
	SKU          string               `json:"sku"`
	Name         string               `json:"name"`
	Description  *string              `json:"description,omitempty"`
	SellingPrice Number               `json:"selling_price"`
	IsActive     bool                 `json:"is_active"`
	Items        []BundleItemResponse `json:"items"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

type BundleItemCostResponse struct {
	BundleItemID               uuid.UUID `json:"bundle_item_id"`
	ProductID                  uuid.UUID `json:"product_id"`
	ProductName                string    `json:"product_name"`
	ProductSKU                 string    `json:"product_sku"`
	Quantity                   Number    `json:"quantity"`
	RecipeID                   uuid.UUID `json:"recipe_id"`
	RecipeVersion              int       `json:"recipe_version"`
	UnitMaterialCost           Number    `json:"unit_material_cost"`
	UnitFixedOverheadCost      Number    `json:"unit_fixed_overhead_cost"`
	UnitPercentageOverheadCost Number    `json:"unit_percentage_overhead_cost"`
	UnitTotalCOGS              Number    `json:"unit_total_cogs"`
	LineMaterialCost           Number    `json:"line_material_cost"`
	LineFixedOverheadCost      Number    `json:"line_fixed_overhead_cost"`
	LinePercentageOverheadCost Number    `json:"line_percentage_overhead_cost"`
	LineTotalCOGS              Number    `json:"line_total_cogs"`
}

type BundleCostResponse struct {
	BundleID               uuid.UUID                `json:"bundle_id"`
	BundleName             string                   `json:"bundle_name"`
	BundleSKU              string                   `json:"bundle_sku"`
	Items                  []BundleItemCostResponse `json:"items"`
	MaterialCost           Number                   `json:"material_cost"`
	FixedOverheadCost      Number                   `json:"fixed_overhead_cost"`
	PercentageOverheadCost Number                   `json:"percentage_overhead_cost"`
	TotalCOGS              Number                   `json:"total_cogs"`
	SellingPrice           Number                   `json:"selling_price"`
	GrossProfit            Number                   `json:"gross_profit"`
	GrossMarginPercentage  Number                   `json:"gross_margin_percentage"`
}

func NewBundleResponse(aggregate *BundleAggregate) BundleResponse {
	items := make([]BundleItemResponse, 0, len(aggregate.Items))
	for i := range aggregate.Items {
		item := aggregate.Items[i]
		items = append(items, BundleItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductSKU:  item.ProductSKU,
			Quantity:    NewNumber(item.Quantity),
		})
	}
	return BundleResponse{
		ID:           aggregate.Bundle.ID,
		TenantID:     aggregate.Bundle.TenantID,
		SKU:          aggregate.Bundle.SKU,
		Name:         aggregate.Bundle.Name,
		Description:  aggregate.Bundle.Description,
		SellingPrice: NewNumber(aggregate.Bundle.SellingPrice),
		IsActive:     aggregate.Bundle.IsActive,
		Items:        items,
		CreatedAt:    aggregate.Bundle.CreatedAt,
		UpdatedAt:    aggregate.Bundle.UpdatedAt,
	}
}
