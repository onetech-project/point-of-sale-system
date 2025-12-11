package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NotificationPreferencesHandler handles notification preference endpoints
type NotificationPreferencesHandler struct {
	userService interface {
		GetUsersWithNotificationPreferences(tenantID string) ([]map[string]interface{}, error)
		UpdateUserNotificationPreference(tenantID, userID string, receive bool) error
	}
}

// NewNotificationPreferencesHandler creates a new notification preferences handler
func NewNotificationPreferencesHandler(userService interface {
	GetUsersWithNotificationPreferences(tenantID string) ([]map[string]interface{}, error)
	UpdateUserNotificationPreference(tenantID, userID string, receive bool) error
}) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{
		userService: userService,
	}
}

// GetNotificationPreferences handles GET /api/v1/users/notification-preferences
func (h *NotificationPreferencesHandler) GetNotificationPreferences(c echo.Context) error {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.Get("tenant_id").(string)

	// Get all users with their notification preferences
	users, err := h.userService.GetUsersWithNotificationPreferences(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch notification preferences",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// PatchNotificationPreferences handles PATCH /api/v1/users/:user_id/notification-preferences
func (h *NotificationPreferencesHandler) PatchNotificationPreferences(c echo.Context) error {
	// Get tenant ID from context
	tenantID := c.Get("tenant_id").(string)

	// Get user ID from path parameter
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	// Parse request body
	var req struct {
		ReceiveOrderNotifications *bool `json:"receive_order_notifications"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.ReceiveOrderNotifications == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "receive_order_notifications field is required",
		})
	}

	// Update user preference
	if err := h.userService.UpdateUserNotificationPreference(tenantID, userID, *req.ReceiveOrderNotifications); err != nil {
		// Check if user not found
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update notification preference",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"user_id":                     userID,
			"receive_order_notifications": *req.ReceiveOrderNotifications,
		},
	})
}
