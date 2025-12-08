package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pos/backend/product-service/src/models"
)

type CatalogService struct {
	db *sql.DB
}

func NewCatalogService(db *sql.DB) *CatalogService {
	return &CatalogService{
		db: db,
	}
}

func (s *CatalogService) GetPublicCatalog(ctx context.Context, tenantID, category string, availableOnly bool) ([]models.PublicProduct, error) {
	query := `
SELECT 
    p.id, 
    p.name, 
    p.description, 
    p.selling_price, 
    p.photo_path, 
    p.category_id, 
    c.name as category_name, 
    p.sku, 
    p.stock_quantity,
    COALESCE(
        p.stock_quantity - (
            SELECT COALESCE(SUM(ir.quantity), 0)
            FROM inventory_reservations ir
            WHERE ir.product_id = p.id AND ir.status = 'active'
        ), 0
    ) as available_stock
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
WHERE p.tenant_id = $1 
    AND p.archived_at IS NULL
`
	args := []interface{}{tenantID}
	argCount := 1

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		args = append(args, category)
	}

	// Filter by available stock using subquery
	if availableOnly {
		query += `
    AND (p.stock_quantity - COALESCE((
        SELECT SUM(ir.quantity)
        FROM inventory_reservations ir
        WHERE ir.product_id = p.id AND ir.status = 'active'
    ), 0)) > 0
`
	}

	query += " ORDER BY p.name ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.PublicProduct
	for rows.Next() {
		var p models.PublicProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.ImageURL,
			&p.CategoryID, &p.CategoryName, &p.SKU, &p.Stock, &p.AvailableStock,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		p.IsAvailable = p.AvailableStock > 0
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return products, nil
}
