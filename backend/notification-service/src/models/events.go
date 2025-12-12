package models

import "time"

// OrderPaidEvent represents the event published when an order is paid
type OrderPaidEvent struct {
	EventID   string                 `json:"event_id" validate:"required"`
	EventType string                 `json:"event_type" validate:"required"` // "order.paid"
	TenantID  string                 `json:"tenant_id" validate:"required"`
	Timestamp time.Time              `json:"timestamp" validate:"required"`
	Data      OrderPaidEventMetadata `json:"data" validate:"required"`
}

// OrderPaidEventMetadata contains the order details for the event
type OrderPaidEventMetadata struct {
	OrderID         string      `json:"order_id" validate:"required"`
	OrderReference  string      `json:"order_reference" validate:"required"`
	TransactionID   string      `json:"transaction_id" validate:"required"`
	CustomerName    string      `json:"customer_name" validate:"required"`
	CustomerPhone   string      `json:"customer_phone" validate:"required"`
	CustomerEmail   string      `json:"customer_email,omitempty"`
	DeliveryType    string      `json:"delivery_type" validate:"required"` // "delivery", "pickup", "dine_in"
	DeliveryAddress string      `json:"delivery_address,omitempty"`
	TableNumber     string      `json:"table_number,omitempty"`
	Items           []OrderItem `json:"items" validate:"required,min=1"`
	SubtotalAmount  int         `json:"subtotal_amount" validate:"required,min=0"`
	DeliveryFee     int         `json:"delivery_fee" validate:"min=0"`
	TotalAmount     int         `json:"total_amount" validate:"required,min=0"`
	PaymentMethod   string      `json:"payment_method" validate:"required"`
	PaidAt          time.Time   `json:"paid_at" validate:"required"`
	CreatedAt       time.Time   `json:"created_at" validate:"required"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID   string `json:"product_id" validate:"required"`
	ProductName string `json:"product_name" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`
	UnitPrice   int    `json:"unit_price" validate:"required,min=0"`
	TotalPrice  int    `json:"total_price" validate:"required,min=0"`
}

// Event type constants
const (
	EventTypeOrderPaidStaff    = "order.paid.staff"
	EventTypeOrderPaidCustomer = "order.paid.customer"
)
