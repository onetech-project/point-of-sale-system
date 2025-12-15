package utils

import (
	"log"
	"os"
	"strconv"
	"strings"
)

func GetLocaleFromHeader(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en"
	}

	parts := strings.Split(acceptLanguage, ",")
	if len(parts) > 0 {
		locale := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		if len(locale) >= 2 {
			return strings.ToLower(locale[:2])
		}
	}

	return "en"
}

func GetLocalizedMessage(locale, key string) string {
	messages := map[string]map[string]string{
		"en": {
			"validation.invalidRequest":    "Invalid request format",
			"validation.requiredFields":    "Email and password are required",
			"auth.login.failed":            "Invalid email or password",
			"auth.login.rateLimitExceeded": "Too many login attempts. Please try again later.",
			"auth.login.accountDisabled":   "Account is disabled. Please contact support.",
			"auth.logout.success":          "Successfully logged out",
			"auth.session.notFound":        "Session not found",
			"auth.session.invalid":         "Invalid session",
			"auth.session.expired":         "Session expired",
			"errors.internalServer":        "An error occurred. Please try again later.",
		},
		"id": {
			"validation.invalidRequest":    "Format permintaan tidak valid",
			"validation.requiredFields":    "Email dan kata sandi wajib diisi",
			"auth.login.failed":            "Email atau kata sandi tidak valid",
			"auth.login.rateLimitExceeded": "Terlalu banyak percobaan login. Silakan coba lagi nanti.",
			"auth.login.accountDisabled":   "Akun dinonaktifkan. Silakan hubungi dukungan.",
			"auth.logout.success":          "Berhasil keluar",
			"auth.session.notFound":        "Sesi tidak ditemukan",
			"auth.session.invalid":         "Sesi tidak valid",
			"auth.session.expired":         "Sesi kedaluwarsa",
			"errors.internalServer":        "Terjadi kesalahan. Silakan coba lagi nanti.",
		},
	}

	if localeMessages, ok := messages[locale]; ok {
		if msg, ok := localeMessages[key]; ok {
			return msg
		}
	}

	if msg, ok := messages["en"][key]; ok {
		return msg
	}

	return key
}

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
