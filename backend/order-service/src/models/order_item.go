package models

import (
	"fmt"
	"time"
)

// OrderItem represents a line item in a guest order
type OrderItem struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	ProductSKU  *string   `json:"product_sku,omitempty"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int       `json:"unit_price"`  // Price at time of order (IDR cents)
	TotalPrice  int       `json:"total_price"` // quantity * unit_price
	CreatedAt   time.Time `json:"created_at"`
}

// Validate checks if the order item is valid
func (oi *OrderItem) Validate() error {
	if oi.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if oi.UnitPrice < 0 {
		return fmt.Errorf("unit_price cannot be negative")
	}
	if oi.TotalPrice != oi.Quantity*oi.UnitPrice {
		return fmt.Errorf("total_price must equal quantity * unit_price")
	}
	return nil
}
