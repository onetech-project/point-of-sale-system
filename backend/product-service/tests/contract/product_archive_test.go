package contract

import (
"net/http"
"net/http/httptest"
"testing"

"github.com/google/uuid"
"github.com/labstack/echo/v4"
"github.com/pos/backend/product-service/api"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/mock"
)

type MockProductServiceForArchive struct {
mock.Mock
}

func (m *MockProductServiceForArchive) ArchiveProduct(id uuid.UUID) error {
args := m.Called(id)
return args.Error(0)
}

func (m *MockProductServiceForArchive) RestoreProduct(id uuid.UUID) error {
args := m.Called(id)
return args.Error(0)
}

// T088: Contract test for PATCH /products/{id}/archive endpoint
func TestArchiveProduct_Success(t *testing.T) {
e := echo.New()
productID := uuid.New()
tenantID := uuid.New()

req := httptest.NewRequest(http.MethodPatch, "/products/"+productID.String()+"/archive", nil)
rec := httptest.NewRecorder()
c := e.NewContext(req, rec)
c.SetPath("/products/:id/archive")
c.SetParamNames("id")
c.SetParamValues(productID.String())
c.Set("tenant_id", tenantID)

mockService := new(MockProductServiceForArchive)
mockService.On("ArchiveProduct", productID).Return(nil)

handler := api.NewProductHandler(mockService, nil)

err := handler.ArchiveProduct(c)

assert.NoError(t, err)
assert.Equal(t, http.StatusOK, rec.Code)

mockService.AssertExpectations(t)
}

func TestArchiveProduct_NotFound(t *testing.T) {
e := echo.New()
productID := uuid.New()
tenantID := uuid.New()

req := httptest.NewRequest(http.MethodPatch, "/products/"+productID.String()+"/archive", nil)
rec := httptest.NewRecorder()
c := e.NewContext(req, rec)
c.SetPath("/products/:id/archive")
c.SetParamNames("id")
c.SetParamValues(productID.String())
c.Set("tenant_id", tenantID)

mockService := new(MockProductServiceForArchive)
mockService.On("ArchiveProduct", productID).Return(echo.NewHTTPError(http.StatusNotFound, "Product not found"))

handler := api.NewProductHandler(mockService, nil)

err := handler.ArchiveProduct(c)

assert.Error(t, err)
httpErr := err.(*echo.HTTPError)
assert.Equal(t, http.StatusNotFound, httpErr.Code)

mockService.AssertExpectations(t)
}

// T089: Contract test for PATCH /products/{id}/restore endpoint
func TestRestoreProduct_Success(t *testing.T) {
e := echo.New()
productID := uuid.New()
tenantID := uuid.New()

req := httptest.NewRequest(http.MethodPatch, "/products/"+productID.String()+"/restore", nil)
rec := httptest.NewRecorder()
c := e.NewContext(req, rec)
c.SetPath("/products/:id/restore")
c.SetParamNames("id")
c.SetParamValues(productID.String())
c.Set("tenant_id", tenantID)

mockService := new(MockProductServiceForArchive)
mockService.On("RestoreProduct", productID).Return(nil)

handler := api.NewProductHandler(mockService, nil)

err := handler.RestoreProduct(c)

assert.NoError(t, err)
assert.Equal(t, http.StatusOK, rec.Code)

mockService.AssertExpectations(t)
}
