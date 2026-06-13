package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/analytics-service/src/models"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type InventoryAnalyticsRepository struct {
	db       *sql.DB
	timezone string
}

func NewInventoryAnalyticsRepository(db *sql.DB, timezone string) *InventoryAnalyticsRepository {
	return &InventoryAnalyticsRepository{
		db:       db,
		timezone: timezone,
	}
}

func (r *InventoryAnalyticsRepository) GetProductProfitability(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.ProductProfitability, error) {
	query := fmt.Sprintf(`
		SELECT 
			p.id as product_id,
			p.name as product_name,
			COALESCE(SUM(s.selling_price * s.quantity), 0) as total_revenue,
			COALESCE(SUM(s.total_cogs), 0) as total_cogs,
			COALESCE(SUM(s.selling_price*s.quantity - s.total_cogs), 0) as gross_profit,
			CASE 
				WHEN SUM(s.selling_price * s.quantity) > 0 
				THEN (SUM(s.selling_price*s.quantity - s.total_cogs) / SUM(s.selling_price * s.quantity)) * 100
				ELSE 0 
			END as gross_margin_pct,
			COALESCE(SUM(s.quantity), 0) as total_quantity_sold,
			CASE 
				WHEN SUM(s.quantity) > 0 
				THEN SUM(s.selling_price * s.quantity) / SUM(s.quantity)
				ELSE 0 
			END as avg_selling_price,
			CASE 
				WHEN SUM(s.quantity) > 0 
				THEN SUM(s.total_cogs) / SUM(s.quantity)
				ELSE 0 
			END as avg_cogs_per_unit,
			COALESCE(MAX(s.recipe_version), 0) as recipe_version
		FROM order_item_cost_snapshots s
		JOIN products p ON s.product_id = p.id AND p.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1
			AND s.product_id IS NOT NULL
			AND (s.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		GROUP BY p.id, p.name
		ORDER BY gross_profit DESC
		LIMIT $4
	`, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get product profitability")
		return nil, err
	}
	defer rows.Close()

	var results []models.ProductProfitability
	for rows.Next() {
		var p models.ProductProfitability
		err := rows.Scan(
			&p.ProductID,
			&p.ProductName,
			&p.TotalRevenue,
			&p.TotalCOGS,
			&p.GrossProfit,
			&p.GrossMarginPct,
			&p.TotalQuantitySold,
			&p.AvgSellingPrice,
			&p.AvgCOGSPerUnit,
			&p.RecipeVersion,
		)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan product profitability row")
			return nil, err
		}
		results = append(results, p)
	}

	return results, nil
}

func (r *InventoryAnalyticsRepository) GetIngredientUsage(ctx context.Context, tenantID string, start, end time.Time) ([]models.IngredientUsage, error) {
	query := fmt.Sprintf(`
		SELECT 
			i.id as ingredient_id,
			i.name as ingredient_name,
			COALESCE(SUM(sm.quantity_base), 0) as total_consumed,
			u.code as base_unit,
			COALESCE(SUM(sm.total_cost), 0) as total_cost,
			CASE 
				WHEN SUM(sm.quantity_base) > 0 
				THEN SUM(sm.total_cost) / SUM(sm.quantity_base)
				ELSE 0 
			END as avg_cost_per_unit,
			COALESCE(i.current_stock_base, 0) as current_stock
		FROM stock_movements sm
		JOIN ingredients i ON sm.ingredient_id = i.id AND i.tenant_id = sm.tenant_id
		JOIN uoms u ON i.base_uom_id = u.id
		WHERE sm.tenant_id = $1
			AND sm.movement_type = 'SALE_CONSUMPTION'
			AND (sm.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		GROUP BY i.id, i.name, u.code, i.current_stock_base
		ORDER BY total_cost DESC
	`, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get ingredient usage")
		return nil, err
	}
	defer rows.Close()

	var results []models.IngredientUsage
	for rows.Next() {
		var u models.IngredientUsage
		err := rows.Scan(
			&u.IngredientID,
			&u.IngredientName,
			&u.TotalConsumed,
			&u.BaseUnit,
			&u.TotalCost,
			&u.AvgCostPerUnit,
			&u.CurrentStock,
		)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan ingredient usage row")
			return nil, err
		}
		results = append(results, u)
	}

	return results, nil
}

