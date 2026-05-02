package models

import (
	"time"
)

type Tenant struct {
	ID                 string                 `json:"id" db:"id"`
	BusinessName       string                 `json:"business_name" db:"business_name"`
	Slug               string                 `json:"slug" db:"slug"`
	Status             string                 `json:"status" db:"status"`
	Settings           map[string]interface{} `json:"settings" db:"settings"`
	StorageUsedBytes   int64                  `json:"storage_used_bytes" db:"storage_used_bytes"`
	StorageQuotaBytes  int64                  `json:"storage_quota_bytes" db:"storage_quota_bytes"`
	SubscriptionPlan   string                 `json:"subscription_plan" db:"subscription_plan"`
	BillingCycle       string                 `json:"billing_cycle" db:"billing_cycle"`
	TrialStartedAt     *time.Time             `json:"trial_started_at" db:"trial_started_at"`
	TrialEndsAt        *time.Time             `json:"trial_ends_at" db:"trial_ends_at"`
	SubscribedAt       *time.Time             `json:"subscribed_at" db:"subscribed_at"`
	SubscriptionEndsAt *time.Time             `json:"subscription_ends_at" db:"subscription_ends_at"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// Subscription plan constants
type SubscriptionPlan string

const (
	SubscriptionPlanTrial        SubscriptionPlan = "trial"
	SubscriptionPlanStarter      SubscriptionPlan = "starter"
	SubscriptionPlanProfessional SubscriptionPlan = "professional"
	SubscriptionPlanEnterprise   SubscriptionPlan = "enterprise"
)

// Billing cycle constants
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleAnnual  BillingCycle = "annual"
)

// Storage and trial constants
const (
	DefaultStorageQuotaBytes    int64 = 2 * 1024 * 1024 * 1024 // 2 GB
	TrialDurationDays           int   = 7
	DefaultAnnualDiscountPercent int  = 20
)

type CreateTenantRequest struct {
	BusinessName string   `json:"business_name" validate:"required,min=1,max=100"`
	Slug         string   `json:"slug,omitempty" validate:"omitempty,min=3,max=50"`
	Email        string   `json:"email" validate:"required,email"`
	Password     string   `json:"password" validate:"required,min=8"`
	FirstName    string   `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName     string   `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Consents     []string `json:"consents" validate:"dive,oneof=analytics advertising"` // Optional consents granted (required consents implicit)
}

type TenantResponse struct {
	ID                 string                 `json:"id"`
	BusinessName       string                 `json:"business_name"`
	Slug               string                 `json:"slug"`
	Status             string                 `json:"status"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
	StorageUsedBytes   int64                  `json:"storage_used_bytes"`
	StorageQuotaBytes  int64                  `json:"storage_quota_bytes"`
	SubscriptionPlan   string                 `json:"subscription_plan"`
	BillingCycle       string                 `json:"billing_cycle"`
	TrialStartedAt     *time.Time             `json:"trial_started_at,omitempty"`
	TrialEndsAt        *time.Time             `json:"trial_ends_at,omitempty"`
	SubscribedAt       *time.Time             `json:"subscribed_at,omitempty"`
	SubscriptionEndsAt *time.Time             `json:"subscription_ends_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

func (t *Tenant) ToResponse() *TenantResponse {
	return &TenantResponse{
		ID:                 t.ID,
		BusinessName:       t.BusinessName,
		Slug:               t.Slug,
		Status:             t.Status,
		Settings:           t.Settings,
		StorageUsedBytes:   t.StorageUsedBytes,
		StorageQuotaBytes:  t.StorageQuotaBytes,
		SubscriptionPlan:   t.SubscriptionPlan,
		BillingCycle:       t.BillingCycle,
		TrialStartedAt:     t.TrialStartedAt,
		TrialEndsAt:        t.TrialEndsAt,
		SubscribedAt:       t.SubscribedAt,
		SubscriptionEndsAt: t.SubscriptionEndsAt,
		CreatedAt:          t.CreatedAt,
	}
}
