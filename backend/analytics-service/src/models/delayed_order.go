package models

import (
	"time"

	"github.com/google/uuid"
)

// DelayedOrder represents an order that has exceeded the expected processing time (>15 minutes)
type DelayedOrder struct {
	OrderID        uuid.UUID `json:"order_id" db:"order_id"`
	OrderNumber    string    `json:"order_number" db:"order_number"`
	CustomerID     int64     `json:"customer_id" db:"customer_id"`
	CustomerPhone  string    `json:"customer_phone" db:"customer_phone"` // Encrypted phone
	CustomerName   string    `json:"customer_name" db:"customer_name"`   // Encrypted name
	CustomerEmail  string    `json:"customer_email" db:"customer_email"` // Encrypted email
	MaskedPhone    string    `json:"masked_phone" db:"-"`                // Masked for display (last 4 digits)
	MaskedName     string    `json:"masked_name" db:"-"`                 // Masked for display (first char)
	MaskedEmail    string    `json:"masked_email" db:"-"`                // Masked for display (first char + domain)
	TotalAmount    float64   `json:"total_amount" db:"total_amount"`
	Status         string    `json:"status" db:"status"` // e.g., "pending", "processing"
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	ElapsedMinutes int       `json:"elapsed_minutes" db:"elapsed_minutes"` // Minutes since order creation
}

// DelayedOrdersResponse represents the response for the delayed orders endpoint
type DelayedOrdersResponse struct {
	Count         int            `json:"count"`
	UrgentCount   int            `json:"urgent_count"`  // Orders > 30 minutes
	WarningCount  int            `json:"warning_count"` // Orders 15-30 minutes
	DelayedOrders []DelayedOrder `json:"delayed_orders"`
}

// IsUrgent checks if the order is critically delayed (>30 minutes)
func (d *DelayedOrder) IsUrgent() bool {
	return d.ElapsedMinutes > 30
}

// IsWarning checks if the order is in warning state (15-30 minutes)
func (d *DelayedOrder) IsWarning() bool {
	return d.ElapsedMinutes >= 15 && d.ElapsedMinutes <= 30
}
