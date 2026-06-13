package models

import (
	"encoding/json"
	"time"
)

type DiscountTargetType string

const (
	DiscountTargetProduct DiscountTargetType = "product"
	DiscountTargetBundle  DiscountTargetType = "bundle"
)

type DiscountType string

const (
	DiscountTypePercentage  DiscountType = "percentage"
	DiscountTypeFixedAmount DiscountType = "fixed_amount"
)

type DiscountRule struct {
	ID            string             `json:"id"`
	TenantID      string             `json:"tenant_id"`
	Name          string             `json:"name"`
	Description   *string            `json:"description,omitempty"`
	TargetType    DiscountTargetType `json:"target_type"`
	TargetID      string             `json:"target_id"`
	DiscountType  DiscountType       `json:"discount_type"`
	DiscountValue float64            `json:"discount_value"`
	StartsAt      *time.Time         `json:"starts_at,omitempty"`
	EndsAt        *time.Time         `json:"ends_at,omitempty"`
	IsActive      bool               `json:"is_active"`
	Exclusive     bool               `json:"exclusive"`
	Priority      int                `json:"priority"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type CreateDiscountRuleRequest struct {
	Name          string             `json:"name"`
	Description   *string            `json:"description,omitempty"`
	TargetType    DiscountTargetType `json:"target_type"`
	TargetID      string             `json:"target_id"`
	DiscountType  DiscountType       `json:"discount_type"`
	DiscountValue float64            `json:"discount_value"`
	StartsAt      *time.Time         `json:"starts_at,omitempty"`
	EndsAt        *time.Time         `json:"ends_at,omitempty"`
	IsActive      *bool              `json:"is_active,omitempty"`
	Exclusive     *bool              `json:"exclusive,omitempty"`
	Priority      *int               `json:"priority,omitempty"`
}

type UpdateDiscountRuleRequest struct {
	Name          *string             `json:"name,omitempty"`
	Description   *string             `json:"description,omitempty"`
	TargetType    *DiscountTargetType `json:"target_type,omitempty"`
	TargetID      *string             `json:"target_id,omitempty"`
	DiscountType  *DiscountType       `json:"discount_type,omitempty"`
	DiscountValue *float64            `json:"discount_value,omitempty"`
	StartsAt      *time.Time          `json:"starts_at,omitempty"`
	EndsAt        *time.Time          `json:"ends_at,omitempty"`
	IsActive      *bool               `json:"is_active,omitempty"`
	Exclusive     *bool               `json:"exclusive,omitempty"`
	Priority      *int                `json:"priority,omitempty"`
}

type ListDiscountRulesFilter struct {
	TenantID   string
	TargetType *DiscountTargetType
	TargetID   *string
	IsActive   *bool
	Limit      int
	Offset     int
}

type PricingPreviewRequest struct {
	TenantID       string             `json:"tenant_id,omitempty"`
	ItemType       DiscountTargetType `json:"item_type"`
	ProductID      *string            `json:"product_id,omitempty"`
	BundleID       *string            `json:"bundle_id,omitempty"`
	Quantity       int                `json:"quantity"`
	UnitPrice      int                `json:"unit_price"`
	DiscountRuleID *string            `json:"discount_rule_id,omitempty"`
	ApplyDiscount  bool               `json:"apply_discount,omitempty"`
	At             *time.Time         `json:"at,omitempty"`
}

type PricingResult struct {
	ItemType        DiscountTargetType `json:"item_type"`
	ProductID       *string            `json:"product_id,omitempty"`
	BundleID        *string            `json:"bundle_id,omitempty"`
	Quantity        int                `json:"quantity"`
	ListUnitPrice   int                `json:"list_unit_price"`
	ListTotalPrice  int                `json:"list_total_price"`
	UnitPrice       int                `json:"unit_price"`
	TotalPrice      int                `json:"total_price"`
	DiscountAmount  int                `json:"discount_amount"`
	DiscountRuleID  *string            `json:"discount_rule_id,omitempty"`
	DiscountType    *DiscountType      `json:"discount_type,omitempty"`
	DiscountValue   *float64           `json:"discount_value,omitempty"`
	DiscountName    *string            `json:"discount_name,omitempty"`
	PricingSnapshot json.RawMessage    `json:"pricing_snapshot,omitempty"`
}
