package middleware

import (
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type JWTClaims struct {
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId"`
	TenantID  string `json:"tenantId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Current behaviour: authentication token is provided via secure cookie `auth_token`.
			cookie, err := c.Cookie("auth_token")
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing authentication token",
				})
			}

			token, err := jwt.ParseWithClaims(cookie.Value, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				secret := os.Getenv("JWT_SECRET")
				if secret == "" {
					secret = "default-secret-change-in-production"
				}
				return []byte(secret), nil
			})

			if err != nil {
				c.Logger().Errorf("JWT parse error: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authentication token",
				})
			}

			if !token.Valid {
				c.Logger().Warn("JWT token is not valid")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authentication token",
				})
			}

			claims, ok := token.Claims.(*JWTClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token claims",
				})
			}

			c.Set("user_id", claims.UserID)
			c.Set("tenant_id", claims.TenantID)
			c.Set("email", claims.Email)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}
