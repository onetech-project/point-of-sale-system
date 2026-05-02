package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pos/tenant-service/src/models"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) Create(ctx context.Context, tx *sql.Tx, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, business_name, slug, status, subscription_plan, billing_cycle,
			trial_started_at, trial_ends_at, storage_quota_bytes, storage_used_bytes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}

	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	if tenant.Status == "" {
		tenant.Status = string(models.TenantStatusInactive)
	}

	if tenant.SubscriptionPlan == "" {
		tenant.SubscriptionPlan = string(models.SubscriptionPlanTrial)
	}

	if tenant.BillingCycle == "" {
		tenant.BillingCycle = string(models.BillingCycleMonthly)
	}

	if tenant.TrialStartedAt == nil {
		tenant.TrialStartedAt = &now
	}

	if tenant.TrialEndsAt == nil {
		trialEnd := now.AddDate(0, 0, models.TrialDurationDays)
		tenant.TrialEndsAt = &trialEnd
	}

	if tenant.StorageQuotaBytes == 0 {
		tenant.StorageQuotaBytes = models.DefaultStorageQuotaBytes
	}

	_, err := tx.ExecContext(ctx, query,
		tenant.ID,
		tenant.BusinessName,
		tenant.Slug,
		tenant.Status,
		tenant.SubscriptionPlan,
		tenant.BillingCycle,
		tenant.TrialStartedAt,
		tenant.TrialEndsAt,
		tenant.StorageQuotaBytes,
		tenant.StorageUsedBytes,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	return err
}

func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	query := `
		SELECT id, business_name, slug, status, subscription_plan, billing_cycle,
		       trial_started_at, trial_ends_at, subscribed_at, subscription_ends_at,
		       storage_quota_bytes, storage_used_bytes, created_at, updated_at
		FROM tenants
		WHERE slug = $1 AND status != 'deleted'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
		&tenant.SubscriptionPlan,
		&tenant.BillingCycle,
		&tenant.TrialStartedAt,
		&tenant.TrialEndsAt,
		&tenant.SubscribedAt,
		&tenant.SubscriptionEndsAt,
		&tenant.StorageQuotaBytes,
		&tenant.StorageUsedBytes,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*models.Tenant, error) {
	query := `
		SELECT id, business_name, slug, status, subscription_plan, billing_cycle,
		       trial_started_at, trial_ends_at, subscribed_at, subscription_ends_at,
		       storage_quota_bytes, storage_used_bytes, created_at, updated_at
		FROM tenants
		WHERE id = $1 AND status != 'deleted'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
		&tenant.SubscriptionPlan,
		&tenant.BillingCycle,
		&tenant.TrialStartedAt,
		&tenant.TrialEndsAt,
		&tenant.SubscribedAt,
		&tenant.SubscriptionEndsAt,
		&tenant.StorageQuotaBytes,
		&tenant.StorageUsedBytes,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	query := `
		UPDATE tenants
		SET business_name = $1, slug = $2, status = $3, updated_at = $4
		WHERE id = $5
	`

	tenant.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		tenant.BusinessName,
		tenant.Slug,
		tenant.Status,
		tenant.UpdatedAt,
		tenant.ID,
	)

	return err
}
