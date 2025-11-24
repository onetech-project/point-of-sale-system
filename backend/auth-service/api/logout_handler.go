package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/services"
)

type LogoutHandler struct {
	authService *services.AuthService
	jwtService  *services.JWTService
}

func NewLogoutHandler(authService *services.AuthService, jwtService *services.JWTService) *LogoutHandler {
	return &LogoutHandler{
		authService: authService,
		jwtService:  jwtService,
	}
}

// Logout terminates the current session
func (h *LogoutHandler) Logout(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	// Extract JWT token from cookie
	cookie, err := c.Cookie("auth_token")
	if err != nil {
		c.Logger().Debug("No auth token cookie found for logout")
		// Even if no cookie, return success (already logged out)
		return c.JSON(http.StatusOK, map[string]string{
			"message": getLocalizedMessage(locale, "auth.logout.success"),
		})
	}

	// Validate JWT token to get session ID
	claims, err := h.jwtService.Validate(cookie.Value)
	if err != nil {
		c.Logger().Warnf("Invalid JWT token on logout: %v", err)
		// Even if token is invalid, clear the cookie
		clearAuthCookie(c)
		return c.JSON(http.StatusOK, map[string]string{
			"message": getLocalizedMessage(locale, "auth.logout.success"),
		})
	}

	// Terminate session in Redis
	err = h.authService.TerminateSession(c.Request().Context(), claims.SessionID)
	if err != nil {
		c.Logger().Errorf("Failed to terminate session: %v", err)
		// Continue to clear cookie even if Redis fails
	}

	// Clear the auth cookie
	clearAuthCookie(c)

	c.Logger().Infof("User logged out: sessionId=%s, userId=%s", claims.SessionID, claims.UserID)

	return c.JSON(http.StatusOK, map[string]string{
		"message": getLocalizedMessage(locale, "auth.logout.success"),
	})
}

// clearAuthCookie removes the auth_token cookie
func clearAuthCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
	}
	c.SetCookie(cookie)
}
