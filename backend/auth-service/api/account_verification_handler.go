package api

// account verification handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/services"
)

type AccountVerificationHandler struct {
	authService *services.AuthService
}

func NewAccountVerificationHandler(authService *services.AuthService) *AccountVerificationHandler {
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

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
