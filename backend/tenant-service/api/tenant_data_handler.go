package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/tenant-service/src/repository"
	"github.com/pos/tenant-service/src/services"
	"github.com/pos/tenant-service/src/utils"
)

type TenantDataHandler struct {
	tenantDataService *services.TenantDataService
}

func NewTenantDataHandler(db *sql.DB, auditPublisher *utils.AuditPublisher) (*TenantDataHandler, error) {
	tenantRepo := repository.NewTenantRepository(db)
	tenantConfigRepo, err := repository.NewTenantConfigRepositoryWithVault(db, auditPublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant config repository: %w", err)
	}

	// Create Vault encryptor for PII decryption
	encryptor, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create vault encryptor: %w", err)
	}

	tenantDataService := services.NewTenantDataService(tenantRepo, tenantConfigRepo, db, encryptor)

	return &TenantDataHandler{
		tenantDataService: tenantDataService,
	}, nil
}

// GetTenantData retrieves all tenant data for UU PDP compliance (Article 3 - right to access)
// GET /api/v1/tenant/data
func (h *TenantDataHandler) GetTenantData(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Missing tenant ID",
		})
	}

	// Verify the requesting user is the tenant owner (set by RBAC middleware in API Gateway)
	userRole := c.Request().Header.Get("X-User-Role")
	if userRole != "owner" {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only tenant owners can access tenant data",
		})
	}

	data, err := h.tenantDataService.GetAllTenantData(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get tenant data: %v", err),
		})
	}

	return c.JSON(http.StatusOK, data)
}

// ExportTenantData generates JSON export of all tenant data for UU PDP compliance (Article 4 - data portability)
// POST /api/v1/tenant/data/export
func (h *TenantDataHandler) ExportTenantData(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Missing tenant ID",
		})
	}

	// Verify the requesting user is the tenant owner (set by RBAC middleware in API Gateway)
	userRole := c.Request().Header.Get("X-User-Role")
	if userRole != "owner" {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only tenant owners can export tenant data",
		})
	}

	jsonData, err := h.tenantDataService.ExportData(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to export tenant data: %v", err),
		})
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=tenant-data-%s.json", tenantID))

	return c.Blob(http.StatusOK, "application/json", jsonData)
}
