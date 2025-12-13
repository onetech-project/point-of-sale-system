package models

import (
	"time"
)

type Tenant struct {
	ID                string                 `json:"id" db:"id"`
	BusinessName      string                 `json:"business_name" db:"business_name"`
	Slug              string                 `json:"slug" db:"slug"`
	Status            string                 `json:"status" db:"status"`
	Settings          map[string]interface{} `json:"settings" db:"settings"`
	StorageUsedBytes  int64                  `json:"storage_used_bytes" db:"storage_used_bytes"`
	StorageQuotaBytes int64                  `json:"storage_quota_bytes" db:"storage_quota_bytes"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

type CreateTenantRequest struct {
	BusinessName string `json:"business_name" validate:"required,min=1,max=100"`
	Slug         string `json:"slug,omitempty" validate:"omitempty,min=3,max=50"`
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=8"`
	FirstName    string `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName     string `json:"last_name,omitempty" validate:"omitempty,max=50"`
}

type TenantResponse struct {
	ID                string                 `json:"id"`
	BusinessName      string                 `json:"business_name"`
	Slug              string                 `json:"slug"`
	Status            string                 `json:"status"`
	Settings          map[string]interface{} `json:"settings,omitempty"`
	StorageUsedBytes  int64                  `json:"storage_used_bytes"`
	StorageQuotaBytes int64                  `json:"storage_quota_bytes"`
	CreatedAt         time.Time              `json:"created_at"`
}

func (t *Tenant) ToResponse() *TenantResponse {
	return &TenantResponse{
		ID:                t.ID,
		BusinessName:      t.BusinessName,
		Slug:              t.Slug,
		Status:            t.Status,
		Settings:          t.Settings,
		StorageUsedBytes:  t.StorageUsedBytes,
		StorageQuotaBytes: t.StorageQuotaBytes,
		CreatedAt:         t.CreatedAt,
	}
}
