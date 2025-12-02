package contract

import (
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

type MockProductServiceForInventory struct {
	mock.Mock
}

func (m *MockProductServiceForInventory) GetProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, int, error) {
	args := m.Called(filters, limit, offset)
	return args.Get(0).([]*models.Product), args.Int(1), args.Error(2)
}

func (m *MockProductServiceForInventory) CreateProduct(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductServiceForInventory) GetProduct(id uuid.UUID) (*models.Product, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

// T059: Contract test for GET /products with low_stock filter
func TestGetProducts_LowStockFilter(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()

	tests := []struct {
		name           string
		queryParams    map[string]string
		mockSetup      func(*MockProductServiceForInventory)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "filter low stock products",
			queryParams: map[string]string{
				"low_stock": "true",
			},
			mockSetup: func(service *MockProductServiceForInventory) {
				lowStockProducts := []*models.Product{
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "LOW-STOCK-001",
						Name:          "Low Stock Product 1",
						SellingPrice:  29.99,
						StockQuantity: 5,
					},
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "LOW-STOCK-002",
						Name:          "Low Stock Product 2",
						SellingPrice:  19.99,
						StockQuantity: 3,
					},
				}
				service.On("GetProducts", mock.MatchedBy(func(f map[string]interface{}) bool {
					return f["low_stock"] == true
				}), 50, 0).Return(lowStockProducts, len(lowStockProducts), nil)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				products := response["products"].([]interface{})
				assert.Equal(t, 2, len(products))
				
				for _, p := range products {
					product := p.(map[string]interface{})
					stockQty := int(product["stock_quantity"].(float64))
					assert.LessOrEqual(t, stockQty, 10, "Stock quantity should be low")
				}
			},
		},
		{
			name: "filter out of stock products",
			queryParams: map[string]string{
				"out_of_stock": "true",
			},
			mockSetup: func(service *MockProductServiceForInventory) {
				outOfStockProducts := []*models.Product{
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "OUT-STOCK-001",
						Name:          "Out of Stock Product",
						SellingPrice:  39.99,
						StockQuantity: 0,
					},
				}
				service.On("GetProducts", mock.MatchedBy(func(f map[string]interface{}) bool {
					return f["out_of_stock"] == true
				}), 50, 0).Return(outOfStockProducts, len(outOfStockProducts), nil)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				products := response["products"].([]interface{})
				assert.Equal(t, 1, len(products))
				
				product := products[0].(map[string]interface{})
				assert.Equal(t, float64(0), product["stock_quantity"])
			},
		},
		{
			name: "filter with custom threshold",
			queryParams: map[string]string{
				"low_stock_threshold": "20",
			},
			mockSetup: func(service *MockProductServiceForInventory) {
				products := []*models.Product{
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "THRESHOLD-001",
						Name:          "Below Threshold",
						SellingPrice:  29.99,
						StockQuantity: 15,
					},
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "THRESHOLD-002",
						Name:          "Below Threshold 2",
						SellingPrice:  19.99,
						StockQuantity: 10,
					},
				}
				service.On("GetProducts", mock.MatchedBy(func(f map[string]interface{}) bool {
					threshold, ok := f["low_stock_threshold"].(int)
					return ok && threshold == 20
				}), 50, 0).Return(products, len(products), nil)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				products := response["products"].([]interface{})
				assert.Equal(t, 2, len(products))
				
				for _, p := range products {
					product := p.(map[string]interface{})
					stockQty := int(product["stock_quantity"].(float64))
					assert.LessOrEqual(t, stockQty, 20)
				}
			},
		},
		{
			name: "no low stock products",
			queryParams: map[string]string{
				"low_stock": "true",
			},
			mockSetup: func(service *MockProductServiceForInventory) {
				service.On("GetProducts", mock.MatchedBy(func(f map[string]interface{}) bool {
					return f["low_stock"] == true
				}), 50, 0).Return([]*models.Product{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				products := response["products"].([]interface{})
				assert.Equal(t, 0, len(products))
				assert.Equal(t, float64(0), response["total"])
			},
		},
		{
			name: "combine low_stock with category filter",
			queryParams: map[string]string{
				"low_stock":   "true",
				"category_id": uuid.New().String(),
			},
			mockSetup: func(service *MockProductServiceForInventory) {
				products := []*models.Product{
					{
						ID:            uuid.New(),
						TenantID:      tenantID,
						SKU:           "CAT-LOW-001",
						Name:          "Category Low Stock",
						SellingPrice:  29.99,
						StockQuantity: 4,
					},
				}
				service.On("GetProducts", mock.MatchedBy(func(f map[string]interface{}) bool {
					return f["low_stock"] == true && f["category_id"] != nil
				}), 50, 0).Return(products, len(products), nil)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				products := response["products"].([]interface{})
				assert.GreaterOrEqual(t, len(products), 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockProductServiceForInventory)
			tt.mockSetup(mockService)

			handler := api.NewProductHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/products", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("tenant_id", tenantID)

			err := handler.GetProducts(c)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetProducts_StockLevelCombinations(t *testing.T) {
	e := echo.New()
	tenantID := uuid.New()

	mockService := new(MockProductServiceForInventory)
	
	allProducts := []*models.Product{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			SKU:           "STOCK-HIGH",
			Name:          "High Stock Product",
			SellingPrice:  29.99,
			StockQuantity: 500,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			SKU:           "STOCK-NORMAL",
			Name:          "Normal Stock Product",
			SellingPrice:  19.99,
			StockQuantity: 50,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			SKU:           "STOCK-LOW",
			Name:          "Low Stock Product",
			SellingPrice:  39.99,
			StockQuantity: 5,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			SKU:           "STOCK-OUT",
			Name:          "Out of Stock Product",
			SellingPrice:  49.99,
			StockQuantity: 0,
		},
	}

	mockService.On("GetProducts", mock.Anything, 50, 0).Return(allProducts, len(allProducts), nil)

	handler := api.NewProductHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", tenantID)

	err := handler.GetProducts(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	
	products := response["products"].([]interface{})
	assert.Equal(t, 4, len(products))

	mockService.AssertExpectations(t)
}
