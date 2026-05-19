package utils

import (
	"os"
	"strconv"
)

// GetEnv returns the value of an environment variable or panics if not set.
func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic("Environment variable " + key + " is not set")
}

// GetEnvBool returns the value of an environment variable as a bool or panics if not set/invalid.
func GetEnvBool(key string) bool {
	if value := os.Getenv(key); value != "" {
		boolVal, err := strconv.ParseBool(value)
		if err == nil {
			return boolVal
		}
	}
	panic("Environment variable " + key + " is not set or is not a valid boolean")
}

// GetEnvInt returns the value of an environment variable as an int,
// falling back to the provided default if unset or invalid.
func GetEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err == nil {
			return intVal
		}
	}
	return fallback
}
