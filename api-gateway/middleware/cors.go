package middleware

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CORS() echo.MiddlewareFunc {
	allowOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowOrigins == "" {
		// throw error: no allowed origins specified
		panic("ALLOWED_ORIGINS environment variable is not set")
	}

	origins := strings.Split(allowOrigins, ",")

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Request-ID", "X-Tenant-ID", "X-User-ID", "X-User-Email", "X-User-Role", "X-Session-Id"},
		AllowCredentials: true,
		MaxAge:           3600,
	})
}
