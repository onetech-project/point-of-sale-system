package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/services"
)

type OnboardingHandler struct {
	onboardingService interface {
		GetProgress(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error)
		Complete(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error)
	}
}

type onboardingServiceAdapter struct {
	service *services.OnboardingService
}

func (a onboardingServiceAdapter) GetProgress(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	return a.service.GetProgress(c.Request().Context(), identity, tourKey, tourVersion)
}

func (a onboardingServiceAdapter) Complete(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	return a.service.Complete(c.Request().Context(), identity, tourKey, tourVersion)
}

func NewOnboardingHandler(onboardingService *services.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{onboardingService: onboardingServiceAdapter{service: onboardingService}}
}

func NewOnboardingHandlerWithService(onboardingService interface {
	GetProgress(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error)
	Complete(c echo.Context, identity services.OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error)
}) *OnboardingHandler {
	return &OnboardingHandler{onboardingService: onboardingService}
}

func (h *OnboardingHandler) GetProgress(c echo.Context) error {
	tourKey := c.QueryParam("tour_key")
	tourVersion, err := strconv.Atoi(c.QueryParam("tour_version"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tour_version is required"})
	}

	progress, err := h.onboardingService.GetProgress(c, onboardingIdentityFromHeaders(c), tourKey, tourVersion)
	if err != nil {
		return onboardingErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, progress)
}

func (h *OnboardingHandler) Complete(c echo.Context) error {
	var req models.CompleteOnboardingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	progress, err := h.onboardingService.Complete(c, onboardingIdentityFromHeaders(c), req.TourKey, req.TourVersion)
	if err != nil {
		return onboardingErrorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":      true,
		"completed":    true,
		"completed_at": progress.CompletedAt,
	})
}

func onboardingIdentityFromHeaders(c echo.Context) services.OnboardingIdentity {
	return services.OnboardingIdentity{
		TenantID: c.Request().Header.Get("X-Tenant-ID"),
		UserID:   c.Request().Header.Get("X-User-ID"),
		Role:     c.Request().Header.Get("X-User-Role"),
	}
}

func onboardingErrorResponse(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrOnboardingMissingAuth):
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized - user context not found"})
	case errors.Is(err, services.ErrOnboardingInvalidInput):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid onboarding request"})
	case errors.Is(err, services.ErrOnboardingInvalidTour):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tour does not match user role"})
	default:
		c.Logger().Errorf("Onboarding request failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update onboarding progress"})
	}
}
