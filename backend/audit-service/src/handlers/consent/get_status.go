package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetConsentStatus retrieves current consent status for authenticated user
// GET /api/v1/consent/status
func (h *Handler) GetConsentStatus(c echo.Context) error {
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
		// Guest consent query
		subjectType = "guest"
		subjectID = guestOrderID
	} else {
		// Tenant user consent query
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

	// Get active consents
	consents, err := h.consentRepo.GetActiveConsents(ctx, tenantID, subjectType, subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve consent status",
			},
		})
	}

	// Get current privacy policy
	policy, err := h.consentRepo.GetCurrentPrivacyPolicy(ctx, "en")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve privacy policy",
			},
		})
	}

	// Check if user needs to reconsent (if policy version changed)
	requiresReconsent := false
	if len(consents) > 0 && consents[0].PolicyVersion != policy.Version {
		requiresReconsent = true
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"subject_type":       subjectType,
			"consents":           consents,
			"policy_version":     policy.Version,
			"requires_reconsent": requiresReconsent,
		},
	})
}
