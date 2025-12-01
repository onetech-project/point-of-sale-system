package middleware

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/config"
)

func TenantMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Try to get tenant ID from header first (set by API Gateway)
		tenantID := c.Request().Header.Get("X-Tenant-ID")
		
		// Fallback to context if not in header
		if tenantID == "" {
			tenantIDCtx := c.Get("tenant_id")
			if tenantIDCtx != nil {
				tenantID = tenantIDCtx.(string)
			}
		}
		
		if tenantID == "" {
			c.Logger().Error("Tenant ID not found in header or context")
			c.Logger().Errorf("Headers: %v", c.Request().Header)
			return echo.NewHTTPError(401, "Tenant ID not found")
		}

		// Set in context for handlers to use
		c.Set("tenant_id", tenantID)
		c.Logger().Debugf("Tenant ID set: %s", tenantID)

		// Get user ID from header (set by API Gateway)
		userID := c.Request().Header.Get("X-User-ID")
		if userID != "" {
			c.Set("user_id", userID)
			c.Logger().Debugf("User ID set: %s", userID)
		}

		// Set RLS context in database
		// Note: SET LOCAL doesn't support parameterized queries, but tenant_id is a UUID so it's safe
		setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
		_, err := config.DB.Exec(setContextSQL)
		if err != nil {
			c.Logger().Errorf("Failed to set RLS context: %v", err)
			return echo.NewHTTPError(500, "Failed to set tenant context")
		}

		return next(c)
	}
}
