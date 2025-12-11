package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SnapshotHandler returns a lightweight snapshot of recent orders for a tenant.
// This is a minimal stub for development and testing. Production should implement
// pagination, filters, and proper authorization.
func SnapshotHandler(c echo.Context) error {
	tenant := c.QueryParam("tenant_id")
	if tenant == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
	}

	// Minimal static response for dev/testing
	snapshot := map[string]interface{}{
		"tenant_id": tenant,
		"orders":    []interface{}{},
	}
	return c.JSON(http.StatusOK, snapshot)
}
