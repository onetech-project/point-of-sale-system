package utils

import (
	"regexp"
	"strings"
)

// emailRegex is a simple regex for email validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Trim whitespace
	email = strings.TrimSpace(email)

	// Check length constraints
	if len(email) < 5 || len(email) > 254 {
		return false
	}

	// Check format using regex
	return emailRegex.MatchString(email)
}

// ValidateNotEmpty checks if a string is not empty after trimming
func ValidateNotEmpty(value string) bool {
	return strings.TrimSpace(value) != ""
}
