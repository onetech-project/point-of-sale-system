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

func (r *TenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, business_name, slug, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}

	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	if tenant.Status == "" {
		tenant.Status = string(models.TenantStatusActive)
	}

	_, err := r.db.ExecContext(ctx, query,
		tenant.ID,
		tenant.BusinessName,
		tenant.Slug,
		tenant.Status,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	return err
}

func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	query := `
		SELECT id, business_name, slug, status, created_at, updated_at
		FROM tenants
		WHERE slug = $1 AND status != 'deleted'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
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
		SELECT id, business_name, slug, status, created_at, updated_at
		FROM tenants
		WHERE id = $1 AND status != 'deleted'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
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
