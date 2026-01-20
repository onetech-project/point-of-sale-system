package utils

import (
	"fmt"
	"os"
)

// GetEnv retrieves environment variable or panics if not found (fail-fast pattern)
func GetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic(fmt.Sprintf("Environment variable %s is required but not set", key))
}
