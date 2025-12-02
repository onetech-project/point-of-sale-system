package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	FindAll(ctx context.Context) ([]models.Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	HasProducts(ctx context.Context, id uuid.UUID) (bool, error)
}

type categoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	query := `
		INSERT INTO categories (tenant_id, name, display_order)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		category.TenantID, category.Name, category.DisplayOrder,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
}

func (r *categoryRepository) FindAll(ctx context.Context) ([]models.Category, error) {
	query := `
		SELECT id, tenant_id, name, display_order, created_at, updated_at
		FROM categories
		ORDER BY display_order, name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var c models.Category
		err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, rows.Err()
}

func (r *categoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	query := `
		SELECT id, tenant_id, name, display_order, created_at, updated_at
		FROM categories
		WHERE id = $1
	`

	var c models.Category
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *models.Category) error {
	query := `
		UPDATE categories
		SET name = $2, display_order = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		category.ID, category.Name, category.DisplayOrder,
	).Scan(&category.UpdatedAt)
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM categories WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *categoryRepository) HasProducts(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM products WHERE category_id = $1 LIMIT 1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}
