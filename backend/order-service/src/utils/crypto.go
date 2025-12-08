package utils

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// GenerateOrderReference generates a cryptographically secure order reference
// Format: GO-XXXXXX (6 uppercase alphanumeric characters)
func GenerateOrderReference() (string, error) {
	// Generate 6 random bytes (will encode to more than 6 chars)
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base32 and take first 6 characters
	encoded := base32.StdEncoding.EncodeToString(bytes)
	encoded = strings.ToUpper(encoded)

	// Remove padding and take first 6 chars
	encoded = strings.TrimRight(encoded, "=")
	if len(encoded) > 6 {
		encoded = encoded[:6]
	}

	return "GO-" + encoded, nil
}

// ValidateOrderReference checks if an order reference is valid format
func ValidateOrderReference(ref string) bool {
	if len(ref) != 9 { // GO-XXXXXX = 9 characters
		return false
	}
	if !strings.HasPrefix(ref, "GO-") {
		return false
	}
	// Check remaining 6 characters are alphanumeric
	suffix := ref[3:]
	for _, c := range suffix {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
