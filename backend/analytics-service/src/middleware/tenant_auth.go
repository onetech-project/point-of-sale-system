package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// TenantAuth middleware extracts tenant_id from request headers (set by api-gateway)
// Analytics service doesn't handle JWT authentication - that's done by the API Gateway
// We just extract the tenant_id that the gateway has validated and set in headers
func TenantAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract tenant_id from header (set by API Gateway after JWT validation)
			tenantID := c.Request().Header.Get("X-Tenant-ID")
			if tenantID == "" {
				log.Warn().Msg("Missing X-Tenant-ID header - request not authenticated by gateway")
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{
					"status": "error",
					"error": map[string]string{
						"code":    "UNAUTHORIZED",
						"message": "Missing tenant authentication",
					}["message"],
				})
			}

			// Store tenant_id in context for use in handlers
			c.Set("tenant_id", tenantID)

			log.Debug().
				Str("tenant_id", tenantID).
				Str("path", c.Request().URL.Path).
				Msg("Tenant context set from gateway header")

			return next(c)
		}
	}
}

// GetTenantID retrieves tenant_id from Echo context
func GetTenantID(c echo.Context) string {
	if tenantID, ok := c.Get("tenant_id").(string); ok {
		return tenantID
	}
	return ""
}
