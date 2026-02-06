package models

import "github.com/google/uuid"

// RestockAlert represents a product that has reached or fallen below its low stock threshold
type RestockAlert struct {
	ProductID          uuid.UUID `json:"product_id" db:"product_id"`
	ProductName        string    `json:"product_name" db:"product_name"`
	CategoryName       string    `json:"category_name" db:"category_name"`
	SKU                string    `json:"sku" db:"sku"`
	CurrentStock       int       `json:"current_stock" db:"current_stock"`
	LowStockThreshold  int       `json:"low_stock_threshold" db:"low_stock_threshold"`
	RecommendedReorder int       `json:"recommended_reorder" db:"-"` // Calculated: threshold * 2 - current
	Status             string    `json:"status" db:"-"`              // "critical" (0 stock), "low" (<= threshold)
	SellingPrice       float64   `json:"selling_price" db:"selling_price"`
	CostPrice          float64   `json:"cost_price" db:"cost_price"`
	ImageURL           string    `json:"image_url,omitempty" db:"image_url"`
}

// RestockAlertsResponse represents the response for the restock alerts endpoint
type RestockAlertsResponse struct {
	Count         int            `json:"count"`
	CriticalCount int            `json:"critical_count"`  // Products with 0 stock
	LowStockCount int            `json:"low_stock_count"` // Products below threshold
	RestockAlerts []RestockAlert `json:"restock_alerts"`
}

// IsCritical checks if the product is out of stock
func (r *RestockAlert) IsCritical() bool {
	return r.CurrentStock == 0
}

// IsLowStock checks if the product is below threshold but not out of stock
func (r *RestockAlert) IsLowStock() bool {
	return r.CurrentStock > 0 && r.CurrentStock <= r.LowStockThreshold
}

// CalculateRecommendedReorder calculates the recommended reorder quantity
func (r *RestockAlert) CalculateRecommendedReorder() int {
	reorder := r.LowStockThreshold*2 - r.CurrentStock
	if reorder < 0 {
		return 0
	}
	return reorder
}
