package consent

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/audit-service/src/repository"
	"github.com/pos/audit-service/src/services"
)

// Handler handles HTTP requests for consent management
type Handler struct {
	consentService *services.ConsentService
	consentRepo    *repository.ConsentRepository
}

// NewHandler creates a new consent handler
func NewHandler(consentService *services.ConsentService, consentRepo *repository.ConsentRepository) *Handler {
	return &Handler{
		consentService: consentService,
		consentRepo:    consentRepo,
	}
}

// ListConsentPurposes retrieves all available consent purposes
// GET /api/v1/consent/purposes
func (h *Handler) ListConsentPurposes(c echo.Context) error {
	ctx := c.Request().Context()

	purposes, err := h.consentRepo.ListConsentPurposes(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve consent purposes",
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": purposes,
		"meta": map[string]interface{}{
			"total": len(purposes),
		},
	})
}

// GetConsentPurposeByCode retrieves a specific consent purpose
// GET /api/v1/consent/purposes/:purpose_code
func (h *Handler) GetConsentPurposeByCode(c echo.Context) error {
	ctx := c.Request().Context()
	purposeCode := c.Param("purpose_code")

	purpose, err := h.consentRepo.GetConsentPurposeByCode(ctx, purposeCode)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "Consent purpose not found",
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": purpose,
	})
}
