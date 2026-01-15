package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Role represents user roles in the system
type Role string

const (
	RoleOwner   Role = "owner"   // Tenant owner - full access
	RoleManager Role = "manager" // Manager - limited access
	RoleCashier Role = "cashier" // Cashier - minimal access
	RoleAdmin   Role = "admin"   // Platform admin - full access
)

// RBACMiddleware checks if authenticated user has one of the allowed roles (T109)
// Usage: e.GET("/audit/tenant", handler, middleware.JWTAuth(), middleware.RBACMiddleware(RoleOwner, RoleAdmin))
func RBACMiddleware(allowedRoles ...Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract role from context (set by JWTAuth middleware)
			userRoleInterface := c.Get("role")
			if userRoleInterface == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			userRole := Role(userRoleInterface.(string))

			// Check if user role matches any allowed role
			for _, allowedRole := range allowedRoles {
				if userRole == allowedRole {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Insufficient permissions - only tenant owners and platform admins can access audit logs",
			})
		}
	}
}
