package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/api"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) CreateProduct(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductService) GetProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, int, error) {
	args := m.Called(filters, limit, offset)
	return args.Get(0).([]*models.Product), args.Int(1), args.Error(2)
}

func TestCreateProduct_Success(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()
	
	reqBody := map[string]interface{}{
		"sku":           "TEST-001",
		"name":          "Test Product",
		"description":   "Test Description",
		"selling_price": 29.99,
		"cost_price":    15.00,
		"tax_rate":      10.0,
		"stock_quantity": 100,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	c.Set("tenant_id", tenantID.String())
	
	mockService := new(MockProductService)
	mockService.On("CreateProduct", mock.AnythingOfType("*models.Product")).Return(nil)
	
	handler := api.NewProductHandler(mockService)
	
	err := handler.CreateProduct(c)
	
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	
	var response models.Product
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "TEST-001", response.SKU)
	assert.Equal(t, "Test Product", response.Name)
}

func TestCreateProduct_SKUConflict(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()
	
	reqBody := map[string]interface{}{
		"sku":           "DUPLICATE-SKU",
		"name":          "Duplicate Product",
		"selling_price": 29.99,
		"cost_price":    15.00,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	c.Set("tenant_id", tenantID.String())
	
	mockService := new(MockProductService)
	mockService.On("CreateProduct", mock.AnythingOfType("*models.Product")).Return(
		services.ErrSKUExists,
	)
	
	handler := api.NewProductHandler(mockService)
	
	handler.CreateProduct(c)
	
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateProduct_MissingTenantID(t *testing.T) {
	e := echo.New()
	
	reqBody := map[string]interface{}{
		"sku":           "TEST-002",
		"name":          "Test Product",
		"selling_price": 29.99,
		"cost_price":    15.00,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	// No tenant_id set
	
	mockService := new(MockProductService)
	handler := api.NewProductHandler(mockService)
	
	handler.CreateProduct(c)
	
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateProduct_InvalidRequestBody(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()
	
	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	c.Set("tenant_id", tenantID.String())
	
	mockService := new(MockProductService)
	handler := api.NewProductHandler(mockService)
	
	handler.CreateProduct(c)
	
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
