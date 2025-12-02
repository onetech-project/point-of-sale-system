package unit

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductRepositoryCreate tests the Create method of ProductRepository
func TestProductRepositoryCreate(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	tests := []struct {
		name      string
		product   *models.Product
		mockSetup func(sqlmock.Sqlmock, *models.Product)
		wantErr   bool
	}{
		{
			name: "successful product creation",
			product: &models.Product{
				TenantID:      uuid.New(),
				SKU:           "PROD-001",
				Name:          "Test Product",
				Description:   stringPtr("Test description"),
				SellingPrice:  15.99,
				CostPrice:     8.50,
				TaxRate:       10.00,
				StockQuantity: 50,
			},
			mockSetup: func(mock sqlmock.Sqlmock, p *models.Product) {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(uuid.New(), "2024-01-01 00:00:00", "2024-01-01 00:00:00")
				
				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs(p.TenantID, p.SKU, p.Name, p.Description, p.CategoryID,
						p.SellingPrice, p.CostPrice, p.TaxRate, p.StockQuantity).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "successful creation with minimal fields",
			product: &models.Product{
				TenantID:      uuid.New(),
				SKU:           "PROD-002",
				Name:          "Minimal Product",
				SellingPrice:  10.00,
				CostPrice:     5.00,
				StockQuantity: 0,
			},
			mockSetup: func(mock sqlmock.Sqlmock, p *models.Product) {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(uuid.New(), "2024-01-01 00:00:00", "2024-01-01 00:00:00")
				
				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs(p.TenantID, p.SKU, p.Name, p.Description, p.CategoryID,
						p.SellingPrice, p.CostPrice, p.TaxRate, p.StockQuantity).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "fail with database error",
			product: &models.Product{
				TenantID:      uuid.New(),
				SKU:           "PROD-003",
				Name:          "Error Product",
				SellingPrice:  10.00,
				CostPrice:     5.00,
				StockQuantity: 0,
			},
			mockSetup: func(mock sqlmock.Sqlmock, p *models.Product) {
				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs(p.TenantID, p.SKU, p.Name, p.Description, p.CategoryID,
						p.SellingPrice, p.CostPrice, p.TaxRate, p.StockQuantity).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
		{
			name: "fail with duplicate SKU",
			product: &models.Product{
				TenantID:      uuid.New(),
				SKU:           "DUPLICATE-SKU",
				Name:          "Duplicate Product",
				SellingPrice:  10.00,
				CostPrice:     5.00,
				StockQuantity: 0,
			},
			mockSetup: func(mock sqlmock.Sqlmock, p *models.Product) {
				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs(p.TenantID, p.SKU, p.Name, p.Description, p.CategoryID,
						p.SellingPrice, p.CostPrice, p.TaxRate, p.StockQuantity).
					WillReturnError(sql.ErrNoRows) // Simulate unique constraint violation
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a TDD test - it should FAIL until implementation exists
			t.Skip("Implementation not yet complete - test should fail first (TDD)")

			// Setup mock expectations
			if tt.mockSetup != nil {
				tt.mockSetup(mock, tt.product)
			}

			// Execute
			err := repo.Create(ctx, tt.product)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.product.ID)
				assert.NotZero(t, tt.product.CreatedAt)
				assert.NotZero(t, tt.product.UpdatedAt)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestProductRepositoryCreate_WithCategory tests creating a product with category
func TestProductRepositoryCreate_WithCategory(t *testing.T) {
	t.Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Product can be created with category_id
	// 2. Category_id is properly stored in database
	// 3. Foreign key constraint is respected
}

// TestProductRepositoryCreate_TenantIsolation tests RLS tenant isolation
func TestProductRepositoryCreate_TenantIsolation(t *testing.T) {
	t.Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Product is created only for the specified tenant
	// 2. Product from one tenant is not accessible by another tenant
	// 3. app.current_tenant_id session variable is properly set
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
