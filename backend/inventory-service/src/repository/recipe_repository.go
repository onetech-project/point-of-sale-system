package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type RecipeRepository struct {
	db *sql.DB
}

func NewRecipeRepository(db *sql.DB) *RecipeRepository {
	return &RecipeRepository{db: db}
}

func (r *RecipeRepository) DB() *sql.DB {
	return r.db
}

func (r *RecipeRepository) FindProduct(ctx context.Context, tenantID, productID uuid.UUID) (*models.ProductSnapshot, error) {
	var product *models.ProductSnapshot
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		product, err = r.FindProductTx(ctx, tx, tenantID, productID)
		return err
	})
	return product, err
}

func (r *RecipeRepository) FindProductTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID) (*models.ProductSnapshot, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, sku, selling_price
		FROM products
		WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL
	`, productID, tenantID)
	return scanProductSnapshot(row)
}

func (r *RecipeRepository) NextVersionTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID) (int, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM recipes
		WHERE tenant_id = $1 AND product_id = $2
	`, tenantID, productID).Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to determine next recipe version: %w", err)
	}
	return version, nil
}

func (r *RecipeRepository) DeactivateActiveTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE recipes
		SET is_active = FALSE
		WHERE tenant_id = $1 AND product_id = $2 AND is_active = TRUE
	`, tenantID, productID)
	if err != nil {
		return fmt.Errorf("failed to deactivate active recipe: %w", err)
	}
	return nil
}

func (r *RecipeRepository) CreateRecipeTx(ctx context.Context, tx *sql.Tx, recipe *models.Recipe) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO recipes (tenant_id, product_id, version, yield_quantity, is_active, effective_from)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, recipe.TenantID, recipe.ProductID, recipe.Version, recipe.YieldQuantity, recipe.IsActive, recipe.EffectiveFrom).
		Scan(&recipe.ID, &recipe.CreatedAt, &recipe.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create recipe: %w", err)
	}
	return nil
}

func (r *RecipeRepository) CreateItemTx(ctx context.Context, tx *sql.Tx, item *models.RecipeItem) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO recipe_items (
			tenant_id, recipe_id, ingredient_id, quantity, uom_id,
			normalized_quantity_base, waste_percentage
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, item.TenantID, item.RecipeID, item.IngredientID, item.Quantity, item.UOMID,
		item.NormalizedQuantityBase, item.WastePercentage).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create recipe item: %w", err)
	}
	return nil
}

func (r *RecipeRepository) CreateOverheadTx(ctx context.Context, tx *sql.Tx, overhead *models.RecipeOverhead) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO recipe_overheads (tenant_id, recipe_id, name, calculation_type, amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, overhead.TenantID, overhead.RecipeID, overhead.Name, overhead.CalculationType, overhead.Amount).
		Scan(&overhead.ID, &overhead.CreatedAt, &overhead.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create recipe overhead: %w", err)
	}
	return nil
}

func (r *RecipeRepository) GetActive(ctx context.Context, tenantID, productID uuid.UUID) (*models.RecipeAggregate, error) {
	var aggregate *models.RecipeAggregate
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		aggregate, err = r.GetActiveTx(ctx, tx, tenantID, productID)
		return err
	})
	return aggregate, err
}

func (r *RecipeRepository) GetActiveTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID) (*models.RecipeAggregate, error) {
	recipe, err := r.getRecipeTx(ctx, tx, `
		SELECT id, tenant_id, product_id, version, yield_quantity, is_active, effective_from, created_at, updated_at
		FROM recipes
		WHERE tenant_id = $1 AND product_id = $2 AND is_active = TRUE
	`, tenantID, productID)
	if err != nil {
		return nil, err
	}
	return r.loadAggregateTx(ctx, tx, tenantID, recipe)
}

func (r *RecipeRepository) GetVersion(ctx context.Context, tenantID, productID uuid.UUID, version int) (*models.RecipeAggregate, error) {
	var aggregate *models.RecipeAggregate
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		aggregate, err = r.GetVersionTx(ctx, tx, tenantID, productID, version)
		return err
	})
	return aggregate, err
}

func (r *RecipeRepository) GetVersionTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID, version int) (*models.RecipeAggregate, error) {
	recipe, err := r.getRecipeTx(ctx, tx, `
		SELECT id, tenant_id, product_id, version, yield_quantity, is_active, effective_from, created_at, updated_at
		FROM recipes
		WHERE tenant_id = $1 AND product_id = $2 AND version = $3
	`, tenantID, productID, version)
	if err != nil {
		return nil, err
	}
	return r.loadAggregateTx(ctx, tx, tenantID, recipe)
}

func (r *RecipeRepository) ListVersions(ctx context.Context, tenantID, productID uuid.UUID) ([]models.RecipeVersionSummary, error) {
	var versions []models.RecipeVersionSummary
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		versions, err = r.ListVersionsTx(ctx, tx, tenantID, productID)
		return err
	})
	return versions, err
}

func (r *RecipeRepository) ListVersionsTx(ctx context.Context, tx *sql.Tx, tenantID, productID uuid.UUID) ([]models.RecipeVersionSummary, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, product_id, version, yield_quantity, is_active, effective_from, created_at
		FROM recipes
		WHERE tenant_id = $1 AND product_id = $2
		ORDER BY version DESC
	`, tenantID, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe versions: %w", err)
	}
	defer rows.Close()

	var versions []models.RecipeVersionSummary
	for rows.Next() {
		var version models.RecipeVersionSummary
		if err := rows.Scan(
			&version.ID,
			&version.ProductID,
			&version.Version,
			&version.YieldQuantity.Decimal,
			&version.IsActive,
			&version.EffectiveFrom,
			&version.CreatedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (r *RecipeRepository) getRecipeTx(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (*models.Recipe, error) {
	var recipe models.Recipe
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&recipe.ID,
		&recipe.TenantID,
		&recipe.ProductID,
		&recipe.Version,
		&recipe.YieldQuantity,
		&recipe.IsActive,
		&recipe.EffectiveFrom,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *RecipeRepository) loadAggregateTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, recipe *models.Recipe) (*models.RecipeAggregate, error) {
	product, err := r.FindProductTx(ctx, tx, tenantID, recipe.ProductID)
	if err != nil {
		return nil, err
	}
	items, err := r.listItemsTx(ctx, tx, tenantID, recipe.ID)
	if err != nil {
		return nil, err
	}
	overheads, err := r.listOverheadsTx(ctx, tx, tenantID, recipe.ID)
	if err != nil {
		return nil, err
	}
	return &models.RecipeAggregate{
		Recipe:    *recipe,
		Product:   *product,
		Items:     items,
		Overheads: overheads,
	}, nil
}

func (r *RecipeRepository) listItemsTx(ctx context.Context, tx *sql.Tx, tenantID, recipeID uuid.UUID) ([]models.RecipeItem, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT ri.id, ri.tenant_id, ri.recipe_id, ri.ingredient_id, i.name,
		       ri.quantity, ri.uom_id, u.code, ri.normalized_quantity_base,
		       ri.waste_percentage, i.average_cost_per_base_unit,
		       ri.created_at, ri.updated_at
		FROM recipe_items ri
		JOIN ingredients i ON i.id = ri.ingredient_id AND i.tenant_id = ri.tenant_id
		JOIN uoms u ON u.id = ri.uom_id AND (u.tenant_id IS NULL OR u.tenant_id = ri.tenant_id)
		WHERE ri.tenant_id = $1 AND ri.recipe_id = $2
		ORDER BY ri.created_at ASC
	`, tenantID, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe items: %w", err)
	}
	defer rows.Close()

	var items []models.RecipeItem
	for rows.Next() {
		var item models.RecipeItem
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.RecipeID,
			&item.IngredientID,
			&item.IngredientName,
			&item.Quantity,
			&item.UOMID,
			&item.UOMCode,
			&item.NormalizedQuantityBase,
			&item.WastePercentage,
			&item.AverageCostPerBaseUnit,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *RecipeRepository) listOverheadsTx(ctx context.Context, tx *sql.Tx, tenantID, recipeID uuid.UUID) ([]models.RecipeOverhead, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, tenant_id, recipe_id, name, calculation_type, amount, created_at, updated_at
		FROM recipe_overheads
		WHERE tenant_id = $1 AND recipe_id = $2
		ORDER BY created_at ASC
	`, tenantID, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe overheads: %w", err)
	}
	defer rows.Close()

	var overheads []models.RecipeOverhead
	for rows.Next() {
		var overhead models.RecipeOverhead
		if err := rows.Scan(
			&overhead.ID,
			&overhead.TenantID,
			&overhead.RecipeID,
			&overhead.Name,
			&overhead.CalculationType,
			&overhead.Amount,
			&overhead.CreatedAt,
			&overhead.UpdatedAt,
		); err != nil {
			return nil, err
		}
		overheads = append(overheads, overhead)
	}
	return overheads, rows.Err()
}

func scanProductSnapshot(row scanner) (*models.ProductSnapshot, error) {
	var product models.ProductSnapshot
	if err := row.Scan(&product.ID, &product.TenantID, &product.Name, &product.SKU, &product.SellingPrice); err != nil {
		return nil, err
	}
	return &product, nil
}
