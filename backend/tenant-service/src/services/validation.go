package services

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex        = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	slugRegex         = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`)
	businessNameRegex = regexp.MustCompile(`^[A-Za-z0-9\s\-'.]{1,100}$`)
)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}
	return slugRegex.MatchString(slug)
}

func IsValidBusinessName(name string) bool {
	if len(name) < 1 || len(name) > 100 {
		return false
	}
	return businessNameRegex.MatchString(name)
}

func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	
	// Check for at least one letter and one digit
	hasLetter := false
	hasDigit := false
	
	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsDigit(char) {
			hasDigit = true
		}
	}
	
	return hasLetter && hasDigit
}

func GenerateSlug(businessName string) string {
	slug := strings.ToLower(businessName)

	slug = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)

	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	slug = strings.Trim(slug, "-")

	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	if len(slug) < 3 {
		slug = slug + "-pos"
	}

	return slug
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
