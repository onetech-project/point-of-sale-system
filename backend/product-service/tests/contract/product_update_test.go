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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProductServiceForUpdate struct {
	mock.Mock
}

func (m *MockProductServiceForUpdate) GetProduct(id uuid.UUID) (*models.Product, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductServiceForUpdate) UpdateProduct(id uuid.UUID, product *models.Product) error {
	args := m.Called(id, product)
	return args.Error(0)
}

func (m *MockProductServiceForUpdate) CreateProduct(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductServiceForUpdate) GetProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, int, error) {
	args := m.Called(filters, limit, offset)
	return args.Get(0).([]*models.Product), args.Int(1), args.Error(2)
}

// T043: Contract test for PUT /products/{id} endpoint
func TestUpdateProduct_Success(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()

	existingProduct := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		SKU:         "TEST-001",
		Name:        "Old Product Name",
		SellingPrice: 29.99,
		CostPrice:   15.00,
		TaxRate:     10.0,
		StockQuantity: 100,
	}

	reqBody := map[string]interface{}{
		"sku":            "TEST-001-UPDATED",
		"name":           "Updated Product Name",
		"description":    "Updated Description",
		"selling_price":  39.99,
		"cost_price":     20.00,
		"tax_rate":       15.0,
		"stock_quantity": 150,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID.String(), bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForUpdate)
	mockService.On("GetProduct", productID).Return(existingProduct, nil)
	mockService.On("UpdateProduct", productID, mock.AnythingOfType("*models.Product")).Return(nil)

	handler := api.NewProductHandler(mockService, nil)

	err := handler.UpdateProduct(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.Equal(t, "Product updated successfully", response["message"])

	mockService.AssertExpectations(t)
}

func TestUpdateProduct_NotFound(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()

	reqBody := map[string]interface{}{
		"name":          "Updated Name",
		"selling_price": 39.99,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID.String(), bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForUpdate)
	mockService.On("GetProduct", productID).Return(nil, echo.NewHTTPError(http.StatusNotFound, "Product not found"))

	handler := api.NewProductHandler(mockService, nil)

	err := handler.UpdateProduct(c)

	assert.Error(t, err)
	httpErr := err.(*echo.HTTPError)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)

	mockService.AssertExpectations(t)
}

func TestUpdateProduct_ValidationError(t *testing.T) {
	e := echo.New()
	productID := uuid.New()
	tenantID := uuid.New()

	reqBody := map[string]interface{}{
		"name":          "", // Empty name should fail validation
		"selling_price": -10.00, // Negative price should fail
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/products/"+productID.String(), bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/products/:id")
	c.SetParamNames("id")
	c.SetParamValues(productID.String())
	c.Set("tenant_id", tenantID)

	mockService := new(MockProductServiceForUpdate)
	handler := api.NewProductHandler(mockService, nil)

	err := handler.UpdateProduct(c)

	assert.Error(t, err)
	httpErr := err.(*echo.HTTPError)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}
