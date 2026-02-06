package utils

import (
	"os"
	"strconv"
	"strings"
)

func GetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}

// convert environment variable to integer
func GetEnvInt(key string) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err == nil {
			return intVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid integer")
}

// convert environment variable to int64
func GetEnvInt64(key string) int64 {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return intVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid int64")
}

// convert environment variable to boolean
func GetEnvBool(key string) bool {
	if value := os.Getenv(key); value != "" {
		boolVal, err := strconv.ParseBool(value)
		if err == nil {
			return boolVal
		}
	}

	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set or is not a valid boolean")
}

// GetLocaleFromHeader extracts locale from Accept-Language header
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

// GetLocalizedMessage returns localized error message
// For now, returns English messages. Full i18n integration would load from translation files
func GetLocalizedMessage(locale, key string) string {
	// Simple mapping for critical messages
	messages := map[string]map[string]string{
		"en": {
			"validation.invalidRequest":        "Invalid request format",
			"validation.businessNameRequired":  "Business name is required and must be 1-100 characters",
			"validation.emailInvalid":          "Invalid email format",
			"validation.passwordRequirements":  "Password must be at least 8 characters and contain letters and numbers",
			"auth.register.businessNameExists": "Business name already taken",
			"auth.register.success":            "Tenant registered successfully. We've sent you a verification email.",
			"errors.internalServer":            "Failed to register tenant. Please try again later.",
		},
		"id": {
			"validation.invalidRequest":        "Format permintaan tidak valid",
			"validation.businessNameRequired":  "Nama bisnis wajib diisi dan harus 1-100 karakter",
			"validation.emailInvalid":          "Format email tidak valid",
			"validation.passwordRequirements":  "Kata sandi harus minimal 8 karakter dan mengandung huruf dan angka",
			"auth.register.businessNameExists": "Nama bisnis sudah digunakan",
			"auth.register.success":            "Tenant berhasil didaftarkan. Kami telah mengirimkan email verifikasi kepada Anda.",
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

// MaskEmail masks email for logging (user@example.com -> u***@example.com)
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
