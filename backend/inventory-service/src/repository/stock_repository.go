package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/shopspring/decimal"
)

type StockRepository struct {
	db *sql.DB
}

func NewStockRepository(db *sql.DB) *StockRepository {
	return &StockRepository{db: db}
}

func (r *StockRepository) DB() *sql.DB {
	return r.db
}

func (r *StockRepository) CreateMovementTx(ctx context.Context, tx *sql.Tx, movement *models.StockMovement) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO stock_movements (
			tenant_id, outlet_id, ingredient_id, movement_type, quantity_base,
			unit_cost, total_cost, reference_type, reference_id, idempotency_key,
			reason, occurred_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`, movement.TenantID, movement.OutletID, movement.IngredientID, movement.MovementType,
		movement.QuantityBase, movement.UnitCost, movement.TotalCost, movement.ReferenceType,
		movement.ReferenceID, movement.IdempotencyKey, movement.Reason, movement.OccurredAt,
		movement.CreatedBy).Scan(&movement.ID, &movement.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create stock movement: %w", err)
	}
	return nil
}

func (r *StockRepository) LockIngredientTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID uuid.UUID) (*models.Ingredient, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT i.id, i.tenant_id, i.outlet_id, i.sku, i.name, i.base_uom_id, u.code,
		       i.current_stock_base, i.average_cost_per_base_unit, i.minimum_stock_base,
		       i.track_stock, i.is_active, i.created_at, i.updated_at
		FROM ingredients i
		JOIN uoms u ON u.id = i.base_uom_id AND (u.tenant_id IS NULL OR u.tenant_id = i.tenant_id)
		WHERE i.id = $1 AND i.tenant_id = $2
		FOR UPDATE
	`, ingredientID, tenantID)
	return scanIngredient(row)
}

func (r *StockRepository) UpdateIngredientStockTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID uuid.UUID, newStock, averageCost decimal.Decimal) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE ingredients
		SET current_stock_base = $1, average_cost_per_base_unit = $2
		WHERE id = $3 AND tenant_id = $4
	`, newStock, averageCost, ingredientID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update ingredient stock: %w", err)
	}
	return nil
}

func (r *StockRepository) ListMovements(ctx context.Context, tenantID, ingredientID uuid.UUID, limit, offset int) ([]models.StockMovement, int, error) {
	var movements []models.StockMovement
	var total int
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		movements, total, err = r.ListMovementsTx(ctx, tx, tenantID, ingredientID, limit, offset)
		return err
	})
	return movements, total, err
}

func (r *StockRepository) ListMovementsTx(ctx context.Context, tx *sql.Tx, tenantID, ingredientID uuid.UUID, limit, offset int) ([]models.StockMovement, int, error) {
	var total int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM stock_movements WHERE tenant_id = $1 AND ingredient_id = $2
	`, tenantID, ingredientID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count movements: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, tenant_id, outlet_id, ingredient_id, movement_type, quantity_base,
		       unit_cost, total_cost, reference_type, reference_id, idempotency_key,
		       reason, occurred_at, created_by, created_at
		FROM stock_movements
		WHERE tenant_id = $1 AND ingredient_id = $2
		ORDER BY occurred_at DESC, created_at DESC
		LIMIT $3 OFFSET $4
	`, tenantID, ingredientID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list movements: %w", err)
	}
	defer rows.Close()

	var movements []models.StockMovement
	for rows.Next() {
		movement, err := scanMovement(rows)
		if err != nil {
			return nil, 0, err
		}
		movements = append(movements, *movement)
	}
	return movements, total, rows.Err()
}

func (r *StockRepository) ListLowStock(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]models.Ingredient, int, error) {
	var ingredients []models.Ingredient
	var total int
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		ingredients, total, err = r.ListLowStockTx(ctx, tx, tenantID, limit, offset)
		return err
	})
	return ingredients, total, err
}

func (r *StockRepository) ListLowStockTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, limit, offset int) ([]models.Ingredient, int, error) {
	where := `
		WHERE i.tenant_id = $1
		  AND i.is_active = TRUE
		  AND i.track_stock = TRUE
		  AND i.current_stock_base <= i.minimum_stock_base
	`
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ingredients i `+where, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count low-stock ingredients: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT i.id, i.tenant_id, i.outlet_id, i.sku, i.name, i.base_uom_id, u.code,
		       i.current_stock_base, i.average_cost_per_base_unit, i.minimum_stock_base,
		       i.track_stock, i.is_active, i.created_at, i.updated_at
		FROM ingredients i
		JOIN uoms u ON u.id = i.base_uom_id AND (u.tenant_id IS NULL OR u.tenant_id = i.tenant_id)
		`+where+`
		ORDER BY i.current_stock_base ASC, i.name
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list low-stock ingredients: %w", err)
	}
	defer rows.Close()
	var ingredients []models.Ingredient
	for rows.Next() {
		ingredient, err := scanIngredient(rows)
		if err != nil {
			return nil, 0, err
		}
		ingredients = append(ingredients, *ingredient)
	}
	return ingredients, total, rows.Err()
}

func (r *StockRepository) Valuation(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		total, err = r.ValuationTx(ctx, tx, tenantID)
		return err
	})
	return total, err
}

func (r *StockRepository) ValuationTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(current_stock_base * average_cost_per_base_unit), 0)
		FROM ingredients
		WHERE tenant_id = $1 AND is_active = TRUE AND track_stock = TRUE
	`, tenantID).Scan(&total)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to calculate valuation: %w", err)
	}
	return total, nil
}

func scanMovement(row scanner) (*models.StockMovement, error) {
	var movement models.StockMovement
	var outletID, referenceType, referenceID, idempotencyKey, reason, createdBy sql.NullString
	if err := row.Scan(
		&movement.ID,
		&movement.TenantID,
		&outletID,
		&movement.IngredientID,
		&movement.MovementType,
		&movement.QuantityBase,
		&movement.UnitCost,
		&movement.TotalCost,
		&referenceType,
		&referenceID,
		&idempotencyKey,
		&reason,
		&movement.OccurredAt,
		&createdBy,
		&movement.CreatedAt,
	); err != nil {
		return nil, err
	}
	if outletID.Valid {
		parsed, err := uuid.Parse(outletID.String)
		if err != nil {
			return nil, err
		}
		movement.OutletID = &parsed
	}
	if createdBy.Valid {
		parsed, err := uuid.Parse(createdBy.String)
		if err != nil {
			return nil, err
		}
		movement.CreatedBy = &parsed
	}
	if referenceType.Valid {
		movement.ReferenceType = &referenceType.String
	}
	if referenceID.Valid {
		movement.ReferenceID = &referenceID.String
	}
	if idempotencyKey.Valid {
		movement.IdempotencyKey = &idempotencyKey.String
	}
	if reason.Valid {
		movement.Reason = &reason.String
	}
	return &movement, nil
}
