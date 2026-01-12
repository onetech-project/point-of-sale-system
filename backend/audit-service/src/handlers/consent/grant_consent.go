package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/audit-service/src/services"
)

// GrantConsentRequest represents the request body for granting consent
type GrantConsentRequest struct {
	PurposeCodes  []string `json:"purpose_codes" validate:"required,min=1"`
	SubjectType   string   `json:"subject_type" validate:"required,oneof=tenant guest"`
	SubjectID     string   `json:"subject_id"`
	PolicyVersion string   `json:"policy_version"`
	ConsentMethod string   `json:"consent_method" validate:"required,oneof=registration checkout settings_update"`
	GuestOrderID  string   `json:"guest_order_id"` // For guest consent
}

// GrantConsent records user consent for specified purposes
// POST /api/v1/consent/grant
func (h *Handler) GrantConsent(c echo.Context) error {
	ctx := c.Request().Context()

	var req GrantConsentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	// Extract tenant ID from header (set by API gateway)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]string{
				"code":    "MISSING_TENANT_ID",
				"message": "Tenant ID is required",
			},
		})
	}

	// Extract user ID from header for tenant users
	subjectID := req.SubjectID
	if req.SubjectType == "tenant" {
		userID := c.Request().Header.Get("X-User-ID")
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": map[string]string{
					"code":    "UNAUTHORIZED",
					"message": "User ID not found",
				},
			})
		}
		subjectID = userID
	} else if req.SubjectType == "guest" && req.GuestOrderID != "" {
		subjectID = req.GuestOrderID
	}

	// Get IP address and user agent from request
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Validate and grant consents
	grantReq := services.ConsentGrantRequest{
		TenantID:      tenantID,
		SubjectType:   req.SubjectType,
		SubjectID:     subjectID,
		PurposeCodes:  req.PurposeCodes,
		PolicyVersion: req.PolicyVersion,
		ConsentMethod: req.ConsentMethod,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	}

	if err := h.consentService.GrantConsents(ctx, grantReq); err != nil {
		// Check if validation error (missing required consents)
		if err.Error() == "missing required consent purposes" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "CONSENT_REQUIRED",
					"message": err.Error(),
				},
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to grant consents",
			},
		})
	}

	// Retrieve created consent records for response
	consents, err := h.consentRepo.GetActiveConsents(ctx, tenantID, req.SubjectType, subjectID)
	if err != nil {
		// Consents were created, but failed to retrieve - return success anyway
		return c.JSON(http.StatusCreated, map[string]interface{}{
			"data": map[string]interface{}{
				"message": "Consents granted successfully",
			},
			"meta": map[string]interface{}{
				"consent_count": len(req.PurposeCodes),
			},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"data": consents,
		"meta": map[string]interface{}{
			"consent_count": len(consents),
		},
	})
}
