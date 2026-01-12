package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetPrivacyPolicy retrieves the current privacy policy
// GET /api/v1/privacy-policy
func (h *Handler) GetPrivacyPolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current privacy policy
	policy, err := h.consentRepo.GetCurrentPrivacyPolicy(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve privacy policy",
			},
		})
	}

	// Optional: get specific version if requested
	version := c.QueryParam("version")
	if version != "" && version != policy.Version {
		policy, err = h.consentRepo.GetPrivacyPolicyByVersion(ctx, version)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": map[string]string{
					"code":    "NOT_FOUND",
					"message": "Privacy policy version not found",
				},
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": policy,
	})
}
