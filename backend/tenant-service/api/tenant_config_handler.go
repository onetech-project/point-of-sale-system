package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/pos/tenant-service/src/services"
)

type TenantConfigHandler struct {
	configService *services.TenantConfigService
}

func NewTenantConfigHandler(configService *services.TenantConfigService) *TenantConfigHandler {
	return &TenantConfigHandler{
		configService: configService,
	}
}

// GetPublicTenantConfig handles GET /public/tenants/:tenant_slug/config
func (h *TenantConfigHandler) GetPublicTenantConfig(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	if tenantSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_slug is required",
		})
	}

	config, err := h.configService.GetDeliveryConfig(c.Request().Context(), tenantSlug)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve tenant configuration",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateTenantConfig handles PATCH /admin/tenants/:tenant_id/config (for admin use)
func (h *TenantConfigHandler) UpdateTenantConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	var req services.DeliveryConfig
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.TenantID = tenantID

	if err := h.configService.UpdateDeliveryConfig(c.Request().Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Tenant configuration updated successfully",
	})
}

// GetMidtransConfig handles GET /admin/tenants/:tenant_id/midtrans-config
func (h *TenantConfigHandler) GetMidtransConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	config, err := h.configService.GetMidtransConfig(c.Request().Context(), tenantID)
	if err != nil {
		c.Logger().Errorf("Failed to get Midtrans config for tenant %s: %v", tenantID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve Midtrans configuration",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateMidtransConfig handles PATCH /admin/tenants/:tenant_id/midtrans-config
func (h *TenantConfigHandler) UpdateMidtransConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	var req services.MidtransConfig
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.TenantID = tenantID

	if err := h.configService.UpdateMidtransConfig(c.Request().Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Midtrans configuration updated successfully",
	})
}
