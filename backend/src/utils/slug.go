package utils

import (
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug creates a URL-safe slug from a business name
func GenerateSlug(businessName string) string {
	// Convert to lowercase
	slug := strings.ToLower(businessName)
	
	// Replace spaces and special characters with hyphens
	slug = slugRegex.ReplaceAllString(slug, "-")
	
	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")
	
	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	
	return slug
}

// ValidateSlug checks if a slug is valid
func ValidateSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}
	
	validSlugRegex := regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`)
	return validSlugRegex.MatchString(slug)
}
