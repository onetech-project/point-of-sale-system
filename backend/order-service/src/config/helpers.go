package config

import (
	"os"
	"strconv"
)

// GetEnvAsInt returns an environment variable as an integer with a default value
func GetEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// GetEnvAsString returns an environment variable as a string with a default value
func GetEnvAsString(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
