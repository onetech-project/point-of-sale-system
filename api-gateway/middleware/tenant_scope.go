package middleware

import (
	"github.com/labstack/echo/v4"
)

func TenantScope() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				c.Logger().Warn("Request processed without tenant_id in context")
			}

			c.Request().Header.Set("X-Tenant-ID", tenantID.(string))
			
			userID := c.Get("user_id")
			if userID != nil {
				c.Request().Header.Set("X-User-ID", userID.(string))
			}

			role := c.Get("role")
			if role != nil {
				c.Request().Header.Set("X-User-Role", role.(string))
			}

			return next(c)
		}
	}
}
