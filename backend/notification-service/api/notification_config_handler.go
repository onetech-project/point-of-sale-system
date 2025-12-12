package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NotificationConfigHandler handles notification configuration endpoints
type NotificationConfigHandler struct {
	configService interface {
		GetNotificationConfig(tenantID string) (map[string]interface{}, error)
		UpdateNotificationConfig(tenantID string, config map[string]interface{}) error
	}
}

// NewNotificationConfigHandler creates a new notification config handler
func NewNotificationConfigHandler(configService interface {
	GetNotificationConfig(tenantID string) (map[string]interface{}, error)
	UpdateNotificationConfig(tenantID string, config map[string]interface{}) error
}) *NotificationConfigHandler {
	return &NotificationConfigHandler{
		configService: configService,
	}
}

// GetNotificationConfig handles GET /api/v1/notifications/config
func (h *NotificationConfigHandler) GetNotificationConfig(c echo.Context) error {
	// Get tenant ID from header (set by API gateway)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		// Fallback to context if header not present
		if tenantIDVal := c.Get("tenant_id"); tenantIDVal != nil {
			tenantID = tenantIDVal.(string)
		}
	}
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized - tenant ID not found",
		})
	}

	// Get notification config
	config, err := h.configService.GetNotificationConfig(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch notification configuration",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"config": config,
	})
}

// PatchNotificationConfig handles PATCH /api/v1/notifications/config
func (h *NotificationConfigHandler) PatchNotificationConfig(c echo.Context) error {
	// Get tenant ID from header (set by API gateway)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		// Fallback to context if header not present
		if tenantIDVal := c.Get("tenant_id"); tenantIDVal != nil {
			tenantID = tenantIDVal.(string)
		}
	}
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized - tenant ID not found",
		})
	}

	// Parse request body
	var config map[string]interface{}
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update notification config
	if err := h.configService.UpdateNotificationConfig(tenantID, config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update notification configuration",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Notification configuration updated successfully",
		"config":  config,
	})
}
