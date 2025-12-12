package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
)

type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	FindAll(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]models.Product, error)
	FindByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Product, error)
	FindByIDWithCategory(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	UpdateStock(ctx context.Context, id uuid.UUID, newQuantity int) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	Archive(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	Restore(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	FindLowStock(ctx context.Context, tenantID uuid.UUID, threshold int) ([]models.Product, error)
	HasSalesHistory(ctx context.Context, id uuid.UUID) (bool, error)
	Count(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}) (int, error)
	CreateStockAdjustment(ctx context.Context, adjustment *models.StockAdjustment) error
}

type productRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	query := `
		INSERT INTO products (tenant_id, sku, name, description, category_id, selling_price, cost_price, tax_rate, stock_quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		product.TenantID, product.SKU, product.Name, product.Description, product.CategoryID,
		product.SellingPrice, product.CostPrice, product.TaxRate, product.StockQuantity,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (r *productRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]models.Product, error) {
	query := `
		SELECT p.id, p.tenant_id, p.sku, p.name, p.description, p.category_id, c.name as category_name,
		       p.selling_price, p.cost_price, p.tax_rate, p.stock_quantity, 
		       p.photo_path, p.photo_size, p.archived_at, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
		WHERE p.tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND p.name ILIKE $%d", argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	if categoryID, ok := filters["category_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND category_id = $%d", argCount)
		args = append(args, categoryID)
		argCount++
	}

	if lowStock, ok := filters["low_stock"].(int); ok {
		query += fmt.Sprintf(" AND stock_quantity <= $%d", argCount)
		args = append(args, lowStock)
		argCount++
	}

	if archived, ok := filters["archived"].(bool); ok {
		if archived {
			query += " AND archived_at IS NOT NULL"
		} else {
			query += " AND archived_at IS NULL"
		}
	} else {
		query += " AND archived_at IS NULL"
	}

	query += " ORDER BY p.name"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName,
			&p.SellingPrice, &p.CostPrice, &p.TaxRate, &p.StockQuantity,
			&p.PhotoPath, &p.PhotoSize, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, rows.Err()
}

func (r *productRepository) FindByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Product, error) {
	query := `
		SELECT p.id, p.tenant_id, p.sku, p.name, p.description, p.category_id, c.name as category_name,
		       p.selling_price, p.cost_price, p.tax_rate, p.stock_quantity, 
		       p.photo_path, p.photo_size, p.archived_at, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
		WHERE p.id = $1 AND p.tenant_id = $2
	`

	var p models.Product
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName,
		&p.SellingPrice, &p.CostPrice, &p.TaxRate, &p.StockQuantity,
		&p.PhotoPath, &p.PhotoSize, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *productRepository) FindByIDWithCategory(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.Product, error) {
	return r.FindByID(ctx, tenantID, id)
}

func (r *productRepository) Update(ctx context.Context, product *models.Product) error {
	query := `
		UPDATE products
		SET sku = $2, name = $3, description = $4, category_id = $5, selling_price = $6,
		    cost_price = $7, tax_rate = $8, stock_quantity = $9, photo_path = $10, 
		    photo_size = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		product.ID, product.SKU, product.Name, product.Description, product.CategoryID,
		product.SellingPrice, product.CostPrice, product.TaxRate, product.StockQuantity,
		product.PhotoPath, product.PhotoSize,
	).Scan(&product.UpdatedAt)
}

func (r *productRepository) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `DELETE FROM products WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, tenantID)
	return err
}

func (r *productRepository) Archive(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `UPDATE products SET archived_at = NOW() WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, tenantID)
	return err
}

func (r *productRepository) Restore(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `UPDATE products SET archived_at = NULL WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, tenantID)
	return err
}

func (r *productRepository) FindLowStock(ctx context.Context, tenantID uuid.UUID, threshold int) ([]models.Product, error) {
	filters := map[string]interface{}{
		"low_stock": threshold,
	}
	return r.FindAll(ctx, tenantID, filters, 100, 0)
}

func (r *productRepository) HasSalesHistory(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}

func (r *productRepository) Count(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM products WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if archived, ok := filters["archived"].(bool); ok {
		if archived {
			query += " AND archived_at IS NOT NULL"
		} else {
			query += " AND archived_at IS NULL"
		}
	} else {
		query += " AND archived_at IS NULL"
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *productRepository) UpdateStock(ctx context.Context, id uuid.UUID, newQuantity int) error {
	query := `
		UPDATE products
		SET stock_quantity = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, newQuantity, id)
	return err
}

func (r *productRepository) CreateStockAdjustment(ctx context.Context, adjustment *models.StockAdjustment) error {
	query := `
		INSERT INTO stock_adjustments (tenant_id, product_id, user_id, previous_quantity, new_quantity, reason, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, quantity_delta, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		adjustment.TenantID, adjustment.ProductID, adjustment.UserID,
		adjustment.PreviousQuantity, adjustment.NewQuantity, adjustment.Reason, adjustment.Notes,
	).Scan(&adjustment.ID, &adjustment.QuantityDelta, &adjustment.CreatedAt)
}
