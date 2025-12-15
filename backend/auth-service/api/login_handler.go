package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/config"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/services"
	. "github.com/pos/auth-service/src/utils"
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
	locale := GetLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid login request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.invalidRequest"),
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		c.Logger().Warn("Missing required login fields")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": GetLocalizedMessage(locale, "validation.requiredFields"),
		})
	}

	// Get client info for audit
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Log login attempt (without password)
	c.Logger().Infof("Login attempt: email=%s, ip=%s",
		MaskEmail(req.Email), ipAddress)

	// Attempt login
	response, token, err := h.authService.Login(c.Request().Context(), &req, ipAddress, userAgent)
	if err != nil {
		// Handle specific errors
		if rateLimitErr, ok := err.(*services.RateLimitError); ok {
			c.Logger().Warnf("Rate limit exceeded for email=%s",
				MaskEmail(req.Email))

			retryAfterSeconds := int(rateLimitErr.RetryAfter.Seconds())
			c.Response().Header().Set("Retry-After", string(rune(retryAfterSeconds)))

			return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error":      GetLocalizedMessage(locale, "auth.login.rateLimitExceeded"),
				"retryAfter": retryAfterSeconds,
			})
		}

		if statusErr, ok := err.(*services.UserStatusError); ok {
			c.Logger().Warnf("Login attempt for %s account: email=%s",
				statusErr.Status, MaskEmail(req.Email))
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": GetLocalizedMessage(locale, "auth.login.accountDisabled"),
			})
		}

		if err == services.ErrInvalidCredentials {
			c.Logger().Warnf("Invalid credentials for email=%s",
				MaskEmail(req.Email))
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": GetLocalizedMessage(locale, "auth.login.failed"),
			})
		}

		// Generic error
		c.Logger().Errorf("Login failed for email=%s: %v", MaskEmail(req.Email), err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": GetLocalizedMessage(locale, "errors.internalServer"),
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
		SameSite: http.SameSiteLaxMode,            // Lax allows cookie on redirects
		MaxAge:   config.SESSION_TTL_MINUTES * 60, // 15 minutes
	}
	c.SetCookie(cookie)

	// Log successful login
	c.Logger().Infof("Login successful: user=%s, tenant=%s, ip=%s",
		response.User.ID, response.User.TenantID, ipAddress)

	return c.JSON(http.StatusOK, response)
}
