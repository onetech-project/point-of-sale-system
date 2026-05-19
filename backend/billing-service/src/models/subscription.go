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
	ID                   string     `json:"id"`
	TenantID             string     `json:"tenant_id"`
	InvoiceNumber        string     `json:"invoice_number"`
	AmountIDR            int        `json:"amount_idr"`
	BillingInterval      string     `json:"billing_interval"`
	PeriodStart          time.Time  `json:"period_start"`
	PeriodEnd            time.Time  `json:"period_end"`
	DueAt                time.Time  `json:"due_at"`
	Status               string     `json:"status"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	MidtransOrderID      *string    `json:"midtrans_order_id,omitempty"`
	MidtransPaymentURL   *string    `json:"payment_url,omitempty"`
	PaymentLinkExpiresAt *time.Time `json:"payment_link_expires_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// BillingPaymentAttempt represents a single payment attempt for an invoice.
type BillingPaymentAttempt struct {
	ID              string    `json:"id"`
	InvoiceID       string    `json:"invoice_id"`
	TenantID        string    `json:"tenant_id"`
	AmountIDR       int       `json:"amount_idr"`
	MidtransOrderID *string   `json:"midtrans_order_id,omitempty"`
	PaymentMethod   *string   `json:"payment_method,omitempty"`
	Status          string    `json:"status"`
	ErrorMsg        *string   `json:"error_msg,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// SubscriptionStatusResponse is returned by both the public and internal subscription endpoints.
type SubscriptionStatusResponse struct {
	TenantID           string          `json:"tenant_id"`
	SubscriptionPlan   string          `json:"subscription_plan"`
	SubscriptionStatus string          `json:"subscription_status"`
	BillingCycle       string          `json:"billing_cycle"`
	TrialEndsAt        *time.Time      `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt *time.Time      `json:"subscription_ends_at,omitempty"`
	RetentionStartedAt *time.Time      `json:"retention_started_at,omitempty"`
	RetentionCleanupAt *time.Time      `json:"retention_cleanup_at,omitempty"`
	DataAnonymizedAt   *time.Time      `json:"data_anonymized_at,omitempty"`
	PaymentDueAt       *time.Time      `json:"payment_due_at,omitempty"`
	CurrentInvoice     *BillingInvoice `json:"current_invoice,omitempty"`
	IsActive           bool            `json:"is_active"`
	DaysRemaining      int             `json:"days_remaining"`
}
