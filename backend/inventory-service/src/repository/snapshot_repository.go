package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
)

type SnapshotRepository struct {
	db *sql.DB
}

func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

func (r *SnapshotRepository) CreateTx(ctx context.Context, tx *sql.Tx, snapshot *models.OrderItemCostSnapshot) error {
	pricingSnapshot := snapshot.PricingSnapshot
	if len(pricingSnapshot) == 0 {
		pricingSnapshot = json.RawMessage(`{}`)
	}
	componentCosts := snapshot.ComponentCosts
	if len(componentCosts) == 0 {
		componentCosts = json.RawMessage(`[]`)
	}
	itemType := snapshot.ItemType
	if itemType == "" {
		itemType = "product"
	}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO order_item_cost_snapshots (
			tenant_id, order_id, order_item_id, item_type, product_id, bundle_id,
			recipe_id, recipe_version,
			quantity, material_cost, fixed_overhead_cost, percentage_overhead_cost,
			total_cogs, selling_price, gross_profit, gross_margin_percentage,
			discount_amount, pricing_snapshot, component_costs
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at
	`, snapshot.TenantID, snapshot.OrderID, snapshot.OrderItemID, itemType, snapshot.ProductID,
		snapshot.BundleID, snapshot.RecipeID, snapshot.RecipeVersion, snapshot.Quantity, snapshot.MaterialCost,
		snapshot.FixedOverheadCost, snapshot.PercentageOverheadCost, snapshot.TotalCOGS,
		snapshot.SellingPrice, snapshot.GrossProfit, snapshot.GrossMarginPercentage,
		snapshot.DiscountAmount, pricingSnapshot, componentCosts).Scan(&snapshot.ID, &snapshot.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create order item cost snapshot: %w", err)
	}
	snapshot.ItemType = itemType
	snapshot.PricingSnapshot = pricingSnapshot
	snapshot.ComponentCosts = componentCosts
	return nil
}

func (r *SnapshotRepository) ExistsByOrderItemTx(ctx context.Context, tx *sql.Tx, tenantID, orderItemID uuid.UUID) (bool, error) {
	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM order_item_cost_snapshots
			WHERE tenant_id = $1 AND order_item_id = $2
		)
	`, tenantID, orderItemID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check order item cost snapshot: %w", err)
	}
	return exists, nil
}

func (r *SnapshotRepository) ListByOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]models.OrderItemCostSnapshot, error) {
	var snapshots []models.OrderItemCostSnapshot
	err := WithTenantTx(ctx, r.db, tenantID, nil, func(tx *sql.Tx) error {
		var err error
		snapshots, err = r.ListByOrderTx(ctx, tx, tenantID, orderID)
		return err
	})
	return snapshots, err
}

func (r *SnapshotRepository) ListByOrderTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uuid.UUID) ([]models.OrderItemCostSnapshot, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, tenant_id, order_id, order_item_id, item_type, product_id, bundle_id,
		       recipe_id, recipe_version, quantity, material_cost, fixed_overhead_cost,
		       percentage_overhead_cost, total_cogs, selling_price, gross_profit,
		       gross_margin_percentage, discount_amount, pricing_snapshot, component_costs,
		       created_at
		FROM order_item_cost_snapshots
		WHERE tenant_id = $1 AND order_id = $2
		ORDER BY created_at ASC, order_item_id ASC
	`, tenantID, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order item cost snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []models.OrderItemCostSnapshot
	for rows.Next() {
		snapshot, err := scanOrderItemCostSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, *snapshot)
	}
	return snapshots, rows.Err()
}

func scanOrderItemCostSnapshot(row scanner) (*models.OrderItemCostSnapshot, error) {
	var snapshot models.OrderItemCostSnapshot
	var productID, bundleID, recipeID sql.NullString
	var pricingSnapshot, componentCosts []byte
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.TenantID,
		&snapshot.OrderID,
		&snapshot.OrderItemID,
		&snapshot.ItemType,
		&productID,
		&bundleID,
		&recipeID,
		&snapshot.RecipeVersion,
		&snapshot.Quantity,
		&snapshot.MaterialCost,
		&snapshot.FixedOverheadCost,
		&snapshot.PercentageOverheadCost,
		&snapshot.TotalCOGS,
		&snapshot.SellingPrice,
		&snapshot.GrossProfit,
		&snapshot.GrossMarginPercentage,
		&snapshot.DiscountAmount,
		&pricingSnapshot,
		&componentCosts,
		&snapshot.CreatedAt,
	); err != nil {
		return nil, err
	}
	if productID.Valid {
		parsed, err := uuid.Parse(productID.String)
		if err != nil {
			return nil, err
		}
		snapshot.ProductID = &parsed
	}
	if bundleID.Valid {
		parsed, err := uuid.Parse(bundleID.String)
		if err != nil {
			return nil, err
		}
		snapshot.BundleID = &parsed
	}
	if recipeID.Valid {
		parsed, err := uuid.Parse(recipeID.String)
		if err != nil {
			return nil, err
		}
		snapshot.RecipeID = &parsed
	}
	if len(pricingSnapshot) == 0 {
		pricingSnapshot = []byte(`{}`)
	}
	if len(componentCosts) == 0 {
		componentCosts = []byte(`[]`)
	}
	snapshot.PricingSnapshot = json.RawMessage(pricingSnapshot)
	snapshot.ComponentCosts = json.RawMessage(componentCosts)
	return &snapshot, nil
}
