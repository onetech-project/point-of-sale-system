package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	OrderFulfilledEventType = "order.fulfilled"
	OrderCancelledEventType = "order.cancelled"
	OrderItemTypeProduct    = "product"
	OrderItemTypeBundle     = "bundle"
)

type OrderLifecycleEvent struct {
	EventID        string                    `json:"event_id"`
	EventType      string                    `json:"event_type"`
	TenantID       uuid.UUID                 `json:"tenant_id"`
	OrderID        uuid.UUID                 `json:"order_id"`
	OrderReference string                    `json:"order_reference"`
	OrderType      string                    `json:"order_type"`
	OldStatus      string                    `json:"old_status"`
	NewStatus      string                    `json:"new_status"`
	OccurredAt     time.Time                 `json:"occurred_at"`
	Items          []OrderLifecycleItemEvent `json:"items"`
}

type OrderLifecycleItemEvent struct {
	OrderItemID     uuid.UUID       `json:"order_item_id"`
	ItemType        string          `json:"item_type"`
	ProductID       *uuid.UUID      `json:"product_id,omitempty"`
	BundleID        *uuid.UUID      `json:"bundle_id,omitempty"`
	ProductName     string          `json:"product_name"`
	Quantity        int             `json:"quantity"`
	ListUnitPrice   int             `json:"list_unit_price"`
	UnitPrice       int             `json:"unit_price"`
	TotalPrice      int             `json:"total_price"`
	DiscountRuleID  *uuid.UUID      `json:"discount_rule_id,omitempty"`
	DiscountType    *string         `json:"discount_type,omitempty"`
	DiscountValue   *float64        `json:"discount_value,omitempty"`
	DiscountAmount  int             `json:"discount_amount"`
	PricingSnapshot json.RawMessage `json:"pricing_snapshot,omitempty"`
}
