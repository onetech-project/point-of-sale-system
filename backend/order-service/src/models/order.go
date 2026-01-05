package models

import (
	"fmt"
	"time"
)

// OrderStatus represents the lifecycle of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusComplete  OrderStatus = "COMPLETE"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// DeliveryType represents how the order will be fulfilled
type DeliveryType string

const (
	DeliveryTypePickup   DeliveryType = "pickup"
	DeliveryTypeDelivery DeliveryType = "delivery"
	DeliveryTypeDineIn   DeliveryType = "dine_in"
)

// GuestOrder represents an order placed by an unauthenticated guest
type GuestOrder struct {
	ID             string       `json:"id"`
	OrderReference string       `json:"order_reference"`
	TenantID       string       `json:"tenant_id"`
	Status         OrderStatus  `json:"status"`
	SubtotalAmount int          `json:"subtotal_amount"` // In smallest currency unit (IDR cents)
	DeliveryFee    int          `json:"delivery_fee"`
	TotalAmount    int          `json:"total_amount"`
	CustomerName   string       `json:"customer_name"`
	CustomerPhone  string       `json:"customer_phone"`
	CustomerEmail  *string      `json:"customer_email,omitempty"`
	DeliveryType   DeliveryType `json:"delivery_type"`
	TableNumber    *string      `json:"table_number,omitempty"`
	Notes          *string      `json:"notes,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	PaidAt         *time.Time   `json:"paid_at,omitempty"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	CancelledAt    *time.Time   `json:"cancelled_at,omitempty"`
	SessionID      string       `json:"session_id,omitempty"`
	IPAddress      *string      `json:"ip_address,omitempty"`
	UserAgent      *string      `json:"user_agent,omitempty"`
	IsAnonymized   bool         `json:"is_anonymized"`
	AnonymizedAt   *time.Time   `json:"anonymized_at,omitempty"`
}

// CreateOrderRequest represents the request to create a new order
type CreateOrderRequest struct {
	TenantID      string               `json:"tenant_id" validate:"required,uuid"`
	CustomerName  string               `json:"customer_name" validate:"required,min=2,max=255"`
	CustomerPhone string               `json:"customer_phone" validate:"required,e164"`
	DeliveryType  DeliveryType         `json:"delivery_type" validate:"required,oneof=pickup delivery dine_in"`
	TableNumber   *string              `json:"table_number,omitempty"`
	Notes         *string              `json:"notes,omitempty"`
	Items         []CreateOrderItemReq `json:"items" validate:"required,min=1,dive"`
	DeliveryAddr  *DeliveryAddressReq  `json:"delivery_address,omitempty"`
	SessionID     string               `json:"session_id" validate:"required"`
}

// CreateOrderItemReq represents an item in the create order request
type CreateOrderItemReq struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

// DeliveryAddressReq represents delivery address in the create order request
type DeliveryAddressReq struct {
	AddressText string `json:"address_text" validate:"required,min=10"`
}

// OrderResponse represents the order response with related entities
type OrderResponse struct {
	*GuestOrder
	Items           []OrderItem      `json:"items"`
	DeliveryAddress *DeliveryAddress `json:"delivery_address,omitempty"`
	PaymentInfo     *PaymentInfo     `json:"payment_info,omitempty"`
}

// PaymentInfo represents simplified payment information for order response
type PaymentInfo struct {
	Status        string     `json:"status"`
	PaymentType   *string    `json:"payment_type,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	TransactionID *string    `json:"transaction_id,omitempty"`
}

// Scan implements sql.Scanner for OrderStatus
func (s *OrderStatus) Scan(value interface{}) error {
	if value == nil {
		*s = OrderStatusPending
		return nil
	}
	*s = OrderStatus(value.(string))
	return nil
}

// Scan implements sql.Scanner for DeliveryType
func (d *DeliveryType) Scan(value interface{}) error {
	if value == nil {
		*d = DeliveryTypePickup
		return nil
	}
	*d = DeliveryType(value.(string))
	return nil
}

// ValidateStatusTransition checks if a status transition is allowed
func (o *GuestOrder) ValidateStatusTransition(newStatus OrderStatus) error {
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
		OrderStatusPaid:      {OrderStatusComplete, OrderStatusCancelled},
		OrderStatusComplete:  {}, // Terminal state
		OrderStatusCancelled: {}, // Terminal state
	}

	allowed := transitions[o.Status]
	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition: %s -> %s", o.Status, newStatus)
}

// IsTerminalStatus checks if the order is in a terminal state
func (o *GuestOrder) IsTerminalStatus() bool {
	return o.Status == OrderStatusComplete || o.Status == OrderStatusCancelled
}

// RequiresPayment checks if the order requires payment
func (o *GuestOrder) RequiresPayment() bool {
	return o.Status == OrderStatusPending
}
