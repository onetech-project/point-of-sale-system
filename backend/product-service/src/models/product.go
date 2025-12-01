package models

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TenantID     uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SKU          string     `json:"sku" db:"sku" validate:"required,min=1,max=50"`
	Name         string     `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description  *string    `json:"description,omitempty" db:"description"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty" db:"category_id"`
	CategoryName *string    `json:"category_name,omitempty" db:"category_name"`
	SellingPrice float64    `json:"selling_price" db:"selling_price" validate:"required,gte=0"`
	CostPrice    float64    `json:"cost_price" db:"cost_price" validate:"required,gte=0"`
	TaxRate      float64    `json:"tax_rate" db:"tax_rate" validate:"gte=0,lte=100"`
	StockQuantity int       `json:"stock_quantity" db:"stock_quantity"`
	PhotoPath    *string    `json:"photo_path,omitempty" db:"photo_path"`
	PhotoSize    *int       `json:"photo_size,omitempty" db:"photo_size"`
	ArchivedAt   *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}
