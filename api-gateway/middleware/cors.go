package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pos/api-gateway/utils"
)

func CORS() echo.MiddlewareFunc {
	allowOrigins := utils.GetEnv("ALLOWED_ORIGINS")

	origins := strings.Split(allowOrigins, ",")

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Request-ID", "X-Tenant-ID", "X-User-ID", "X-User-Email", "X-User-Role", "X-Session-Id"},
		AllowCredentials: true,
		MaxAge:           3600,
	})
}
