package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TenantMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID := c.Request().Header.Get("X-Tenant-ID")
		if tenantID == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Tenant ID not found")
		}
		if _, err := uuid.Parse(tenantID); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid tenant ID")
		}

		c.Set("tenant_id", tenantID)
		if userID := c.Request().Header.Get("X-User-ID"); userID != "" {
			c.Set("user_id", userID)
		}

		return next(c)
	}
}
