package models

import "time"

// OrderSettings represents the order configuration for a tenant
type OrderSettings struct {
	ID                       string    `json:"id" db:"id"`
	TenantID                 string    `json:"tenant_id" db:"tenant_id"`
	DeliveryEnabled          bool      `json:"delivery_enabled" db:"delivery_enabled"`
	PickupEnabled            bool      `json:"pickup_enabled" db:"pickup_enabled"`
	DineInEnabled            bool      `json:"dine_in_enabled" db:"dine_in_enabled"`
	DefaultDeliveryFee       int       `json:"default_delivery_fee" db:"default_delivery_fee"`
	MinOrderAmount           int       `json:"min_order_amount" db:"min_order_amount"`
	MaxDeliveryDistance      float64   `json:"max_delivery_distance" db:"max_delivery_distance"`
	EstimatedPrepTime        int       `json:"estimated_prep_time" db:"estimated_prep_time"`
	AutoAcceptOrders         bool      `json:"auto_accept_orders" db:"auto_accept_orders"`
	RequirePhoneVerification bool      `json:"require_phone_verification" db:"require_phone_verification"`
	ChargeDeliveryFee        bool      `json:"charge_delivery_fee" db:"charge_delivery_fee"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateOrderSettingsRequest represents the request to update order settings
type UpdateOrderSettingsRequest struct {
	DeliveryEnabled          *bool    `json:"delivery_enabled"`
	PickupEnabled            *bool    `json:"pickup_enabled"`
	DineInEnabled            *bool    `json:"dine_in_enabled"`
	DefaultDeliveryFee       *int     `json:"default_delivery_fee"`
	MinOrderAmount           *int     `json:"min_order_amount"`
	MaxDeliveryDistance      *float64 `json:"max_delivery_distance"`
	EstimatedPrepTime        *int     `json:"estimated_prep_time"`
	AutoAcceptOrders         *bool    `json:"auto_accept_orders"`
	RequirePhoneVerification *bool    `json:"require_phone_verification"`
	ChargeDeliveryFee        *bool    `json:"charge_delivery_fee"`
}
