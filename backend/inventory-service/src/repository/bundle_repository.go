package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type BundleRepository struct {
	db *sql.DB
}

func NewBundleRepository(db *sql.DB) *BundleRepository {
	return &BundleRepository{db: db}
}

func (r *BundleRepository) DB() *sql.DB {
	return r.db
}

func (r *BundleRepository) List(ctx context.Context, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.BundleAggregate, int, error) {
	var bundles []models.BundleAggregate
	var total int
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		bundles, total, err = r.ListTx(ctx, tx, tenantID, search, activeOnly, limit, offset)
		return err
	})
	return bundles, total, err
}

func (r *BundleRepository) ListTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.BundleAggregate, int, error) {
	where := `WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	arg := 2
	if search != "" {
		where += fmt.Sprintf(` AND (name ILIKE $%d OR sku ILIKE $%d)`, arg, arg)
		args = append(args, "%"+search+"%")
		arg++
	}
	if activeOnly {
		where += ` AND is_active = TRUE`
	}

	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bundles `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count bundles: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, sku, name, description, selling_price, is_active, created_at, updated_at
		FROM bundles
		%s
		ORDER BY name
		LIMIT $%d OFFSET $%d
	`, where, arg, arg+1)
	args = append(args, limit, offset)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list bundles: %w", err)
	}

	var rawBundles []models.Bundle
	for rows.Next() {
		bundle, err := scanBundle(rows)
		if err != nil {
			return nil, 0, err
		}
		rawBundles = append(rawBundles, *bundle)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}

	bundles := make([]models.BundleAggregate, 0, len(rawBundles))
	for i := range rawBundles {
		bundle := rawBundles[i]
		aggregate, err := r.loadAggregateTx(ctx, tx, tenantID, &bundle)
		if err != nil {
			return nil, 0, err
		}
		bundles = append(bundles, *aggregate)
	}
	return bundles, total, nil
}

func (r *BundleRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*models.BundleAggregate, error) {
	var aggregate *models.BundleAggregate
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		aggregate, err = r.FindByIDTx(ctx, tx, tenantID, id)
		return err
	})
	return aggregate, err
}

func (r *BundleRepository) FindByIDTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) (*models.BundleAggregate, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, sku, name, description, selling_price, is_active, created_at, updated_at
		FROM bundles
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	bundle, err := scanBundle(row)
	if err != nil {
		return nil, err
	}
	return r.loadAggregateTx(ctx, tx, tenantID, bundle)
}

func (r *BundleRepository) CreateTx(ctx context.Context, tx *sql.Tx, bundle *models.Bundle) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO bundles (tenant_id, sku, name, description, selling_price, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, bundle.TenantID, bundle.SKU, bundle.Name, bundle.Description, bundle.SellingPrice, bundle.IsActive).
		Scan(&bundle.ID, &bundle.CreatedAt, &bundle.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}
	return nil
}

func (r *BundleRepository) CreateItemTx(ctx context.Context, tx *sql.Tx, item *models.BundleItem) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO bundle_items (tenant_id, bundle_id, product_id, quantity)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, item.TenantID, item.BundleID, item.ProductID, item.Quantity).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create bundle item: %w", err)
	}
	return nil
}

func (r *BundleRepository) UpdateTx(ctx context.Context, tx *sql.Tx, bundle *models.Bundle) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE bundles
		SET sku = $1, name = $2, description = $3, selling_price = $4, is_active = $5
		WHERE id = $6 AND tenant_id = $7
	`, bundle.SKU, bundle.Name, bundle.Description, bundle.SellingPrice, bundle.IsActive, bundle.ID, bundle.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update bundle: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BundleRepository) ReplaceItemsTx(ctx context.Context, tx *sql.Tx, tenantID, bundleID uuid.UUID, items []models.BundleItem) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM bundle_items
		WHERE tenant_id = $1 AND bundle_id = $2
	`, tenantID, bundleID); err != nil {
		return fmt.Errorf("failed to replace bundle items: %w", err)
	}
	for i := range items {
		item := items[i]
		if err := r.CreateItemTx(ctx, tx, &item); err != nil {
			return err
		}
	}
	return nil
}

func (r *BundleRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, id)
	})
}

func (r *BundleRepository) DeleteTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE bundles
		SET is_active = FALSE
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete bundle: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BundleRepository) loadAggregateTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, bundle *models.Bundle) (*models.BundleAggregate, error) {
	items, err := r.listItemsTx(ctx, tx, tenantID, bundle.ID)
	if err != nil {
		return nil, err
	}
	return &models.BundleAggregate{Bundle: *bundle, Items: items}, nil
}

func (r *BundleRepository) listItemsTx(ctx context.Context, tx *sql.Tx, tenantID, bundleID uuid.UUID) ([]models.BundleItem, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT bi.id, bi.tenant_id, bi.bundle_id, bi.product_id, p.name, p.sku,
		       bi.quantity, bi.created_at, bi.updated_at
		FROM bundle_items bi
		JOIN products p ON p.id = bi.product_id AND p.tenant_id = bi.tenant_id
		WHERE bi.tenant_id = $1 AND bi.bundle_id = $2
		ORDER BY bi.created_at ASC
	`, tenantID, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list bundle items: %w", err)
	}
	defer rows.Close()

	var items []models.BundleItem
	for rows.Next() {
		item, err := scanBundleItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanBundle(row scanner) (*models.Bundle, error) {
	var bundle models.Bundle
	var description sql.NullString
	if err := row.Scan(
		&bundle.ID,
		&bundle.TenantID,
		&bundle.SKU,
		&bundle.Name,
		&description,
		&bundle.SellingPrice,
		&bundle.IsActive,
		&bundle.CreatedAt,
		&bundle.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if description.Valid {
		bundle.Description = &description.String
	}
	return &bundle, nil
}

func scanBundleItem(row scanner) (*models.BundleItem, error) {
	var item models.BundleItem
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.BundleID,
		&item.ProductID,
		&item.ProductName,
		&item.ProductSKU,
		&item.Quantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}
