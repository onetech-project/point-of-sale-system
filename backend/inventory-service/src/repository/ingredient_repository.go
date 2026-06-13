package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type IngredientRepository struct {
	db *sql.DB
}

func NewIngredientRepository(db *sql.DB) *IngredientRepository {
	return &IngredientRepository{db: db}
}

func (r *IngredientRepository) List(ctx context.Context, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.Ingredient, int, error) {
	var ingredients []models.Ingredient
	var total int
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		ingredients, total, err = r.ListTx(ctx, tx, tenantID, search, activeOnly, limit, offset)
		return err
	})
	return ingredients, total, err
}

func (r *IngredientRepository) ListTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.Ingredient, int, error) {
	where := `WHERE i.tenant_id = $1`
	args := []interface{}{tenantID}
	arg := 2
	if search != "" {
		where += fmt.Sprintf(` AND (i.name ILIKE $%d OR i.sku ILIKE $%d)`, arg, arg)
		args = append(args, "%"+search+"%")
		arg++
	}
	if activeOnly {
		where += ` AND i.is_active = TRUE`
	}

	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingredients i `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count ingredients: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT i.id, i.tenant_id, i.outlet_id, i.sku, i.name, i.base_uom_id, u.code,
		       i.current_stock_base, i.average_cost_per_base_unit, i.minimum_stock_base,
		       i.track_stock, i.is_active, i.created_at, i.updated_at
		FROM ingredients i
		JOIN uoms u ON u.id = i.base_uom_id AND (u.tenant_id IS NULL OR u.tenant_id = i.tenant_id)
		%s
		ORDER BY i.name
		LIMIT $%d OFFSET $%d
	`, where, arg, arg+1)
	args = append(args, limit, offset)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []models.Ingredient
	for rows.Next() {
		item, err := scanIngredient(rows)
		if err != nil {
			return nil, 0, err
		}
		ingredients = append(ingredients, *item)
	}
	return ingredients, total, rows.Err()
}

func (r *IngredientRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Ingredient, error) {
	var ingredient *models.Ingredient
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		ingredient, err = r.FindByIDTx(ctx, tx, tenantID, id)
		return err
	})
	return ingredient, err
}

func (r *IngredientRepository) FindByIDTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) (*models.Ingredient, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT i.id, i.tenant_id, i.outlet_id, i.sku, i.name, i.base_uom_id, u.code,
		       i.current_stock_base, i.average_cost_per_base_unit, i.minimum_stock_base,
		       i.track_stock, i.is_active, i.created_at, i.updated_at
		FROM ingredients i
		JOIN uoms u ON u.id = i.base_uom_id AND (u.tenant_id IS NULL OR u.tenant_id = i.tenant_id)
		WHERE i.id = $1 AND i.tenant_id = $2
	`, id, tenantID)
	return scanIngredient(row)
}

func (r *IngredientRepository) Create(ctx context.Context, ingredient *models.Ingredient) error {
	return WithTenantTx(ctx, r.db, ingredient.TenantID, nil, func(tx *sql.Tx) error {
		return r.CreateTx(ctx, tx, ingredient)
	})
}

func (r *IngredientRepository) CreateTx(ctx context.Context, tx *sql.Tx, ingredient *models.Ingredient) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO ingredients (
			tenant_id, outlet_id, sku, name, base_uom_id,
			average_cost_per_base_unit, minimum_stock_base, track_stock, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, current_stock_base, created_at, updated_at
	`, ingredient.TenantID, ingredient.OutletID, ingredient.SKU, ingredient.Name, ingredient.BaseUOMID,
		ingredient.AverageCostPerBaseUnit, ingredient.MinimumStockBase, ingredient.TrackStock, ingredient.IsActive).
		Scan(&ingredient.ID, &ingredient.CurrentStockBase, &ingredient.CreatedAt, &ingredient.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create ingredient: %w", err)
	}
	return nil
}

func (r *IngredientRepository) Update(ctx context.Context, ingredient *models.Ingredient) error {
	return WithTenantTx(ctx, r.db, ingredient.TenantID, nil, func(tx *sql.Tx) error {
		return r.UpdateTx(ctx, tx, ingredient)
	})
}

func (r *IngredientRepository) UpdateTx(ctx context.Context, tx *sql.Tx, ingredient *models.Ingredient) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE ingredients
		SET outlet_id = $1, sku = $2, name = $3, base_uom_id = $4,
		    average_cost_per_base_unit = $5, minimum_stock_base = $6,
		    track_stock = $7, is_active = $8
		WHERE id = $9 AND tenant_id = $10
	`, ingredient.OutletID, ingredient.SKU, ingredient.Name, ingredient.BaseUOMID,
		ingredient.AverageCostPerBaseUnit, ingredient.MinimumStockBase, ingredient.TrackStock,
		ingredient.IsActive, ingredient.ID, ingredient.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update ingredient: %w", err)
	}
	return nil
}

func (r *IngredientRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, id)
	})
}

func (r *IngredientRepository) DeleteTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE ingredients SET is_active = FALSE WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete ingredient: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanIngredient(row scanner) (*models.Ingredient, error) {
	var ingredient models.Ingredient
	var outletID sql.NullString
	var baseCode sql.NullString
	if err := row.Scan(
		&ingredient.ID,
		&ingredient.TenantID,
		&outletID,
		&ingredient.SKU,
		&ingredient.Name,
		&ingredient.BaseUOMID,
		&baseCode,
		&ingredient.CurrentStockBase,
		&ingredient.AverageCostPerBaseUnit,
		&ingredient.MinimumStockBase,
		&ingredient.TrackStock,
		&ingredient.IsActive,
		&ingredient.CreatedAt,
		&ingredient.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if outletID.Valid {
		parsed, err := uuid.Parse(outletID.String)
		if err != nil {
			return nil, err
		}
		ingredient.OutletID = &parsed
	}
	if baseCode.Valid {
		ingredient.BaseUOMCode = &baseCode.String
	}
	return &ingredient, nil
}
