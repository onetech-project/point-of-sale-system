package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	OverheadFixed      = "FIXED"
	OverheadPercentage = "PERCENTAGE"
)

type ProductSnapshot struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	SKU          string          `json:"sku"`
	SellingPrice decimal.Decimal `json:"-"`
}

type Recipe struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	ProductID     uuid.UUID       `json:"product_id" db:"product_id"`
	Version       int             `json:"version" db:"version"`
	YieldQuantity decimal.Decimal `json:"-" db:"yield_quantity"`
	IsActive      bool            `json:"is_active" db:"is_active"`
	EffectiveFrom time.Time       `json:"effective_from" db:"effective_from"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type RecipeItem struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	RecipeID               uuid.UUID       `json:"recipe_id" db:"recipe_id"`
	IngredientID           uuid.UUID       `json:"ingredient_id" db:"ingredient_id"`
	IngredientName         string          `json:"ingredient_name,omitempty"`
	Quantity               decimal.Decimal `json:"-" db:"quantity"`
	UOMID                  uuid.UUID       `json:"uom_id" db:"uom_id"`
	UOMCode                string          `json:"uom_code,omitempty"`
	NormalizedQuantityBase decimal.Decimal `json:"-" db:"normalized_quantity_base"`
	WastePercentage        decimal.Decimal `json:"-" db:"waste_percentage"`
	AverageCostPerBaseUnit decimal.Decimal `json:"-"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
}

type RecipeOverhead struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	RecipeID        uuid.UUID       `json:"recipe_id" db:"recipe_id"`
	Name            string          `json:"name" db:"name"`
	CalculationType string          `json:"calculation_type" db:"calculation_type"`
	Amount          decimal.Decimal `json:"-" db:"amount"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

type RecipeAggregate struct {
	Recipe    Recipe
	Product   ProductSnapshot
	Items     []RecipeItem
	Overheads []RecipeOverhead
}

type CreateRecipeRequest struct {
	YieldQuantity Number                 `json:"yield_quantity"`
	EffectiveFrom *time.Time             `json:"effective_from,omitempty"`
	Items         []CreateRecipeItem     `json:"items"`
	Overheads     []CreateRecipeOverhead `json:"overheads"`
}

type CreateRecipeItem struct {
	IngredientID    uuid.UUID `json:"ingredient_id"`
	Quantity        Number    `json:"quantity"`
	UOMID           uuid.UUID `json:"uom_id"`
	WastePercentage Number    `json:"waste_percentage"`
}

type CreateRecipeOverhead struct {
	Name            string `json:"name"`
	CalculationType string `json:"calculation_type"`
	Amount          Number `json:"amount"`
}

type RecipeItemResponse struct {
	ID                     uuid.UUID `json:"id"`
	IngredientID           uuid.UUID `json:"ingredient_id"`
	IngredientName         string    `json:"ingredient_name,omitempty"`
	Quantity               Number    `json:"quantity"`
	UOMID                  uuid.UUID `json:"uom_id"`
	UOMCode                string    `json:"uom_code,omitempty"`
	NormalizedQuantityBase Number    `json:"normalized_quantity_base"`
	WastePercentage        Number    `json:"waste_percentage"`
	IngredientCost         Number    `json:"ingredient_cost"`
	CostWithWaste          Number    `json:"cost_with_waste"`
}

type RecipeOverheadResponse struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	CalculationType string    `json:"calculation_type"`
	Amount          Number    `json:"amount"`
}

type RecipeCostResponse struct {
	MaterialCost           Number `json:"material_cost"`
	FixedOverheadCost      Number `json:"fixed_overhead_cost"`
	PercentageOverheadCost Number `json:"percentage_overhead_cost"`
	TotalCOGS              Number `json:"total_cogs"`
	SellingPrice           Number `json:"selling_price"`
	GrossProfit            Number `json:"gross_profit"`
	GrossMarginPercentage  Number `json:"gross_margin_percentage"`
}

type RecipeResponse struct {
	ID            uuid.UUID                `json:"id"`
	TenantID      uuid.UUID                `json:"tenant_id"`
	ProductID     uuid.UUID                `json:"product_id"`
	ProductName   string                   `json:"product_name"`
	ProductSKU    string                   `json:"product_sku"`
	Version       int                      `json:"version"`
	YieldQuantity Number                   `json:"yield_quantity"`
	IsActive      bool                     `json:"is_active"`
	EffectiveFrom time.Time                `json:"effective_from"`
	Items         []RecipeItemResponse     `json:"items"`
	Overheads     []RecipeOverheadResponse `json:"overheads"`
	Cost          RecipeCostResponse       `json:"cost"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

type RecipeVersionSummary struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"product_id"`
	Version       int       `json:"version"`
	YieldQuantity Number    `json:"yield_quantity"`
	IsActive      bool      `json:"is_active"`
	EffectiveFrom time.Time `json:"effective_from"`
	CreatedAt     time.Time `json:"created_at"`
}
