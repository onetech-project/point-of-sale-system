package utils

import (
	"log"
	"os"
	"strconv"
	"strings"
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}

	return intValue
}

// getLocaleFromHeader extracts locale from Accept-Language header
func GetLocaleFromHeader(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en"
	}

	// Parse Accept-Language header (e.g., "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")
	parts := strings.Split(acceptLanguage, ",")
	if len(parts) > 0 {
		locale := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		// Extract language code (e.g., "id-ID" -> "id")
		if len(locale) >= 2 {
			return strings.ToLower(locale[:2])
		}
	}

	return "en"
}

// getLocalizedError returns localized error message
// For now, returns English messages. Full i18n integration would load from translation files
func GetLocalizedError(locale, key string) string {
	// Simple mapping for critical messages
	messages := map[string]map[string]string{
		"en": {
			"validation.invalidRequest":        "Invalid request format",
			"validation.businessNameRequired":  "Business name is required and must be 1-100 characters",
			"validation.emailInvalid":          "Invalid email format",
			"validation.passwordRequirements":  "Password must be at least 8 characters and contain letters and numbers",
			"auth.register.businessNameExists": "Business name already taken",
			"auth.register.success":            "Tenant registered successfully. Please login with your credentials.",
			"errors.internalServer":            "Failed to register tenant. Please try again later.",
		},
		"id": {
			"validation.invalidRequest":        "Format permintaan tidak valid",
			"validation.businessNameRequired":  "Nama bisnis wajib diisi dan harus 1-100 karakter",
			"validation.emailInvalid":          "Format email tidak valid",
			"validation.passwordRequirements":  "Kata sandi harus minimal 8 karakter dan mengandung huruf dan angka",
			"auth.register.businessNameExists": "Nama bisnis sudah digunakan",
			"auth.register.success":            "Tenant berhasil didaftarkan. Silakan masuk dengan kredensial Anda.",
			"errors.internalServer":            "Gagal mendaftarkan tenant. Silakan coba lagi nanti.",
		},
	}

	if localeMessages, ok := messages[locale]; ok {
		if msg, ok := localeMessages[key]; ok {
			return msg
		}
	}

	// Fallback to English
	if msg, ok := messages["en"][key]; ok {
		return msg
	}

	return key
}

// maskEmail masks email for logging (user@example.com -> u***@example.com)
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	username := parts[0]
	if len(username) > 1 {
		username = string(username[0]) + "***"
	} else {
		username = "***"
	}

	return username + "@" + parts[1]
}
