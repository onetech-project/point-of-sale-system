package repository

import (
"context"
"database/sql"
"fmt"
)

type ProductRepository struct {
db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
return &ProductRepository{
db: db,
}
}

type ProductStock struct {
ID    string
Stock int
Price int
}

func (r *ProductRepository) GetProductStock(ctx context.Context, productID string) (*ProductStock, error) {
query := `
SELECT id, stock, price
FROM products
WHERE id = $1 AND deleted_at IS NULL
`

var product ProductStock
err := r.db.QueryRowContext(ctx, query, productID).Scan(
&product.ID,
&product.Stock,
&product.Price,
)

if err == sql.ErrNoRows {
return nil, fmt.Errorf("product not found")
}
if err != nil {
return nil, fmt.Errorf("failed to get product stock: %w", err)
}

return &product, nil
}
