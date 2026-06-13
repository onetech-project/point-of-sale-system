package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type UOMRepository struct {
	db *sql.DB
}

func NewUOMRepository(db *sql.DB) *UOMRepository {
	return &UOMRepository{db: db}
}

func (r *UOMRepository) List(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]models.UOM, error) {
	var uoms []models.UOM
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		uoms, err = r.ListTx(ctx, tx, tenantID, includeInactive)
		return err
	})
	return uoms, err
}

func (r *UOMRepository) ListTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, includeInactive bool) ([]models.UOM, error) {
	query := `
		SELECT id, tenant_id, code, name, category, is_base_unit, is_active, created_at, updated_at
		FROM uoms
		WHERE (tenant_id IS NULL OR tenant_id = $1)
	`
	args := []interface{}{tenantID}
	if !includeInactive {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY category, tenant_id NULLS FIRST, name`

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list uoms: %w", err)
	}
	defer rows.Close()

	var uoms []models.UOM
	for rows.Next() {
		uom, err := scanUOM(rows)
		if err != nil {
			return nil, err
		}
		uoms = append(uoms, *uom)
	}
	return uoms, rows.Err()
}

func (r *UOMRepository) FindByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.UOM, error) {
	var uom *models.UOM
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		uom, err = r.FindByIDTx(ctx, tx, tenantID, id)
		return err
	})
	return uom, err
}

func (r *UOMRepository) FindByIDTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, id uuid.UUID) (*models.UOM, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, code, name, category, is_base_unit, is_active, created_at, updated_at
		FROM uoms
		WHERE id = $1 AND (tenant_id IS NULL OR tenant_id = $2)
	`, id, tenantID)
	return scanUOM(row)
}

func (r *UOMRepository) Create(ctx context.Context, tenantID uuid.UUID, uom *models.UOM) error {
	return WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		return r.CreateTx(ctx, tx, tenantID, uom)
	})
}

func (r *UOMRepository) CreateTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, uom *models.UOM) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO uoms (tenant_id, code, name, category, is_base_unit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, tenantID, uom.Code, uom.Name, uom.Category, uom.IsBaseUnit, uom.IsActive).Scan(&uom.ID, &uom.CreatedAt, &uom.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create uom: %w", err)
	}
	uom.TenantID = &tenantID
	return nil
}

func (r *UOMRepository) Update(ctx context.Context, tenantID, id uuid.UUID, req models.UpdateUOMRequest) (*models.UOM, error) {
	var uom *models.UOM
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		uom, err = r.UpdateTx(ctx, tx, tenantID, id, req)
		return err
	})
	return uom, err
}

func (r *UOMRepository) UpdateTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID, req models.UpdateUOMRequest) (*models.UOM, error) {
	current, err := r.FindByIDTx(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if current.TenantID == nil {
		return nil, fmt.Errorf("global uoms are read-only")
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Category != nil {
		current.Category = *req.Category
	}
	if req.IsBaseUnit != nil {
		current.IsBaseUnit = *req.IsBaseUnit
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE uoms
		SET name = $1, category = $2, is_base_unit = $3, is_active = $4
		WHERE id = $5 AND tenant_id = $6
	`, current.Name, current.Category, current.IsBaseUnit, current.IsActive, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to update uom: %w", err)
	}
	return r.FindByIDTx(ctx, tx, tenantID, id)
}

func (r *UOMRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, id)
	})
}

func (r *UOMRepository) DeleteTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM uoms WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete uom: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUOM(row scanner) (*models.UOM, error) {
	var uom models.UOM
	var tenantID sql.NullString
	if err := row.Scan(&uom.ID, &tenantID, &uom.Code, &uom.Name, &uom.Category, &uom.IsBaseUnit, &uom.IsActive, &uom.CreatedAt, &uom.UpdatedAt); err != nil {
		return nil, err
	}
	if tenantID.Valid {
		parsed, err := uuid.Parse(tenantID.String)
		if err != nil {
			return nil, err
		}
		uom.TenantID = &parsed
	}
	return &uom, nil
}
