package middleware

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/utils"
)

type PlatformJWTClaims struct {
	SessionID       string `json:"sessionId"`
	PlatformAdminID string `json:"platformAdminId"`
	Email           string `json:"email"`
	Role            string `json:"role"`
	jwt.RegisteredClaims
}

func PlatformAdminAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("platform_auth_token")
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing platform authentication token"})
			}

			token, err := jwt.ParseWithClaims(cookie.Value, &PlatformJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(utils.GetEnv("JWT_SECRET")), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid platform authentication token"})
			}

			claims, ok := token.Claims.(*PlatformJWTClaims)
			if !ok || claims.PlatformAdminID == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid platform authentication claims"})
			}

			c.Set("platform_session_id", claims.SessionID)
			c.Set("platform_admin_id", claims.PlatformAdminID)
			c.Set("platform_email", claims.Email)
			c.Set("platform_role", claims.Role)

			return next(c)
		}
	}
}
