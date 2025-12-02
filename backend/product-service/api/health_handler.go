package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/utils"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// HealthCheck returns basic health status
// GET /health
func (h *HealthHandler) HealthCheck(c echo.Context) error {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "product-service",
		"timestamp": time.Now().Unix(),
	}

	return c.JSON(http.StatusOK, response)
}

// ReadinessCheck checks if service is ready to accept traffic
// Verifies database connectivity
// GET /ready
func (h *HealthHandler) ReadinessCheck(c echo.Context) error {
	// Check database connectivity
	ctx := c.Request().Context()
	if err := h.db.PingContext(ctx); err != nil {
		utils.Log.Error("Readiness check failed: database not reachable: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "not_ready",
			"service": "product-service",
			"error":   "database not reachable",
		})
	}

	response := map[string]interface{}{
		"status":    "ready",
		"service":   "product-service",
		"timestamp": time.Now().Unix(),
		"checks": map[string]string{
			"database": "ok",
		},
	}

	return c.JSON(http.StatusOK, response)
}
