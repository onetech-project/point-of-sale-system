package api

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/tenant-service/src/models"
	"github.com/pos/tenant-service/src/queue"
	"github.com/pos/tenant-service/src/services"
	. "github.com/pos/tenant-service/src/utils"
)

type RegisterHandler struct {
	tenantService  *services.TenantService
	db             *sql.DB
	eventPublisher *queue.EventPublisher
}

func NewRegisterHandler(db *sql.DB, eventPublisher *queue.EventPublisher) *RegisterHandler {
	return &RegisterHandler{
		tenantService:  services.NewTenantService(db, eventPublisher),
		db:             db,
		eventPublisher: eventPublisher,
	}
}

func (h *RegisterHandler) Register(c echo.Context) error {
	// Extract locale from Accept-Language header
	locale := GetLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	var req models.CreateTenantRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.invalidRequest"),
		})
	}

	masker := NewLogMasker()

	// Debug: Log what we received
	c.Logger().Infof("Tenant registration attempt for BusinessName='%s', Email='%s', FirstName='%s', LastName='%s', Consents=%v",
		req.BusinessName, masker.MaskEmail(req.Email), masker.MaskName(req.FirstName), masker.MaskName(req.LastName), req.Consents)

	if !services.IsValidBusinessName(req.BusinessName) {
		c.Logger().Warnf("Invalid business name: %s (length=%d)", req.BusinessName, len(req.BusinessName))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.businessNameRequired"),
		})
	}

	req.Email = services.NormalizeEmail(req.Email)
	if !services.IsValidEmail(req.Email) {
		masker := NewLogMasker()
		c.Logger().Warnf("Invalid email format: %s", masker.MaskEmail(req.Email))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.emailInvalid"),
		})
	}

	if !services.IsValidPassword(req.Password) {
		c.Logger().Warn("Password validation failed for registration attempt")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.passwordRequirements"),
		})
	}

	// Extract IP address and user agent for consent recording
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	tenant, err := h.tenantService.RegisterTenant(c.Request().Context(), &req, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrTenantExists {
			c.Logger().Warnf("Business name already exists: %s", req.BusinessName)
			return c.JSON(http.StatusConflict, map[string]string{
				"error": GetLocalizedMessage(locale, "auth.register.businessNameExists"),
			})
		}
		if err == services.ErrInvalidSlug {
			c.Logger().Warnf("Invalid slug generated for business: %s", req.BusinessName)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": GetLocalizedMessage(locale, "validation.businessNameRequired"),
			})
		}

		// Log detailed error for debugging, return generic message to user
		c.Logger().Errorf("Failed to register tenant for business %s: %v", req.BusinessName, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": GetLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	// Log successful registration (without sensitive data)
	c.Logger().Infof("Tenant registered successfully: ID=%s, slug=%s, business=%s",
		tenant.ID, tenant.Slug, tenant.BusinessName)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"tenant":  tenant.ToResponse(),
		"message": GetLocalizedMessage(locale, "auth.register.success"),
	})
}
