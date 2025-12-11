package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// TestGetNotificationHistory tests the GET /api/v1/notifications/history endpoint
func TestGetNotificationHistory(t *testing.T) {
	t.Skip("Contract test - implement after handler creation")

	e := echo.New()

	t.Run("should return notification history with pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history?page=1&page_size=20", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "data")

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "notifications")
		assert.Contains(t, data, "pagination")

		pagination := data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["current_page"])
		assert.Equal(t, float64(20), pagination["page_size"])
		assert.Contains(t, pagination, "total_items")
		assert.Contains(t, pagination, "total_pages")
	})

	t.Run("should filter by order_reference", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history?order_reference=ORD-001", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})

		// All notifications should have matching order_reference
		for _, notif := range notifications {
			n := notif.(map[string]interface{})
			if orderRef, ok := n["order_reference"]; ok {
				assert.Equal(t, "ORD-001", orderRef)
			}
		}
	})

	t.Run("should filter by status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history?status=failed", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})

		// All notifications should have status=failed
		for _, notif := range notifications {
			n := notif.(map[string]interface{})
			assert.Equal(t, "failed", n["status"])
		}
	})

	t.Run("should filter by date range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history?start_date=2025-12-01T00:00:00Z&end_date=2025-12-31T23:59:59Z", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("should validate page_size limits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history?page_size=150", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.Error(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response, "error")

		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_PARAMETER", errorData["code"])
		assert.Contains(t, errorData["message"], "page_size")
	})

	t.Run("should return 401 when unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.NewContext(req, rec)
		// No tenant_id set

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.Error(t, err)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("should return empty array when no notifications", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/history", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("X-Tenant-ID", "test-tenant-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("tenant_id", "test-tenant-id")

		// TODO: Call handler when implemented
		// err := handler.GetNotificationHistory(c)
		// assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		notifications := data["notifications"].([]interface{})
		assert.Empty(t, notifications)
	})
}
