// Package utils provides random token generation utilities
// File: backend/src/utils/random.go
// Author: CTO Hero Mode
// Date: 2025-11-23

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// GenerateSecureToken generates a cryptographically secure random token
// Length is in bytes (32 bytes = 64 hex characters)
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		length = 32 // Default to 32 bytes
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// GenerateVerificationToken generates a 32-byte secure token for email verification
func GenerateVerificationToken() (string, error) {
	return GenerateSecureToken(32)
}

// GeneratePasswordResetToken generates a 32-byte secure token for password reset
func GeneratePasswordResetToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateInvitationToken generates a 32-byte secure token for team invitations
func GenerateInvitationToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateRandomCode generates a random numeric code (e.g., for 2FA)
// Length specifies number of digits (e.g., 6 for a 6-digit code)
func GenerateRandomCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}

	max := int64(1)
	for i := 0; i < length; i++ {
		max *= 10
	}

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}

	// Format with leading zeros
	format := "%0" + string(rune(length+'0')) + "d"
	return hex.EncodeToString([]byte(n.String())), nil
}
