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

// maskName shows only first character: "John Doe" -> "J***"
func MaskName(name string) string {
	if len(name) == 0 {
		return "***"
	}
	return string(name[0]) + "***"
}

// maskPhone shows only last 4 digits: "+628123456789" -> "******6789"
func MaskPhone(phone string) string {
	if len(phone) < 4 {
		return "******"
	}
	return "******" + phone[len(phone)-4:]
}

// maskEmail shows first char + domain: "user@example.com" -> "u***@example.com"
func MaskEmail(email string) string {
	if len(email) == 0 {
		return "***"
	}

	parts := SplitEmail(email)
	if len(parts) != 2 {
		return "***"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) == 0 {
		return "***@" + domain
	}

	return string(local[0]) + "***@" + domain
}

// splitEmail splits email into local and domain parts
func SplitEmail(email string) []string {
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return []string{email[:i], email[i+1:]}
		}
	}
	return []string{email}
}
