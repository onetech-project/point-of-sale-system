package api

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pos/tenant-service/src/models"
	"github.com/pos/tenant-service/src/services"
)

type RegisterHandler struct {
	tenantService *services.TenantService
	db            *sql.DB
}

func NewRegisterHandler(db *sql.DB) *RegisterHandler {
	return &RegisterHandler{
		tenantService: services.NewTenantService(db),
		db:            db,
	}
}

func (h *RegisterHandler) Register(c echo.Context) error {
	// Extract locale from Accept-Language header
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))
	
	var req models.CreateTenantRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedError(locale, "validation.invalidRequest"),
		})
	}

	// Debug: Print directly to stdout (will always show)
	log.Printf("========================================")
	log.Printf("DEBUG: Received registration request:")
	log.Printf("  BusinessName: '%s' (length=%d)", req.BusinessName, len(req.BusinessName))
	log.Printf("  Email: '%s'", req.Email)
	log.Printf("  FirstName: '%s'", req.FirstName)
	log.Printf("  LastName: '%s'", req.LastName)
	log.Printf("========================================")

	// Debug: Log what we received
	c.Logger().Infof("DEBUG: Received BusinessName='%s', Email='%s', FirstName='%s', LastName='%s'", 
		req.BusinessName, req.Email, req.FirstName, req.LastName)

	// Log registration attempt (without sensitive data)
	c.Logger().Infof("Tenant registration attempt for business: %s, email: %s", 
		req.BusinessName, maskEmail(req.Email))

	if !services.IsValidBusinessName(req.BusinessName) {
		c.Logger().Warnf("Invalid business name: %s (length=%d)", req.BusinessName, len(req.BusinessName))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedError(locale, "validation.businessNameRequired"),
		})
	}

	req.Email = services.NormalizeEmail(req.Email)
	if !services.IsValidEmail(req.Email) {
		c.Logger().Warnf("Invalid email format: %s", maskEmail(req.Email))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedError(locale, "validation.emailInvalid"),
		})
	}

	if !services.IsValidPassword(req.Password) {
		c.Logger().Warn("Password validation failed for registration attempt")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedError(locale, "validation.passwordRequirements"),
		})
	}

	tenant, err := h.tenantService.RegisterTenant(c.Request().Context(), &req)
	if err != nil {
		if err == services.ErrTenantExists {
			c.Logger().Warnf("Business name already exists: %s", req.BusinessName)
			return c.JSON(http.StatusConflict, map[string]string{
				"error": getLocalizedError(locale, "auth.register.businessNameExists"),
			})
		}
		if err == services.ErrInvalidSlug {
			c.Logger().Warnf("Invalid slug generated for business: %s", req.BusinessName)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": getLocalizedError(locale, "validation.businessNameRequired"),
			})
		}

		// Log detailed error for debugging, return generic message to user
		c.Logger().Errorf("Failed to register tenant for business %s: %v", req.BusinessName, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedError(locale, "errors.internalServer"),
		})
	}

	// Log successful registration (without sensitive data)
	c.Logger().Infof("Tenant registered successfully: ID=%s, slug=%s, business=%s", 
		tenant.ID, tenant.Slug, tenant.BusinessName)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"tenant": tenant.ToResponse(),
		"message": getLocalizedError(locale, "auth.register.success"),
	})
}

// getLocaleFromHeader extracts locale from Accept-Language header
func getLocaleFromHeader(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en"
	}
	
	// Parse Accept-Language header (e.g., "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	parts := strings.Split(acceptLanguage, ",")
	if len(parts) > 0 {
		locale := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		// Extract language code (e.g., "id-ID" -> "id")
		if len(locale) >= 2 {
			return strings.ToLower(locale[:2])
		}
	}
	
	return "en"
}

// getLocalizedError returns localized error message
// For now, returns English messages. Full i18n integration would load from translation files
func getLocalizedError(locale, key string) string {
	// Simple mapping for critical messages
	messages := map[string]map[string]string{
		"en": {
			"validation.invalidRequest":       "Invalid request format",
			"validation.businessNameRequired": "Business name is required and must be 1-100 characters",
			"validation.emailInvalid":         "Invalid email format",
			"validation.passwordRequirements": "Password must be at least 8 characters and contain letters and numbers",
			"auth.register.businessNameExists": "Business name already taken",
			"auth.register.success":           "Tenant registered successfully. Please login with your credentials.",
			"errors.internalServer":           "Failed to register tenant. Please try again later.",
		},
		"id": {
			"validation.invalidRequest":       "Format permintaan tidak valid",
			"validation.businessNameRequired": "Nama bisnis wajib diisi dan harus 1-100 karakter",
			"validation.emailInvalid":         "Format email tidak valid",
			"validation.passwordRequirements": "Kata sandi harus minimal 8 karakter dan mengandung huruf dan angka",
			"auth.register.businessNameExists": "Nama bisnis sudah digunakan",
			"auth.register.success":           "Tenant berhasil didaftarkan. Silakan masuk dengan kredensial Anda.",
			"errors.internalServer":           "Gagal mendaftarkan tenant. Silakan coba lagi nanti.",
		},
	}
	
	if localeMessages, ok := messages[locale]; ok {
		if msg, ok := localeMessages[key]; ok {
			return msg
		}
	}
	
	// Fallback to English
	if msg, ok := messages["en"][key]; ok {
		return msg
	}
	
	return key
}

// maskEmail masks email for logging (user@example.com -> u***@example.com)
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}
	
	username := parts[0]
	if len(username) > 1 {
		username = string(username[0]) + "***"
	} else {
		username = "***"
	}
	
	return username + "@" + parts[1]
}
