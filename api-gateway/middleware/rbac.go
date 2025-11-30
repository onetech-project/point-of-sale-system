package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleManager Role = "manager"
	RoleCashier Role = "cashier"
)

func RBACMiddleware(allowedRoles ...Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("role")
			if userRole == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			role := Role(userRole.(string))
			
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Insufficient permissions",
			})
		}
	}
}

func CheckPermission(c echo.Context, requiredRole Role) bool {
	userRole := c.Get("role")
	if userRole == nil {
		return false
	}

	role := Role(userRole.(string))
	
	switch role {
	case RoleOwner:
		return true
	case RoleManager:
		return requiredRole == RoleManager || requiredRole == RoleCashier
	case RoleCashier:
		return requiredRole == RoleCashier
	default:
		return false
	}
}

func HasRole(userRole string, allowedRoles ...string) bool {
	for _, role := range allowedRoles {
		if strings.EqualFold(userRole, role) {
			return true
		}
	}
	return false
}
