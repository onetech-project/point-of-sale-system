package utils

import (
	"regexp"
	"strings"
)

// LogMasker masks sensitive PII in log messages per UU PDP No.27 Tahun 2022
type LogMasker struct {
	emailRegex *regexp.Regexp
	phoneRegex *regexp.Regexp
	tokenRegex *regexp.Regexp
	ipRegex    *regexp.Regexp
	nameRegex  *regexp.Regexp
}

var (
	globalMasker *LogMasker
)

func init() {
	globalMasker = NewLogMasker()
}

// NewLogMasker creates a new log masker with compiled regex patterns
func NewLogMasker() *LogMasker {
	return &LogMasker{
		// Email masking - show first 2 chars + domain
		emailRegex: regexp.MustCompile(`\b([a-zA-Z0-9])([a-zA-Z0-9._-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})\b`),

		// Phone masking - show last 4 digits
		phoneRegex: regexp.MustCompile(`\b(\+?[\d\s()-]{7,}[\d])\b`),

		// Token masking - show first 3 and last 3 chars
		tokenRegex: regexp.MustCompile(`\b(token[_-]?|key[_-]?|secret[_-]?|bearer[_-]?)?([a-zA-Z0-9+/=]{10,})\b`),

		// IP address masking - show first octet
		ipRegex: regexp.MustCompile(`\b(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\b`),

		// Name masking - show first char of first/last name
		nameRegex: regexp.MustCompile(`\b(first_?name|last_?name|full_?name|customer_?name)["\s:=]+([A-Z][a-z]+)(\s+[A-Z][a-z]+)?\b`),
	}
}

// MaskEmail masks email addresses: user@example.com -> us***@example.com
func (m *LogMasker) MaskEmail(text string) string {
	return m.emailRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := m.emailRegex.FindStringSubmatch(match)
		if len(parts) >= 4 {
			firstChar := parts[1]
			domain := parts[3]
			return firstChar + "***@" + domain
		}
		return match
	})
}

// MaskPhone masks phone numbers: +628123456789 -> ******6789
func (m *LogMasker) MaskPhone(text string) string {
	return m.phoneRegex.ReplaceAllStringFunc(text, func(match string) string {
		cleaned := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(match, " ", ""), "(", ""), ")", "")
		if len(cleaned) >= 4 {
			lastFour := cleaned[len(cleaned)-4:]
			return "******" + lastFour
		}
		return "******"
	})
}

// MaskToken masks tokens/keys: abc123xyz789 -> abc***789
func (m *LogMasker) MaskToken(text string) string {
	return m.tokenRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := m.tokenRegex.FindStringSubmatch(match)
		if len(parts) >= 3 {
			prefix := parts[1]
			token := parts[2]
			if len(token) >= 6 {
				return prefix + token[:3] + "***" + token[len(token)-3:]
			}
			return prefix + "***"
		}
		return "***"
	})
}

// MaskIP masks IP addresses: 192.168.1.100 -> 192.***.***.***
func (m *LogMasker) MaskIP(text string) string {
	return m.ipRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := m.ipRegex.FindStringSubmatch(match)
		if len(parts) >= 5 {
			firstOctet := parts[1]
			return firstOctet + ".***.***.***"
		}
		return "***.***.***.***"
	})
}

// MaskName masks names in structured logs: first_name: John -> first_name: J***
func (m *LogMasker) MaskName(text string) string {
	return m.nameRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := m.nameRegex.FindStringSubmatch(match)
		if len(parts) >= 3 {
			fieldName := parts[1]
			firstName := parts[2]
			firstChar := string(firstName[0])

			if len(parts) >= 4 && parts[3] != "" {
				// Full name: John Doe -> J*** D***
				lastName := strings.TrimSpace(parts[3])
				lastChar := string(lastName[0])
				return fieldName + ": " + firstChar + "*** " + lastChar + "***"
			}

			// Single name: John -> J***
			return fieldName + ": " + firstChar + "***"
		}
		return match
	})
}

// MaskAll applies all masking rules to the text
func (m *LogMasker) MaskAll(text string) string {
	masked := text
	masked = m.MaskEmail(masked)
	masked = m.MaskPhone(masked)
	masked = m.MaskToken(masked)
	masked = m.MaskIP(masked)
	masked = m.MaskName(masked)
	return masked
}

// MaskSensitiveFields masks common sensitive field patterns in JSON-like logs
func (m *LogMasker) MaskSensitiveFields(text string) string {
	sensitiveFields := []string{
		"password", "passwd", "pwd",
		"secret", "api_key", "apikey", "access_key",
		"private_key", "priv_key",
		"authorization", "auth",
		"session_id", "sessionid",
		"credit_card", "card_number", "cvv",
	}

	result := text
	for _, field := range sensitiveFields {
		pattern := regexp.MustCompile(`(?i)(["']?` + field + `["']?\s*[:=]\s*)(["']?)([^"',}\s]+)(["']?)`)
		result = pattern.ReplaceAllString(result, `$1$2***$4`)
	}

	return result
}

// Mask is a convenience function that applies all masking rules
func Mask(text string) string {
	return globalMasker.MaskAll(text)
}

// MaskWithFields applies all masking rules including sensitive field masking
func MaskWithFields(text string) string {
	masked := globalMasker.MaskAll(text)
	return globalMasker.MaskSensitiveFields(masked)
}
