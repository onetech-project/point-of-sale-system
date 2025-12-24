package utils

import "os"

func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}
