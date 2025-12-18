package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetEnv retrieves an environment variable or returns a default value
func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}

// GetEnvBool retrieves a boolean environment variable or returns a default value
func GetEnvBool(key string) bool {
	if value := os.Getenv(key); value != "" {
		boolVal, err := strconv.ParseBool(value)
		if err == nil {
			return boolVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid boolean")
}

// GetEnvInt retrieves an integer environment variable or returns a default value
func GetEnvInt(key string) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err == nil {
			return intVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid integer")
}

// GetEnvInt64 retrieves an int64 environment variable or returns a default value
func GetEnvInt64(key string) int64 {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return intVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid int64")
}

// GetTenantIDFromContext extracts tenant ID from Echo context
// It tries multiple sources: X-Tenant-ID header, context value as UUID, context value as string
func GetTenantIDFromContext(c echo.Context) (uuid.UUID, error) {
	// Try to get from header first (set by tenant middleware)
	tenantIDStr := c.Request().Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		// Try from context value (set by JWT middleware)
		if tenantIDVal := c.Get("tenant_id"); tenantIDVal != nil {
			if tenantID, ok := tenantIDVal.(uuid.UUID); ok {
				return tenantID, nil
			}
			if tenantIDStr, ok := tenantIDVal.(string); ok {
				return uuid.Parse(tenantIDStr)
			}
		}
		return uuid.Nil, fmt.Errorf("tenant ID not found")
	}

	return uuid.Parse(tenantIDStr)
}
