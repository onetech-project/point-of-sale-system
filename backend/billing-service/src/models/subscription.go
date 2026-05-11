package models

import "time"

// Tenant represents a POS tenant with subscription fields.
type Tenant struct {
	ID                 string
	BusinessName       string
	Email              string // populated separately from users table
	SubscriptionPlan   string
	BillingCycle       string
	TrialStartedAt     *time.Time
	TrialEndsAt        *time.Time
	SubscribedAt       *time.Time
	SubscriptionEndsAt *time.Time
	SubscriptionStatus string
	RetentionStartedAt *time.Time
	DataAnonymizedAt   *time.Time
	StorageQuotaBytes  int64
}

// BillingInvoice represents a billing invoice for a tenant.
type BillingInvoice struct {
	ID                 string
	TenantID           string
	InvoiceNumber      string
	AmountIDR          int
	BillingInterval    string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	Status             string
	PaidAt             *time.Time
	MidtransOrderID    *string
	MidtransPaymentURL *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BillingPaymentAttempt represents a single payment attempt for an invoice.
type BillingPaymentAttempt struct {
	ID              string
	InvoiceID       string
	TenantID        string
	AmountIDR       int
	MidtransOrderID *string
	PaymentMethod   *string
	Status          string
	ErrorMsg        *string
	CreatedAt       time.Time
}

// SubscriptionStatusResponse is returned by both the public and internal subscription endpoints.
type SubscriptionStatusResponse struct {
	TenantID           string     `json:"tenant_id"`
	SubscriptionPlan   string     `json:"subscription_plan"`
	SubscriptionStatus string     `json:"subscription_status"`
	BillingCycle       string     `json:"billing_cycle"`
	TrialEndsAt        *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt *time.Time `json:"subscription_ends_at,omitempty"`
	RetentionStartedAt *time.Time `json:"retention_started_at,omitempty"`
	RetentionCleanupAt *time.Time `json:"retention_cleanup_at,omitempty"`
	DataAnonymizedAt   *time.Time `json:"data_anonymized_at,omitempty"`
	IsActive           bool       `json:"is_active"`
	DaysRemaining      int        `json:"days_remaining"`
}
