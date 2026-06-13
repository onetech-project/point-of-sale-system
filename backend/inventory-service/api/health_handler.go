package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "inventory-service",
		"timestamp": time.Now().Unix(),
	})
}

func (h *HealthHandler) ReadinessCheck(c echo.Context) error {
	if err := h.db.PingContext(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "not_ready",
			"service": "inventory-service",
			"error":   "database not reachable",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ready",
		"service":   "inventory-service",
		"timestamp": time.Now().Unix(),
		"checks": map[string]string{
			"database": "ok",
		},
	})
}
