package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetConsentHistory retrieves full consent history for authenticated user
// GET /api/v1/consent/history
func (h *Handler) GetConsentHistory(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{
				"code":    "MISSING_TENANT_ID",
				"message": "Tenant ID is required",
			},
		})
	}

	// Check if this is a guest query
	guestOrderID := c.QueryParam("guest_order_id")
	var subjectType, subjectID string

	if guestOrderID != "" {
		// Guest consent history query
		subjectType = "guest"
		subjectID = guestOrderID
	} else {
		// Tenant user consent history query
		userID := c.Request().Header.Get("X-User-ID")
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": map[string]string{
					"code":    "UNAUTHORIZED",
					"message": "User ID not found",
				},
			})
		}
		subjectType = "tenant"
		subjectID = userID
	}

	// Get consent history
	history, err := h.consentService.GetConsentHistory(ctx, tenantID, subjectType, subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve consent history",
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"subject_type": subjectType,
			"history":      history,
		},
		"meta": map[string]interface{}{
			"total": len(history),
		},
	})
}
