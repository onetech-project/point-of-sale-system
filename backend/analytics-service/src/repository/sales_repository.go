package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/analytics-service/src/models"
	"github.com/rs/zerolog/log"
)

// SalesRepository handles sales data queries
type SalesRepository struct {
	db       *sql.DB
	timezone string
}

// NewSalesRepository creates a new sales repository
func NewSalesRepository(db *sql.DB, timezone string) *SalesRepository {
	return &SalesRepository{
		db:       db,
		timezone: timezone,
	}
}

// GetSalesMetrics calculates sales metrics for a time range with comparison to previous period
func (r *SalesRepository) GetSalesMetrics(ctx context.Context, tenantID string, start, end time.Time) (*models.SalesMetrics, error) {
	// Calculate current period metrics
	log.Debug().Str("tenant_id", tenantID).
		Time("start", start).
		Time("end", end).
		Msg("Calculating sales metrics")

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COUNT(*) as total_orders,
			COALESCE(AVG(total_amount), 0) as average_order_value
		FROM guest_orders
		WHERE tenant_id = $1 
			AND status = 'COMPLETE'
			AND (created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
	`, r.timezone)

	metrics := &models.SalesMetrics{
		StartDate: start,
		EndDate:   end,
	}

	err := r.db.QueryRowContext(ctx, query, tenantID, start, end).Scan(
		&metrics.TotalRevenue,
		&metrics.TotalOrders,
		&metrics.AverageOrderValue,
	)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get current sales metrics")
		return nil, err
	}

	// Calculate inventory value: sum of (product.cost * product.quantity)
	inventoryQuery := `
		SELECT COALESCE(SUM(cost_price * stock_quantity), 0) as inventory_value
		FROM products
		WHERE tenant_id = $1
	`

	err = r.db.QueryRowContext(ctx, inventoryQuery, tenantID).Scan(&metrics.InventoryValue)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to calculate inventory value")
		return nil, err
	}

	// Calculate previous period metrics for comparison
	duration := end.Sub(start)
	prevStart := start.Add(-duration)
	prevEnd := start.Add(-time.Second) // End just before current period starts

	err = r.db.QueryRowContext(ctx, query, tenantID, prevStart, prevEnd).Scan(
		&metrics.PreviousRevenue,
		&metrics.PreviousOrders,
		&metrics.PreviousAOV,
	)
	if err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to get previous sales metrics, using zero values")
		// Continue with zero values for previous period
	}

	// Calculate percentage changes
	if metrics.PreviousRevenue > 0 {
		metrics.RevenueChange = ((metrics.TotalRevenue - metrics.PreviousRevenue) / metrics.PreviousRevenue) * 100
	} else if metrics.TotalRevenue > 0 {
		metrics.RevenueChange = 100 // New sales, 100% increase
	}

	if metrics.PreviousOrders > 0 {
		metrics.OrdersChange = ((float64(metrics.TotalOrders) - float64(metrics.PreviousOrders)) / float64(metrics.PreviousOrders)) * 100
	} else if metrics.TotalOrders > 0 {
		metrics.OrdersChange = 100
	}

	if metrics.PreviousAOV > 0 {
		metrics.AOVChange = ((metrics.AverageOrderValue - metrics.PreviousAOV) / metrics.PreviousAOV) * 100
	} else if metrics.AverageOrderValue > 0 {
		metrics.AOVChange = 100
	}

	return metrics, nil
}

// GetDailySales returns daily sales data for charting
func (r *SalesRepository) GetDailySales(ctx context.Context, tenantID string, start, end time.Time) ([]models.DailySalesData, error) {
	query := fmt.Sprintf(`
		SELECT 
			DATE((created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s') as date,
			COALESCE(SUM(total_amount), 0) as revenue,
			COUNT(*) as orders
		FROM guest_orders
		WHERE tenant_id = $1 
			AND status = 'COMPLETE'
			AND (created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		GROUP BY DATE((created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s')
		ORDER BY date ASC
	`, r.timezone, r.timezone, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get daily sales")
		return nil, err
	}
	defer rows.Close()

	var dailySales []models.DailySalesData
	for rows.Next() {
		var data models.DailySalesData
		if err := rows.Scan(&data.Date, &data.Revenue, &data.Orders); err != nil {
			log.Error().Err(err).Msg("Failed to scan daily sales row")
			continue
		}
		dailySales = append(dailySales, data)
	}

	return dailySales, nil
}

// GetCategoryBreakdown returns sales breakdown by category
func (r *SalesRepository) GetCategoryBreakdown(ctx context.Context, tenantID string, start, end time.Time) ([]models.CategorySales, error) {
	query := fmt.Sprintf(`
		SELECT 
			c.id as category_id,
			c.name as category_name,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT go.id) as order_count
		FROM categories c
		LEFT JOIN products p ON p.category_id = c.id AND p.tenant_id = c.tenant_id
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN guest_orders go ON go.id = oi.order_id 
			AND go.tenant_id = $1 
			AND go.status = 'COMPLETE'
			AND (go.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		WHERE c.tenant_id = $1
		GROUP BY c.id, c.name
		HAVING SUM(oi.total_price) > 0
		ORDER BY revenue DESC
	`, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get category breakdown")
		return nil, err
	}
	defer rows.Close()

	var categories []models.CategorySales
	var totalRevenue float64

	// First pass: collect data and calculate total
	for rows.Next() {
		var cat models.CategorySales
		if err := rows.Scan(&cat.CategoryID, &cat.CategoryName, &cat.Revenue, &cat.OrderCount); err != nil {
			log.Error().Err(err).Msg("Failed to scan category sales row")
			continue
		}
		totalRevenue += cat.Revenue
		categories = append(categories, cat)
	}

	// Second pass: calculate percentages
	for i := range categories {
		if totalRevenue > 0 {
			categories[i].Percentage = (categories[i].Revenue / totalRevenue) * 100
		}
	}

	return categories, nil
}

// GetSalesTrend returns time series data for sales revenue and order count
// Uses generate_series to ensure complete date ranges even for dates with no sales
func (r *SalesRepository) GetSalesTrend(ctx context.Context, tenantID string, start, end time.Time, granularity string) ([]models.TimeSeriesData, []models.TimeSeriesData, error) {
	// Determine the date truncation and interval based on granularity
	var dateTrunc, interval string
	switch granularity {
	case "daily":
		dateTrunc = "day"
		interval = "1 day"
	case "weekly":
		dateTrunc = "week"
		interval = "7 days"
	case "monthly":
		dateTrunc = "month"
		interval = "1 month"
	case "quarterly":
		dateTrunc = "quarter"
		interval = "3 months"
	case "yearly":
		dateTrunc = "year"
		interval = "1 year"
	default:
		dateTrunc = "day"
		interval = "1 day"
	}

	// Query with generate_series to fill gaps
	query := fmt.Sprintf(`
		WITH date_series AS (
			SELECT generate_series(
				date_trunc($4, $2::timestamp),
				date_trunc($4, $3::timestamp),
				$5::interval
			)::date AS date
		)
		SELECT 
			ds.date,
			COALESCE(SUM(go.total_amount), 0) as revenue,
			COUNT(go.id) as orders
		FROM date_series ds
		LEFT JOIN guest_orders go ON 
			date_trunc($4, (go.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s') = ds.date
			AND go.tenant_id = $1
			AND go.status = 'COMPLETE'
			AND (go.created_at AT TIME ZONE 'UTC') AT TIME ZONE '%s' BETWEEN $2 AND $3
		GROUP BY ds.date
		ORDER BY ds.date ASC
	`, r.timezone, r.timezone)

	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end, dateTrunc, interval)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Str("granularity", granularity).Msg("Failed to get sales trend")
		return nil, nil, err
	}
	defer rows.Close()

	var revenueData []models.TimeSeriesData
	var ordersData []models.TimeSeriesData

	for rows.Next() {
		var date time.Time
		var revenue float64
		var orders int

		if err := rows.Scan(&date, &revenue, &orders); err != nil {
			log.Error().Err(err).Msg("Failed to scan sales trend row")
			continue
		}

		// Format date and label based on granularity
		dateStr := date.Format("2006-01-02")
		var label string
		switch granularity {
		case "daily":
			label = date.Format("Jan 02")
		case "weekly":
			label = "Week of " + date.Format("Jan 02")
		case "monthly":
			label = date.Format("Jan 2006")
		case "quarterly":
			quarter := (int(date.Month())-1)/3 + 1
			label = date.Format("2006") + " Q" + string(rune('0'+quarter))
		case "yearly":
			label = date.Format("2006")
		default:
			label = date.Format("Jan 02")
		}

		revenueData = append(revenueData, models.TimeSeriesData{
			Date:  dateStr,
			Label: label,
			Value: revenue,
		})

		ordersData = append(ordersData, models.TimeSeriesData{
			Date:  dateStr,
			Label: label,
			Value: float64(orders),
		})
	}

	return revenueData, ordersData, nil
}
