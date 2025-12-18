package config

import (
	"os"
	"strconv"
)

// GetEnvAsInt returns an environment variable as an integer with a default value
func GetEnvAsInt(key string) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid integer")
}

// GetEnvAsString returns an environment variable as a string with a default value
func GetEnvAsString(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}
