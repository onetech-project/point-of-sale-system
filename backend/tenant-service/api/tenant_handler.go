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
	ID                  string `json:"id"`
	BusinessName        string `json:"businessName"`
	Slug                string `json:"slug"`
	Status              string `json:"status"`
	CreatedAt           string `json:"createdAt"`
	MidtransConfigured  bool   `json:"midtrans_configured"`
	MidtransEnvironment string `json:"midtrans_environment"`
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
		SELECT
			t.id,
			t.business_name,
			t.slug,
			t.status,
			t.created_at,
			(COALESCE(tc.midtrans_server_key, '') <> '' AND COALESCE(tc.midtrans_client_key, '') <> '') AS midtrans_configured,
			COALESCE(tc.midtrans_environment, 'sandbox') AS midtrans_environment
		FROM tenants t
		LEFT JOIN tenant_configs tc ON tc.tenant_id = t.id
		WHERE t.id = $1 AND t.status = 'active'
	`

	var tenant TenantInfo
	var createdAt sql.NullTime

	err := h.db.QueryRowContext(c.Request().Context(), query, tenantID).Scan(
		&tenant.ID,
		&tenant.BusinessName,
		&tenant.Slug,
		&tenant.Status,
		&createdAt,
		&tenant.MidtransConfigured,
		&tenant.MidtransEnvironment,
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

// GetInternalTenantStatus returns the tenant account status for gateway enforcement.
func (h *TenantHandler) GetInternalTenantStatus(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var status, subscriptionStatus, midtransEnvironment string
	var midtransConfigured bool
	err := h.db.QueryRowContext(c.Request().Context(), `
		SELECT
			t.status,
			t.subscription_status,
			(COALESCE(tc.midtrans_server_key, '') <> '' AND COALESCE(tc.midtrans_client_key, '') <> '') AS midtrans_configured,
			COALESCE(tc.midtrans_environment, 'sandbox') AS midtrans_environment
		FROM tenants t
		LEFT JOIN tenant_configs tc ON tc.tenant_id = t.id
		WHERE t.id = $1`, tenantID).Scan(
		&status,
		&subscriptionStatus,
		&midtransConfigured,
		&midtransEnvironment,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
	}
	if err != nil {
		c.Logger().Errorf("Failed to get tenant status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tenant status"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tenant_id":            tenantID,
		"status":               status,
		"subscription_status":  subscriptionStatus,
		"midtrans_configured":  midtransConfigured,
		"midtrans_environment": midtransEnvironment,
	})
}
