package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/services"
)

type SessionHandler struct {
	authService *services.AuthService
	jwtService  *services.JWTService
}

func NewSessionHandler(authService *services.AuthService, jwtService *services.JWTService) *SessionHandler {
	return &SessionHandler{
		authService: authService,
		jwtService:  jwtService,
	}
}

// GetSession validates the current session and returns user info
func (h *SessionHandler) GetSession(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	// Extract JWT token from cookie
	cookie, err := c.Cookie("auth_token")
	if err != nil {
		c.Logger().Debug("No auth token cookie found")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": getLocalizedMessage(locale, "auth.session.notFound"),
		})
	}

	// Validate JWT token
	claims, err := h.jwtService.Validate(cookie.Value)
	if err != nil {
		c.Logger().Warnf("Invalid JWT token: %v", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": getLocalizedMessage(locale, "auth.session.invalid"),
		})
	}

	// Validate session exists in Redis
	sessionData, err := h.authService.ValidateSession(c.Request().Context(), claims.SessionID)
	if err != nil {
		if err == services.ErrSessionNotFound {
			c.Logger().Warnf("Session not found in Redis: sessionId=%s", claims.SessionID)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": getLocalizedMessage(locale, "auth.session.expired"),
			})
		}

		c.Logger().Errorf("Failed to validate session: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	// Return session information
	response := models.SessionResponse{
		Valid: true,
		User: &models.UserInfo{
			ID:        sessionData.UserID,
			Email:     sessionData.Email,
			TenantID:  sessionData.TenantID,
			Role:      sessionData.Role,
			FirstName: sessionData.FirstName,
		},
		TenantID: sessionData.TenantID,
	}

	return c.JSON(http.StatusOK, response)
}
