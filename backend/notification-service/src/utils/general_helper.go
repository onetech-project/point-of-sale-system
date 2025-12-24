package utils

import (
	"fmt"
	"os"
	"strconv"
)

// FormatCurrencyIDR formats an amount in IDR currency with thousand separators
// Example: 50000 -> "50.000"
func FormatCurrencyIDR(amount int) string {
	// Simple formatting for Indonesian Rupiah
	if amount < 0 {
		return fmt.Sprintf("-%s", FormatCurrencyIDR(-amount))
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	// Add thousand separators
	var result string
	for i, c := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}

	return result
}

func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
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
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
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
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}
