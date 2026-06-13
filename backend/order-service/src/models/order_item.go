package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// OrderItem represents a line item in a guest order
type OrderItem struct {
	ID              string          `json:"id"`
	OrderID         string          `json:"order_id"`
	ProductID       string          `json:"product_id,omitempty"`
	ProductName     string          `json:"product_name"`
	ProductSKU      *string         `json:"product_sku,omitempty"`
	Quantity        int             `json:"quantity"`
	UnitPrice       int             `json:"unit_price"`  // Discounted final price at time of order (IDR)
	TotalPrice      int             `json:"total_price"` // quantity * unit_price
	ItemType        string          `json:"item_type,omitempty"`
	BundleID        *string         `json:"bundle_id,omitempty"`
	ListUnitPrice   int             `json:"list_unit_price,omitempty"`
	DiscountRuleID  *string         `json:"discount_rule_id,omitempty"`
	DiscountType    *DiscountType   `json:"discount_type,omitempty"`
	DiscountValue   *float64        `json:"discount_value,omitempty"`
	DiscountAmount  int             `json:"discount_amount,omitempty"`
	PricingSnapshot json.RawMessage `json:"pricing_snapshot,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
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
