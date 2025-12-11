package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NotificationConfigHandler handles notification configuration endpoints
type NotificationConfigHandler struct {
	configRepo interface {
		GetNotificationConfig(tenantID string) (map[string]interface{}, error)
		UpdateNotificationConfig(tenantID string, config map[string]interface{}) error
	}
}

// NewNotificationConfigHandler creates a new notification config handler
func NewNotificationConfigHandler(configRepo interface {
	GetNotificationConfig(tenantID string) (map[string]interface{}, error)
	UpdateNotificationConfig(tenantID string, config map[string]interface{}) error
}) *NotificationConfigHandler {
	return &NotificationConfigHandler{
		configRepo: configRepo,
	}
}

// GetNotificationConfig handles GET /api/v1/notifications/config
func (h *NotificationConfigHandler) GetNotificationConfig(c echo.Context) error {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.Get("tenant_id").(string)

	// Get notification config
	config, err := h.configRepo.GetNotificationConfig(tenantID)
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
	// Get tenant ID from context
	tenantID := c.Get("tenant_id").(string)

	// Parse request body
	var config map[string]interface{}
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update notification config
	if err := h.configRepo.UpdateNotificationConfig(tenantID, config); err != nil {
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
