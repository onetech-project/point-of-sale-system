// Package utils provides validation utilities
// File: backend/src/utils/validation.go
// Author: CTO Hero Mode
// Date: 2025-11-23

package utils

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// emailRegex is the regex pattern for email validation
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// slugRegex is the regex pattern for slug validation (URL-friendly)
	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	ErrInvalidEmail        = errors.New("invalid email format")
	ErrInvalidBusinessName = errors.New("business name must be at least 2 characters")
	ErrInvalidSlug         = errors.New("invalid slug format (use lowercase, numbers, and hyphens only)")
)

// ValidateEmail checks if an email address is valid
func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidateBusinessName checks if a business name is valid
func ValidateBusinessName(name string) error {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < 2 {
		return ErrInvalidBusinessName
	}
	if len(trimmed) > 255 {
		return errors.New("business name must be less than 255 characters")
	}
	return nil
}

// ValidateSlug checks if a slug is valid (URL-friendly identifier)
func ValidateSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 100 {
		return errors.New("slug must be between 2 and 100 characters")
	}
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

// GenerateSlug creates a URL-friendly slug from a business name
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters (keep only alphanumeric and hyphens)
	reg := regexp.MustCompile(`[^a-z0-9\-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// ValidateRole checks if a role is valid
func ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"owner":   true,
		"manager": true,
		"cashier": true,
	}
	return validRoles[role]
}

// SanitizeString removes potentially dangerous characters from user input
func SanitizeString(input string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	return sanitized
}
