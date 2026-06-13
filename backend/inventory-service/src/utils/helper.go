package utils

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic("Environment variable " + key + " is not set")
}

func GetEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetTenantIDFromContext(c echo.Context) (uuid.UUID, error) {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok || tenantID == "" {
		return uuid.Nil, fmt.Errorf("tenant ID not found")
	}
	return uuid.Parse(tenantID)
}

func GetUserIDFromContext(c echo.Context) (*uuid.UUID, error) {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
