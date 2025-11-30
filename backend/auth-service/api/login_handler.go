package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/services"
)

type LoginHandler struct {
	authService *services.AuthService
}

func NewLoginHandler(authService *services.AuthService) *LoginHandler {
	return &LoginHandler{
		authService: authService,
	}
}

func (h *LoginHandler) Login(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid login request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.invalidRequest"),
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		c.Logger().Warn("Missing required login fields")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.requiredFields"),
		})
	}

	// Get client info for audit
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Log login attempt (without password)
	c.Logger().Infof("Login attempt: email=%s, ip=%s",
		maskEmail(req.Email), ipAddress)

	// Attempt login
	response, token, err := h.authService.Login(c.Request().Context(), &req, ipAddress, userAgent)
	if err != nil {
		// Handle specific errors
		if rateLimitErr, ok := err.(*services.RateLimitError); ok {
			c.Logger().Warnf("Rate limit exceeded for email=%s",
				maskEmail(req.Email))

			retryAfterSeconds := int(rateLimitErr.RetryAfter.Seconds())
			c.Response().Header().Set("Retry-After", string(rune(retryAfterSeconds)))

			return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error":      getLocalizedMessage(locale, "auth.login.rateLimitExceeded"),
				"retryAfter": retryAfterSeconds,
			})
		}

		if statusErr, ok := err.(*services.UserStatusError); ok {
			c.Logger().Warnf("Login attempt for %s account: email=%s",
				statusErr.Status, maskEmail(req.Email))
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": getLocalizedMessage(locale, "auth.login.accountDisabled"),
			})
		}

		if err == services.ErrInvalidCredentials {
			c.Logger().Warnf("Invalid credentials for email=%s",
				maskEmail(req.Email))
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": getLocalizedMessage(locale, "auth.login.failed"),
			})
		}

		// Generic error
		c.Logger().Errorf("Login failed for email=%s: %v", maskEmail(req.Email), err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	// Set JWT token in HTTP-only cookie
	// Use Secure flag only in production (HTTPS)
	isProduction := c.Request().Header.Get("X-Forwarded-Proto") == "https"
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode, // Lax allows cookie on redirects
		MaxAge:   15 * 60,               // 15 minutes
	}
	c.SetCookie(cookie)

	// Log successful login
	c.Logger().Infof("Login successful: user=%s, tenant=%s, ip=%s",
		response.User.ID, response.User.TenantID, ipAddress)

	return c.JSON(http.StatusOK, response)
}

// Helper functions

func getLocaleFromHeader(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en"
	}

	parts := strings.Split(acceptLanguage, ",")
	if len(parts) > 0 {
		locale := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		if len(locale) >= 2 {
			return strings.ToLower(locale[:2])
		}
	}

	return "en"
}

func getLocalizedMessage(locale, key string) string {
	messages := map[string]map[string]string{
		"en": {
			"validation.invalidRequest":    "Invalid request format",
			"validation.requiredFields":    "Email and password are required",
			"auth.login.failed":            "Invalid email or password",
			"auth.login.rateLimitExceeded": "Too many login attempts. Please try again later.",
			"auth.login.accountDisabled":   "Account is disabled. Please contact support.",
			"auth.logout.success":          "Successfully logged out",
			"auth.session.notFound":        "Session not found",
			"auth.session.invalid":         "Invalid session",
			"auth.session.expired":         "Session expired",
			"errors.internalServer":        "An error occurred. Please try again later.",
		},
		"id": {
			"validation.invalidRequest":    "Format permintaan tidak valid",
			"validation.requiredFields":    "Email dan kata sandi wajib diisi",
			"auth.login.failed":            "Email atau kata sandi tidak valid",
			"auth.login.rateLimitExceeded": "Terlalu banyak percobaan login. Silakan coba lagi nanti.",
			"auth.login.accountDisabled":   "Akun dinonaktifkan. Silakan hubungi dukungan.",
			"auth.logout.success":          "Berhasil keluar",
			"auth.session.notFound":        "Sesi tidak ditemukan",
			"auth.session.invalid":         "Sesi tidak valid",
			"auth.session.expired":         "Sesi kedaluwarsa",
			"errors.internalServer":        "Terjadi kesalahan. Silakan coba lagi nanti.",
		},
	}

	if localeMessages, ok := messages[locale]; ok {
		if msg, ok := localeMessages[key]; ok {
			return msg
		}
	}

	if msg, ok := messages["en"][key]; ok {
		return msg
	}

	return key
}

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
