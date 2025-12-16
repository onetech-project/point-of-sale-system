package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// NotificationHistoryHandler handles notification history endpoints
type NotificationHistoryHandler struct {
	notificationService interface {
		GetNotificationHistory(tenantID string, filters map[string]interface{}) (map[string]interface{}, error)
	}
}

// NewNotificationHistoryHandler creates a new notification history handler
func NewNotificationHistoryHandler(notificationService interface {
	GetNotificationHistory(tenantID string, filters map[string]interface{}) (map[string]interface{}, error)
}) *NotificationHistoryHandler {
	return &NotificationHistoryHandler{
		notificationService: notificationService,
	}
}

// GetNotificationHistory handles GET /api/v1/notifications/history
func (h *NotificationHistoryHandler) GetNotificationHistory(c echo.Context) error {
	// Get tenant ID from context (set by auth middleware)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		// Fallback to context
		if tid := c.Get("tenant_id"); tid != nil {
			tenantID = tid.(string)
		}
	}
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid authentication token",
			},
		})
	}

	// Parse query parameters
	filters := make(map[string]interface{})

	// Page number (default: 1)
	page := 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	filters["page"] = page

	// Page size (default: 20, max: 100)
	pageSize := 20
	if pageSizeStr := c.QueryParam("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil {
			if ps < 1 || ps > 100 {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error": map[string]interface{}{
						"code":    "INVALID_PARAMETER",
						"message": "page_size must be between 1 and 100",
						"details": map[string]interface{}{
							"field": "page_size",
							"value": ps,
						},
					},
				})
			}
			pageSize = ps
		}
	}
	filters["page_size"] = pageSize

	// Order reference filter
	if orderRef := c.QueryParam("order_reference"); orderRef != "" {
		filters["order_reference"] = orderRef
	}

	// Status filter
	if status := c.QueryParam("status"); status != "" {
		validStatuses := map[string]bool{
			"pending":   true,
			"sent":      true,
			"failed":    true,
			"cancelled": true,
		}
		if !validStatuses[status] {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INVALID_PARAMETER",
					"message": "status must be one of: pending, sent, failed, cancelled",
					"details": map[string]interface{}{
						"field": "status",
						"value": status,
					},
				},
			})
		}
		filters["status"] = status
	}

	// Type filter
	if notifType := c.QueryParam("type"); notifType != "" {
		validTypes := map[string]bool{
			"order_staff":    true,
			"order_customer": true,
		}
		if !validTypes[notifType] {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INVALID_PARAMETER",
					"message": "type must be one of: order_staff, order_customer",
					"details": map[string]interface{}{
						"field": "type",
						"value": notifType,
					},
				},
			})
		}
		filters["type"] = notifType
	}

	// Date range filters
	if startDate := c.QueryParam("start_date"); startDate != "" {
		if _, err := time.Parse(time.RFC3339, startDate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INVALID_PARAMETER",
					"message": "start_date must be in ISO 8601 format",
					"details": map[string]interface{}{
						"field": "start_date",
						"value": startDate,
					},
				},
			})
		}
		filters["start_date"] = startDate
	}

	if endDate := c.QueryParam("end_date"); endDate != "" {
		if _, err := time.Parse(time.RFC3339, endDate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INVALID_PARAMETER",
					"message": "end_date must be in ISO 8601 format",
					"details": map[string]interface{}{
						"field": "end_date",
						"value": endDate,
					},
				},
			})
		}
		filters["end_date"] = endDate
	}

	// Get notification history
	result, err := h.notificationService.GetNotificationHistory(tenantID, filters)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve notification history: " + err.Error(),
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}
