package utils

import (
	"os"
	"strconv"
)

func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// throw error: required environment variable not set
	panic(key + " environment variable is not set")
}

func GetEnvInt(key string) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			// throw error: invalid integer value
			panic("Invalid integer value for " + key)
		}

		return intValue
	}
	// throw error: required environment variable not set
	panic(key + " environment variable is not set")
}

func GetEnvBool(key string) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			// throw error: invalid boolean value
			panic("Invalid boolean value for " + key)
		}

		return boolValue
	}
	// throw error: required environment variable not set
	panic(key + " environment variable is not set")
}
