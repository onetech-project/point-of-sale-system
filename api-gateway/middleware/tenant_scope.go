package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func TenantScope() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				c.Logger().Error("Request processed without tenant_id in context")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Tenant context not found",
				})
			}

			c.Request().Header.Set("X-Tenant-ID", tenantID.(string))

			userID := c.Get("user_id")
			if userID != nil {
				c.Request().Header.Set("X-User-ID", userID.(string))
			}

			email := c.Get("email")
			if email != nil {
				c.Request().Header.Set("X-User-Email", email.(string))
			}

			role := c.Get("role")
			if role != nil {
				c.Request().Header.Set("X-User-Role", role.(string))
			}

			return next(c)
		}
	}
}
