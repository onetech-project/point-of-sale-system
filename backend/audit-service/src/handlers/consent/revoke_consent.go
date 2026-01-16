package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/audit-service/src/services"
)

// RevokeConsentRequest represents the request body for revoking consent
type RevokeConsentRequest struct {
	PurposeCode string `json:"purpose_code" validate:"required"`
}

// RevokeConsent revokes optional consent for authenticated user
// POST /api/v1/consent/revoke
func (h *Handler) RevokeConsent(c echo.Context) error {
	ctx := c.Request().Context()

	var req RevokeConsentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

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

	// Extract user ID from header (only authenticated users can revoke)
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "User ID not found",
			},
		})
	}

	// Get IP address and user agent from request
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Revoke consent
	revokeReq := services.RevokeConsentRequest{
		TenantID:    tenantID,
		SubjectType: "tenant",
		SubjectID:   userID,
		PurposeCode: req.PurposeCode,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	if err := h.consentService.RevokeConsent(ctx, revokeReq); err != nil {
		// Check if trying to revoke required consent
		if err.Error() == "cannot revoke required consent purpose" {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "CONSENT_REQUIRED",
					"message": err.Error(),
				},
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to revoke consent",
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"message":      "Consent revoked successfully",
			"purpose_code": req.PurposeCode,
		},
	})
}
