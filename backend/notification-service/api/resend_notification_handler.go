package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ResendNotificationHandler handles resending failed notifications
type ResendNotificationHandler struct {
	notificationService interface {
		ResendNotification(tenantID, notificationID string) (map[string]interface{}, error)
	}
}

// NewResendNotificationHandler creates a new resend notification handler
func NewResendNotificationHandler(notificationService interface {
	ResendNotification(tenantID, notificationID string) (map[string]interface{}, error)
}) *ResendNotificationHandler {
	return &ResendNotificationHandler{
		notificationService: notificationService,
	}
}

// ResendNotification handles POST /api/v1/notifications/:notification_id/resend
func (h *ResendNotificationHandler) ResendNotification(c echo.Context) error {
	// Get tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid authentication token",
			},
		})
	}

	// Get notification ID from path parameter
	notificationID := c.Param("notification_id")
	if notificationID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "INVALID_PARAMETER",
				"message": "notification_id is required",
			},
		})
	}

	// Resend notification
	result, err := h.notificationService.ResendNotification(tenantID, notificationID)
	if err != nil {
		// Check for specific error types
		errMsg := err.Error()

		switch errMsg {
		case "notification not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "NOTIFICATION_NOT_FOUND",
					"message": "Notification with ID " + notificationID + " not found",
				},
			})

		case "forbidden":
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "FORBIDDEN",
					"message": "You do not have permission to access this notification",
				},
			})

		case "already sent":
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "ALREADY_SENT",
					"message": "Notification already successfully sent and cannot be resent",
				},
			})

		case "max retries exceeded":
			return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "MAX_RETRIES_EXCEEDED",
					"message": "Maximum retry attempts exceeded. Cannot resend.",
					"details": map[string]interface{}{
						"retry_count": result["retry_count"],
						"max_retries": result["max_retries"],
					},
				},
			})

		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to resend notification: " + errMsg,
				},
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}
