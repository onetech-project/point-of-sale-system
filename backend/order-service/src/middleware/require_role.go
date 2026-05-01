package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// Role represents a user role in the system
type Role string

const (
	RoleOwner   Role = "owner"
	RoleManager Role = "manager"
	RoleCashier Role = "cashier"
)

// RequireRole creates middleware that enforces role-based access control
// T089: Middleware for role validation (US4 - Role-Based Deletion)
// Checks X-User-Role header set by API Gateway and validates against allowed roles
func RequireRole(allowedRoles ...Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user role from header (injected by API Gateway from JWT)
			userRole := c.Request().Header.Get("X-User-Role")
			if userRole == "" {
				log.Warn().
					Str("path", c.Path()).
					Str("method", c.Request().Method).
					Msg("User role header missing")
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "user role is required for this operation",
				})
			}

			// Normalize role to lowercase for case-insensitive comparison
			normalizedRole := Role(strings.ToLower(userRole))

			// Check if user's role is in the allowed list
			for _, allowedRole := range allowedRoles {
				if normalizedRole == allowedRole {
					// Store role in context for handler access
					c.Set("user_role", normalizedRole)
					return next(c)
				}
			}

			// Role not authorized
			log.Warn().
				Str("user_role", userRole).
				Str("path", c.Path()).
				Str("method", c.Request().Method).
				Msg("Insufficient permissions for operation")

			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "insufficient permissions",
				"message": "only owners and managers can perform this operation",
			})
		}
	}
}

// GetUserRole retrieves the user role from the Echo context
// Returns empty string if role not set
func GetUserRole(c echo.Context) Role {
	role, ok := c.Get("user_role").(Role)
	if !ok {
		// Fallback to header if not in context
		userRole := c.Request().Header.Get("X-User-Role")
		return Role(strings.ToLower(userRole))
	}
	return role
}

// HasRole checks if the user has any of the specified roles
func HasRole(c echo.Context, roles ...Role) bool {
	userRole := GetUserRole(c)
	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}