func (r *InventoryAnalyticsRepository) GetIngredientDailyBurnRate(ctx context.Context, tenantID string, start, end time.Time) ([]models.DailyBurnRate, error) {
	query := fmt.Sprintf(`
		SELECT 
			(sm.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s'::date as date,
			sm.ingredient_id,
			COALESCE(SUM(sm.quantity_base), 0) as quantity_used
		FROM stock_movements sm
		WHERE sm.tenant_id = $1
			AND sm.movement_type = 'SALE_CONSUMPTION'
			AND (sm.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		GROUP BY date, sm.ingredient_id
		ORDER BY date, sm.ingredient_id
	`, r.timezone, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get ingredient daily burn rate")
		return nil, err
	}
	defer rows.Close()

	var results []models.DailyBurnRate
	for rows.Next() {
		var d models.DailyBurnRate
		err := rows.Scan(&d.Date, &d.IngredientID, &d.QuantityUsed)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan daily burn rate row")
			return nil, err
		}
		results = append(results, d)
	}

	return results, nil
}

func (r *InventoryAnalyticsRepository) GetProductSalesVelocity(ctx context.Context, tenantID string, days int) (map[uuid.UUID]decimal.Decimal, error) {
	start := time.Now().AddDate(0, 0, -days)
	query := fmt.Sprintf(`
		SELECT 
			s.product_id,
			COALESCE(SUM(s.quantity), 0) / $2 as daily_velocity
		FROM order_item_cost_snapshots s
		WHERE s.tenant_id = $1
			AND s.product_id IS NOT NULL
			AND (s.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' >= $3
		GROUP BY s.product_id
	`, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, days, start)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get product sales velocity")
		return nil, err
	}
	defer rows.Close()

	velocity := make(map[uuid.UUID]decimal.Decimal)
	for rows.Next() {
		var productID uuid.UUID
		var dailyVelocity decimal.Decimal
		if err := rows.Scan(&productID, &dailyVelocity); err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan velocity row")
			return nil, err
		}
		velocity[productID] = dailyVelocity
	}

	return velocity, nil
}

func (r *InventoryAnalyticsRepository) GetActiveRecipesWithCosts(ctx context.Context, tenantID string) (map[uuid.UUID]*models.ProductProfitability, error) {
	query := `
		SELECT 
			p.id as product_id,
			p.name as product_name,
			p.selling_price,
			COALESCE(r.version, 0) as recipe_version
		FROM products p
		LEFT JOIN recipes r ON r.product_id = p.id AND r.tenant_id = p.tenant_id AND r.is_active = true
		WHERE p.tenant_id = $1
			AND p.archived_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get active recipes with costs")
		return nil, err
	}
	defer rows.Close()

	results := make(map[uuid.UUID]*models.ProductProfitability)
	for rows.Next() {
		var p models.ProductProfitability
		var sellingPrice decimal.Decimal
		err := rows.Scan(&p.ProductID, &p.ProductName, &sellingPrice, &p.RecipeVersion)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan recipe row")
			return nil, err
		}
		p.AvgSellingPrice = sellingPrice
		results[p.ProductID] = &p
	}

	return results, nil
}

func (r *InventoryAnalyticsRepository) GetBundleItems(ctx context.Context, tenantID string) (map[uuid.UUID][]uuid.UUID, error) {
	query := `
		SELECT 
			b.id as bundle_id,
			bi.product_id
		FROM bundles b
		JOIN bundle_items bi ON bi.bundle_id = b.id
		WHERE b.tenant_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get bundle items")
		return nil, err
	}
	defer rows.Close()

	bundleItems := make(map[uuid.UUID][]uuid.UUID)
	for rows.Next() {
		var bundleID, productID uuid.UUID
		if err := rows.Scan(&bundleID, &productID); err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan bundle item row")
			return nil, err
		}
		bundleItems[bundleID] = append(bundleItems[bundleID], productID)
	}

	return bundleItems, nil
}
