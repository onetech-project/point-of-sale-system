package api

// account verification handler

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/services"
)

type AccountVerificationHandler struct {
	authService accountVerificationService
}

type accountVerificationService interface {
	VerifyAccount(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, email string) error
}

var verificationEmailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func NewAccountVerificationHandler(authService accountVerificationService) *AccountVerificationHandler {
	return &AccountVerificationHandler{authService: authService}
}

func (h *AccountVerificationHandler) VerifyAccount(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	// get the token from body
	var req struct {
		Token string `json:"token"`
	}

	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid verify account request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.invalidRequest"),
		})
	}

	// Validate required fields
	if req.Token == "" {
		c.Logger().Warn("Missing required verify account token")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.requiredFields"),
		})
	}

	// Attempt account verification
	err := h.authService.VerifyAccount(c.Request().Context(), req.Token)
	if err != nil {
		if err == services.ErrInvalidOrExpiredToken {
			c.Logger().Warnf("Invalid or expired verification token: %s", maskToken(req.Token))
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": getLocalizedMessage(locale, "verification.invalidOrExpiredToken"),
			})
		}

		c.Logger().Errorf("Failed to verify account: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "server.internalError"),
		})
	}

	c.Logger().Infof("Account verified successfully for token: %s", maskToken(req.Token))
	return c.JSON(http.StatusOK, map[string]string{
		"message": getLocalizedMessage(locale, "verification.success"),
	})
}

func (h *AccountVerificationHandler) ResendVerification(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	var req struct {
		Email string `json:"email"`
	}

	if err := c.Bind(&req); err != nil {
		c.Logger().Warnf("Invalid resend verification request format: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.invalidRequest"),
		})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !verificationEmailRegex.MatchString(email) {
		c.Logger().Warn("Invalid resend verification email")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": getLocalizedMessage(locale, "validation.emailInvalid"),
		})
	}

	if err := h.authService.ResendVerificationEmail(c.Request().Context(), email); err != nil {
		c.Logger().Errorf("Failed to resend verification email: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": getLocalizedMessage(locale, "verification.resendSuccess"),
	})
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
