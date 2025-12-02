package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
	"github.com/pos/backend/product-service/src/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CreateProductIntegrationTestSuite is the test suite for product creation workflow
type CreateProductIntegrationTestSuite struct {
	suite.Suite
	db       *sql.DB
	ctx      context.Context
	tenantID uuid.UUID
}

// SetupSuite runs once before all tests
func (suite *CreateProductIntegrationTestSuite) SetupSuite() {
	// Get database connection from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		suite.T().Skip("DATABASE_URL not set, skipping integration tests")
	}

	var err error
	suite.db, err = sql.Open("postgres", dbURL)
	require.NoError(suite.T(), err)

	suite.ctx = context.Background()
	suite.tenantID = uuid.New()

	// Set tenant context
	_, err = suite.db.Exec("SELECT set_config('app.current_tenant_id', $1, false)", suite.tenantID.String())
	require.NoError(suite.T(), err)
}

// TearDownSuite runs once after all tests
func (suite *CreateProductIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *CreateProductIntegrationTestSuite) SetupTest() {
	// Clean up products for this tenant before each test
	_, err := suite.db.Exec("DELETE FROM products WHERE tenant_id = $1", suite.tenantID)
	require.NoError(suite.T(), err)
}

// TestCreateProductFullWorkflow tests the complete product creation workflow
func (suite *CreateProductIntegrationTestSuite) TestCreateProductFullWorkflow() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test verifies the full workflow:
	// 1. Create a product with all fields
	// 2. Verify product is persisted in database
	// 3. Verify product can be retrieved by ID
	// 4. Verify product appears in product list
	// 5. Verify all fields match what was inserted

	// Setup
	productRepo := repository.NewProductRepository(suite.db)
	productService := services.NewProductService(productRepo, nil)

	// Create product
	product := &models.Product{
		TenantID:      suite.tenantID,
		SKU:           "INTEGRATION-001",
		Name:          "Integration Test Product",
		Description:   stringPtr("Full workflow test"),
		SellingPrice:  25.99,
		CostPrice:     12.50,
		TaxRate:       10.00,
		StockQuantity: 100,
	}

	err := productService.CreateProduct(product)
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, product.ID)

	// Retrieve product
	retrieved, err := productRepo.FindByIDWithCategory(suite.ctx, product.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), product.SKU, retrieved.SKU)
	assert.Equal(suite.T(), product.Name, retrieved.Name)
	assert.Equal(suite.T(), product.SellingPrice, retrieved.SellingPrice)
	assert.Equal(suite.T(), product.StockQuantity, retrieved.StockQuantity)

	// Verify in product list
	products, err := productRepo.FindAll(suite.ctx, map[string]interface{}{}, 50, 0)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), products, 1)
	assert.Equal(suite.T(), product.ID, products[0].ID)
}

// TestCreateProductWithCategory tests creating a product with category
func (suite *CreateProductIntegrationTestSuite) TestCreateProductWithCategory() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Create a category first
	// 2. Create a product assigned to that category
	// 3. Verify category_name is populated when retrieving product
	// 4. Verify category filter works
}

// TestCreateProductDuplicateSKU tests SKU uniqueness constraint
func (suite *CreateProductIntegrationTestSuite) TestCreateProductDuplicateSKU() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Create a product with SKU "UNIQUE-001"
	// 2. Attempt to create another product with same SKU
	// 3. Verify second creation fails with appropriate error
	// 4. Verify only one product exists in database
}

// TestCreateProductTenantIsolation tests multi-tenant isolation
func (suite *CreateProductIntegrationTestSuite) TestCreateProductTenantIsolation() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Create product for tenant A
	// 2. Switch tenant context to tenant B
	// 3. Attempt to retrieve product created by tenant A
	// 4. Verify product is not accessible (RLS enforcement)
	// 5. Verify tenant B can create product with same SKU (SKU uniqueness is per-tenant)
}

// TestCreateProductWithMinimalFields tests creating product with only required fields
func (suite *CreateProductIntegrationTestSuite) TestCreateProductWithMinimalFields() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Create product with only required fields (sku, name, selling_price, cost_price, stock_quantity)
	// 2. Verify optional fields are null/default values
	// 3. Verify product is fully functional
}

// TestCreateProductValidationErrors tests validation error handling
func (suite *CreateProductIntegrationTestSuite) TestCreateProductValidationErrors() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Empty SKU fails validation
	// 2. Empty name fails validation
	// 3. Negative selling_price fails validation
	// 4. Tax rate > 100 fails validation
	// 5. SKU > 50 chars fails validation
	// 6. Name > 255 chars fails validation
}

// TestCreateProductPhotoUpload tests photo upload after product creation
func (suite *CreateProductIntegrationTestSuite) TestCreateProductPhotoUpload() {
	suite.T().Skip("Implementation not yet complete - test should fail first (TDD)")

	// This test should verify:
	// 1. Create a product without photo
	// 2. Upload a photo for the product
	// 3. Verify photo_path and photo_size are updated
	// 4. Verify photo file exists on filesystem
	// 5. Verify photo can be retrieved
}

// TestSuite entry point
func TestCreateProductIntegration(t *testing.T) {
	suite.Run(t, new(CreateProductIntegrationTestSuite))
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
