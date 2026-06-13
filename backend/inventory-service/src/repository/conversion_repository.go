package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type ConversionRepository struct {
	db *sql.DB
}

func NewConversionRepository(db *sql.DB) *ConversionRepository {
	return &ConversionRepository{db: db}
}

func (r *ConversionRepository) DB() *sql.DB {
	return r.db
}

func (r *ConversionRepository) ListByIngredient(ctx context.Context, tenantID, ingredientID uuid.UUID) ([]models.IngredientUOMConversion, error) {
	var conversions []models.IngredientUOMConversion
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		conversions, err = r.ListByIngredientTx(ctx, tx, tenantID, ingredientID)
		return err
	})
	return conversions, err
}

func (r *ConversionRepository) ListByIngredientTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID uuid.UUID) ([]models.IngredientUOMConversion, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier, created_at, updated_at
		FROM ingredient_uom_conversions
		WHERE tenant_id = $1 AND ingredient_id = $2
		ORDER BY created_at DESC
	`, tenantID, ingredientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversions: %w", err)
	}
	defer rows.Close()

	var conversions []models.IngredientUOMConversion
	for rows.Next() {
		conversion, err := scanConversion(rows)
		if err != nil {
			return nil, err
		}
		conversions = append(conversions, *conversion)
	}
	return conversions, rows.Err()
}

func (r *ConversionRepository) FindByID(ctx context.Context, tenantID, ingredientID, id uuid.UUID) (*models.IngredientUOMConversion, error) {
	var conversion *models.IngredientUOMConversion
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		conversion, err = r.FindByIDTx(ctx, tx, tenantID, ingredientID, id)
		return err
	})
	return conversion, err
}

func (r *ConversionRepository) FindByIDTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID, id uuid.UUID) (*models.IngredientUOMConversion, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier, created_at, updated_at
		FROM ingredient_uom_conversions
		WHERE id = $1 AND tenant_id = $2 AND ingredient_id = $3
	`, id, tenantID, ingredientID)
	return scanConversion(row)
}

func (r *ConversionRepository) FindMultiplier(ctx context.Context, tenantID uuid.UUID, ingredientID *uuid.UUID, fromUOMID, toUOMID uuid.UUID) (*models.IngredientUOMConversion, error) {
	var conversion *models.IngredientUOMConversion
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		conversion, err = r.FindMultiplierTx(ctx, tx, tenantID, ingredientID, fromUOMID, toUOMID)
		return err
	})
	return conversion, err
}

func (r *ConversionRepository) FindMultiplierTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, ingredientID *uuid.UUID, fromUOMID, toUOMID uuid.UUID) (*models.IngredientUOMConversion, error) {
	if ingredientID != nil {
		if c, err := r.findMultiplierQueryTx(ctx, tx, tenantID, ingredientID, fromUOMID, toUOMID); err == nil {
			return c, nil
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier, created_at, updated_at
		FROM ingredient_uom_conversions
		WHERE tenant_id IS NULL AND ingredient_id IS NULL AND from_uom_id = $1 AND to_uom_id = $2
	`, fromUOMID, toUOMID)
	return scanConversion(row)
}

func (r *ConversionRepository) findMultiplierQueryTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, ingredientID *uuid.UUID, fromUOMID, toUOMID uuid.UUID) (*models.IngredientUOMConversion, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier, created_at, updated_at
		FROM ingredient_uom_conversions
		WHERE tenant_id = $1 AND ingredient_id = $2 AND from_uom_id = $3 AND to_uom_id = $4
	`, tenantID, *ingredientID, fromUOMID, toUOMID)
	return scanConversion(row)
}

func (r *ConversionRepository) Create(ctx context.Context, conversion *models.IngredientUOMConversion) error {
	if conversion.TenantID == nil {
		return fmt.Errorf("tenant_id is required")
	}
	return WithTenantTx(ctx, r.db, *conversion.TenantID, nil, func(tx *sql.Tx) error {
		return r.CreateTx(ctx, tx, conversion)
	})
}

func (r *ConversionRepository) CreateTx(ctx context.Context, tx *sql.Tx, conversion *models.IngredientUOMConversion) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO ingredient_uom_conversions (tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, conversion.TenantID, conversion.IngredientID, conversion.FromUOMID, conversion.ToUOMID, conversion.Multiplier).
		Scan(&conversion.ID, &conversion.CreatedAt, &conversion.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conversion: %w", err)
	}
	return nil
}

func (r *ConversionRepository) Update(ctx context.Context, conversion *models.IngredientUOMConversion) error {
	if conversion.TenantID == nil || conversion.IngredientID == nil {
		return fmt.Errorf("tenant_id and ingredient_id are required")
	}
	return WithTenantTx(ctx, r.db, *conversion.TenantID, nil, func(tx *sql.Tx) error {
		return r.UpdateTx(ctx, tx, conversion)
	})
}

func (r *ConversionRepository) UpdateTx(ctx context.Context, tx *sql.Tx, conversion *models.IngredientUOMConversion) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE ingredient_uom_conversions
		SET from_uom_id = $1, to_uom_id = $2, multiplier = $3
		WHERE id = $4 AND tenant_id = $5 AND ingredient_id = $6
	`, conversion.FromUOMID, conversion.ToUOMID, conversion.Multiplier, conversion.ID, conversion.TenantID, conversion.IngredientID)
	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}
	return nil
}

func (r *ConversionRepository) Delete(ctx context.Context, tenantID, ingredientID, id uuid.UUID) error {
	return WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, ingredientID, id)
	})
}

func (r *ConversionRepository) DeleteTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID, id uuid.UUID) error {
	result, err := tx.ExecContext(ctx, `
		DELETE FROM ingredient_uom_conversions WHERE id = $1 AND tenant_id = $2 AND ingredient_id = $3
	`, id, tenantID, ingredientID)
	if err != nil {
		return fmt.Errorf("failed to delete conversion: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanConversion(row scanner) (*models.IngredientUOMConversion, error) {
	var conversion models.IngredientUOMConversion
	var tenantID, ingredientID sql.NullString
	if err := row.Scan(
		&conversion.ID,
		&tenantID,
		&ingredientID,
		&conversion.FromUOMID,
		&conversion.ToUOMID,
		&conversion.Multiplier,
		&conversion.CreatedAt,
		&conversion.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if tenantID.Valid {
		parsed, err := uuid.Parse(tenantID.String)
		if err != nil {
			return nil, err
		}
		conversion.TenantID = &parsed
	}
	if ingredientID.Valid {
		parsed, err := uuid.Parse(ingredientID.String)
		if err != nil {
			return nil, err
		}
		conversion.IngredientID = &parsed
	}
	return &conversion, nil
}
