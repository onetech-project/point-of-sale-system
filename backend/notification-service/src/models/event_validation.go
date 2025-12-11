package models

import (
	"encoding/json"
	"fmt"
)

// ValidateOrderPaidEvent validates an OrderPaidEvent against required fields
func ValidateOrderPaidEvent(event *OrderPaidEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	if event.EventType != "order.paid" {
		return fmt.Errorf("invalid event_type: expected 'order.paid', got '%s'", event.EventType)
	}

	if event.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}

	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate metadata
	if err := ValidateOrderPaidEventMetadata(&event.Metadata); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	return nil
}

// ValidateOrderPaidEventMetadata validates the metadata of an OrderPaidEvent
func ValidateOrderPaidEventMetadata(metadata *OrderPaidEventMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if metadata.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}

	if metadata.OrderReference == "" {
		return fmt.Errorf("order_reference is required")
	}

	if metadata.TransactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}

	if metadata.CustomerName == "" {
		return fmt.Errorf("customer_name is required")
	}

	if metadata.CustomerPhone == "" {
		return fmt.Errorf("customer_phone is required")
	}

	if metadata.DeliveryType == "" {
		return fmt.Errorf("delivery_type is required")
	}

	validDeliveryTypes := map[string]bool{
		"delivery": true,
		"pickup":   true,
		"dine_in":  true,
	}
	if !validDeliveryTypes[metadata.DeliveryType] {
		return fmt.Errorf("invalid delivery_type: %s", metadata.DeliveryType)
	}

	if len(metadata.Items) == 0 {
		return fmt.Errorf("items are required (minimum 1)")
	}

	// Validate items
	for i, item := range metadata.Items {
		if err := ValidateOrderItem(&item); err != nil {
			return fmt.Errorf("item[%d] validation failed: %w", i, err)
		}
	}

	if metadata.SubtotalAmount < 0 {
		return fmt.Errorf("subtotal_amount must be >= 0")
	}

	if metadata.DeliveryFee < 0 {
		return fmt.Errorf("delivery_fee must be >= 0")
	}

	if metadata.TotalAmount < 0 {
		return fmt.Errorf("total_amount must be >= 0")
	}

	if metadata.PaymentMethod == "" {
		return fmt.Errorf("payment_method is required")
	}

	if metadata.PaidAt.IsZero() {
		return fmt.Errorf("paid_at is required")
	}

	return nil
}

// ValidateOrderItem validates an order item
func ValidateOrderItem(item *OrderItem) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	if item.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}

	if item.ProductName == "" {
		return fmt.Errorf("product_name is required")
	}

	if item.Quantity < 1 {
		return fmt.Errorf("quantity must be >= 1")
	}

	if item.UnitPrice < 0 {
		return fmt.Errorf("unit_price must be >= 0")
	}

	if item.TotalPrice < 0 {
		return fmt.Errorf("total_price must be >= 0")
	}

	return nil
}

// ParseOrderPaidEvent parses a JSON byte array into an OrderPaidEvent and validates it
func ParseOrderPaidEvent(data []byte) (*OrderPaidEvent, error) {
	var event OrderPaidEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	if err := ValidateOrderPaidEvent(&event); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}

	return &event, nil
}
