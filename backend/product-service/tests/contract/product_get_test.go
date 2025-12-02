package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/api"
	"github.com/pos/backend/product-service/src/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProductServiceForGet struct {
	mock.Mock
}

func (m *MockProductServiceForGet) GetProduct(id uuid.UUID) (*models.Product, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductServiceForGet) CreateProduct(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductServiceForGet) GetProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, int, error) {
	args := m.Called(filters, limit, offset)
	return args.Get(0).([]*models.Product), args.Int(1), args.Error(2)
}

// T044: Contract test for GET /products/{id} endpoint
func TestGetProduct_Success(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()
	categoryID := uuid.New()

	product := &models.Product{
		ID:            productID,
		TenantID:      tenantID,
		SKU:           "TEST-001",
		Name:          "Test Product",
		Description:   "Test Description",
		CategoryID:    &categoryID,
		SellingPrice:  29.99,
		CostPrice:     15.00,
		TaxRate:       10.0,
		StockQuantity: 100,
		PhotoPath:     "/uploads/test.jpg",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForGet)
	mockService.On("GetProduct", productID).Return(product, nil)

	handler := api.NewProductHandler(mockService, nil)

	err := handler.GetProduct(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.Product
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, productID, response.ID)
	assert.Equal(t, "TEST-001", response.SKU)
	assert.Equal(t, "Test Product", response.Name)
	assert.Equal(t, 29.99, response.SellingPrice)
	assert.Equal(t, 100, response.StockQuantity)

	mockService.AssertExpectations(t)
}

func TestGetProduct_NotFound(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForGet)
	mockService.On("GetProduct", productID).Return(nil, echo.NewHTTPError(http.StatusNotFound, "Product not found"))

	handler := api.NewProductHandler(mockService, nil)

	err := handler.GetProduct(c)

	assert.Error(t, err)
	httpErr := err.(*echo.HTTPError)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)

	mockService.AssertExpectations(t)
}

func TestGetProduct_InvalidUUID(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/products/invalid-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid-uuid")
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForGet)
	handler := api.NewProductHandler(mockService, nil)

	err := handler.GetProduct(c)

	assert.Error(t, err)
	httpErr := err.(*echo.HTTPError)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}

func TestGetProduct_WithCategory(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()
	categoryID := uuid.New()

	product := &models.Product{
		ID:            productID,
		TenantID:      tenantID,
		SKU:           "TEST-002",
		Name:          "Product with Category",
		CategoryID:    &categoryID,
		SellingPrice:  49.99,
		CostPrice:     25.00,
		TaxRate:       10.0,
		StockQuantity: 50,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForGet)
	mockService.On("GetProduct", productID).Return(product, nil)

	handler := api.NewProductHandler(mockService, nil)

	err := handler.GetProduct(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.Product
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NotNil(t, response.CategoryID)
	assert.Equal(t, categoryID, *response.CategoryID)

	mockService.AssertExpectations(t)
}
