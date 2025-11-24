package api

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

type TenantHandler struct {
	db *sql.DB
}

type TenantInfo struct {
	ID           string `json:"id"`
	BusinessName string `json:"businessName"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

func NewTenantHandler(db *sql.DB) *TenantHandler {
	return &TenantHandler{db: db}
}

// GetTenant retrieves tenant information by ID
func (h *TenantHandler) GetTenant(c echo.Context) error {
	// Get tenant ID from header (set by API Gateway middleware)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		// Fallback to context
		if tid := c.Get("tenant_id"); tid != nil {
			tenantID = tid.(string)
		}
	}

	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	query := `
		SELECT id, business_name, slug, status, created_at
		FROM tenants
		WHERE id = $1 AND status = 'active'
	`

	var tenant TenantInfo
	var createdAt sql.NullTime

	err := h.db.QueryRowContext(c.Request().Context(), query, tenantID).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Tenant not found",
		})
	}

	if err != nil {
		c.Logger().Errorf("Failed to get tenant: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve tenant information",
		})
	}

	if createdAt.Valid {
		tenant.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return c.JSON(http.StatusOK, tenant)
}
