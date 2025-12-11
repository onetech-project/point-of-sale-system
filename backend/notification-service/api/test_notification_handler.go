package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// TestNotificationHandler handles test notification endpoints
type TestNotificationHandler struct {
	notificationService interface {
		SendTestNotification(tenantID, recipientEmail, notificationType string) (string, error)
	}
}

// NewTestNotificationHandler creates a new test notification handler
func NewTestNotificationHandler(notificationService interface {
	SendTestNotification(tenantID, recipientEmail, notificationType string) (string, error)
}) *TestNotificationHandler {
	return &TestNotificationHandler{
		notificationService: notificationService,
	}
}

// SendTestNotification handles POST /api/v1/notifications/test
func (h *TestNotificationHandler) SendTestNotification(c echo.Context) error {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.Get("tenant_id").(string)

	// Parse request body
	var req struct {
		RecipientEmail   string `json:"recipient_email"`
		NotificationType string `json:"notification_type"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.RecipientEmail == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "recipient_email is required",
		})
	}

	if req.NotificationType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "notification_type is required",
		})
	}

	// Validate notification type
	validTypes := map[string]bool{
		"staff_order_notification": true,
		"customer_receipt":         true,
	}

	if !validTypes[req.NotificationType] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid notification_type. Must be 'staff_order_notification' or 'customer_receipt'",
		})
	}

	// Send test notification
	notificationID, err := h.notificationService.SendTestNotification(tenantID, req.RecipientEmail, req.NotificationType)
	if err != nil {
		// Check for specific error types
		if err.Error() == "invalid email format" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid email format",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to send test notification: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":         true,
		"message":         "Test notification sent successfully",
		"notification_id": notificationID,
	})
}
