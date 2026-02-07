package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/services"
	"github.com/pos/auth-service/src/utils"
	"github.com/rs/zerolog/log"
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

// RefreshSession attempts to refresh the session by checking Redis
// This allows token renewal even if the cookie is missing/expired but session is still valid
func (h *SessionHandler) RefreshSession(c echo.Context) error {
	locale := getLocaleFromHeader(c.Request().Header.Get("Accept-Language"))

	// Try to get session ID from existing JWT token first
	var sessionID string
	cookie, err := c.Cookie("auth_token")

	if err == nil {
		// Token exists, extract it
		claims, err := h.jwtService.ExtractClaims(cookie.Value)
		if err == nil {
			sessionID = claims.SessionID
		}
	}

	// If no valid token, check if there's a session ID in request header (for recovery)
	if sessionID == "" {
		sessionID = c.Request().Header.Get("X-Session-ID")
	}

	// If still no session ID, cannot refresh
	if sessionID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": getLocalizedMessage(locale, "auth.session.notFound"),
		})
	}

	// Check if session exists in Redis
	sessionData, err := h.authService.ValidateSession(c.Request().Context(), sessionID)
	if err != nil {
		if err == services.ErrSessionNotFound {
			log.Warn().Msgf("Session not found in Redis during refresh: sessionId=%s", sessionID)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": getLocalizedMessage(locale, "auth.session.expired"),
			})
		}

		log.Error().Msgf("Failed to validate session for refresh: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	// Session is valid - generate new JWT token
	newToken, err := h.jwtService.Generate(sessionID, sessionData.UserID, sessionData.TenantID, sessionData.Email, sessionData.Role)
	if err != nil {
		log.Error().Msgf("Failed to generate new JWT token: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": getLocalizedMessage(locale, "errors.internalServer"),
		})
	}

	// Set new auth cookie
	isProduction := c.Request().Header.Get("X-Forwarded-Proto") == "https"
	newCookie := &http.Cookie{
		Name:     "auth_token",
		Value:    newToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   utils.GetEnvInt("SESSION_TTL_MINUTES") * 60,
	}
	c.SetCookie(newCookie)

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

	c.Logger().Infof("Session refreshed successfully: sessionId=%s, userId=%s", sessionID, sessionData.UserID)
	return c.JSON(http.StatusOK, response)
}
