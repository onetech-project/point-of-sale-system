package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/pos/analytics-service/src/models"
	"github.com/rs/zerolog/log"
)

// ProductRepository handles product analytics queries
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetTopProductsByRevenue returns top N products by revenue
func (r *ProductRepository) GetTopProductsByRevenue(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.ProductRanking, error) {
	query := `
		SELECT 
			p.id as product_id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			c.name as category_name
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN guest_orders od ON od.id = oi.order_id 
			AND od.tenant_id = $1 
			AND od.status = 'COMPLETE'
			AND od.created_at BETWEEN $2 AND $3
		LEFT JOIN categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.archived_at IS NULL AND od.status = 'COMPLETE'
		GROUP BY p.id, p.name, p.sku, c.name
		HAVING SUM(oi.total_price) > 0
		ORDER BY revenue DESC
		LIMIT $4
	`

	return r.queryProducts(ctx, query, tenantID, start, end, limit)
}

// GetTopProductsByQuantity returns top N products by quantity sold
func (r *ProductRepository) GetTopProductsByQuantity(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.ProductRanking, error) {
	query := `
		SELECT 
			p.id as product_id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			c.name as category_name
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN guest_orders od ON od.id = oi.order_id 
			AND od.tenant_id = $1 
			AND od.status = 'COMPLETE'
			AND od.created_at BETWEEN $2 AND $3
		LEFT JOIN categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.archived_at IS NULL  AND od.status = 'COMPLETE'
		GROUP BY p.id, p.name, p.sku, c.name
		HAVING SUM(oi.quantity) > 0
		ORDER BY quantity_sold DESC
		LIMIT $4
	`

	return r.queryProducts(ctx, query, tenantID, start, end, limit)
}

// GetBottomProductsByRevenue returns bottom N products by revenue (excluding zero sales)
func (r *ProductRepository) GetBottomProductsByRevenue(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.ProductRanking, error) {
	query := `
		SELECT 
			p.id as product_id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			c.name as category_name
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN guest_orders od ON od.id = oi.order_id 
			AND od.tenant_id = $1 
			AND od.status = 'COMPLETE'
			AND od.created_at BETWEEN $2 AND $3
		LEFT JOIN categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.archived_at IS NULL  AND od.status = 'COMPLETE'
		GROUP BY p.id, p.name, p.sku, c.name
		HAVING SUM(oi.total_price) > 0
		ORDER BY revenue ASC
		LIMIT $4
	`

	return r.queryProducts(ctx, query, tenantID, start, end, limit)
}

// GetBottomProductsByQuantity returns bottom N products by quantity (excluding zero sales)
func (r *ProductRepository) GetBottomProductsByQuantity(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.ProductRanking, error) {
	query := `
		SELECT 
			p.id as product_id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			c.name as category_name
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN guest_orders od ON od.id = oi.order_id 
			AND od.tenant_id = $1 
			AND od.status = 'COMPLETE'
			AND od.created_at BETWEEN $2 AND $3
		LEFT JOIN categories c ON c.id = p.category_id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1 AND p.archived_at IS NULL AND od.status = 'COMPLETE'
		GROUP BY p.id, p.name, p.sku, c.name
		HAVING SUM(oi.quantity) > 0
		ORDER BY quantity_sold ASC
		LIMIT $4
	`

	return r.queryProducts(ctx, query, tenantID, start, end, limit)
}

// queryProducts is a helper function to execute product ranking queries
func (r *ProductRepository) queryProducts(ctx context.Context, query string, tenantID string, start, end time.Time, limit int) ([]models.ProductRanking, error) {
	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to query products")
		return nil, err
	}
	defer rows.Close()

	var products []models.ProductRanking
	for rows.Next() {
		var p models.ProductRanking
		var imageURL sql.NullString
		var categoryName sql.NullString

		if err := rows.Scan(&p.ProductID, &p.Name, &p.SKU, &p.QuantitySold, &p.Revenue, &categoryName); err != nil {
			log.Error().Err(err).Msg("Failed to scan product row")
			continue
		}

		if imageURL.Valid {
			p.ImageURL = imageURL.String
		}
		if categoryName.Valid {
			p.CategoryName = categoryName.String
		}

		products = append(products, p)
	}

	return products, nil
}
