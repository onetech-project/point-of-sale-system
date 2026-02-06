package utils_test

import (
	"testing"

	"github.com/pos/user-service/src/utils"
)

// TestMaskEmail verifies FR-013: Email masking shows first 2 chars + domain
func TestMaskEmail(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard email",
			input:    "user@example.com",
			expected: "u***@example.com",
		},
		{
			name:     "Email with subdomain",
			input:    "admin@mail.example.com",
			expected: "a***@mail.example.com",
		},
		{
			name:     "Long username",
			input:    "verylongusername@example.com",
			expected: "v***@example.com",
		},
		{
			name:     "Short username",
			input:    "ab@example.com",
			expected: "a***@example.com",
		},
		{
			name:     "Email in sentence",
			input:    "Contact us at support@example.com for help",
			expected: "Contact us at s***@example.com for help",
		},
		{
			name:     "Multiple emails",
			input:    "user1@example.com and user2@test.com",
			expected: "u***@example.com and u***@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskEmail(tt.input)
			if result != tt.expected {
				t.Errorf("MaskEmail() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMaskPhone verifies FR-014: Phone masking shows last 4 digits
func TestMaskPhone(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Indonesian mobile with +62",
			input:    "+628123456789",
			expected: "+******6789",
		},
		{
			name:     "Indonesian mobile without +",
			input:    "08123456789",
			expected: "******6789",
		},
		{
			name:     "US format with dashes",
			input:    "555-123-4567",
			expected: "******4567",
		},
		{
			name:     "US format with parentheses",
			input:    "(555) 123-4567",
			expected: "(******4567",
		},
		{
			name:     "International format",
			input:    "+1 555 123 4567",
			expected: "+******4567",
		},
		{
			name:     "Phone in sentence",
			input:    "Call me at +628123456789 tomorrow",
			expected: "Call me at +******6789 tomorrow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskPhone(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPhone() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMaskToken verifies FR-015: Token masking shows first 3 and last 3 chars
func TestMaskToken(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Bearer token",
			input:    "bearer_abc123xyz789def456",
			expected: "bearer_abc***456",
		},
		{
			name:     "API key",
			input:    "key_1234567890abcdef",
			expected: "key_123***def",
		},
		{
			name:     "Secret token",
			input:    "secret_token1234567890",
			expected: "secret_tok***890",
		},
		{
			name:     "Plain token (10+ chars)",
			input:    "abc1234567890",
			expected: "abc***890",
		},
		{
			name:     "Token in authorization header",
			input:    "Authorization: Bearer abc123xyz789def456",
			expected: "Aut***ion: Bearer abc***456",
		},
		{
			name:     "JWT-like token",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJ***CJ9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskToken(tt.input)
			if result != tt.expected {
				t.Errorf("MaskToken() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMaskIP verifies FR-016: IP masking shows first octet
func TestMaskIP(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Private IP 192.168",
			input:    "192.168.1.100",
			expected: "192.***.***.***",
		},
		{
			name:     "Private IP 10.0",
			input:    "10.0.0.1",
			expected: "10.***.***.***",
		},
		{
			name:     "Public IP",
			input:    "203.123.45.67",
			expected: "203.***.***.***",
		},
		{
			name:     "Localhost",
			input:    "127.0.0.1",
			expected: "127.***.***.***",
		},
		{
			name:     "IP in log message",
			input:    "Request from 192.168.1.100 blocked",
			expected: "Request from 192.***.***.*** blocked",
		},
		{
			name:     "Multiple IPs",
			input:    "Source: 10.0.0.1, Destination: 192.168.1.1",
			expected: "Source: 10.***.***.***, Destination: 192.***.***.***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskIP(tt.input)
			if result != tt.expected {
				t.Errorf("MaskIP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMaskName verifies FR-017: Name masking shows first character
func TestMaskName(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "First name field",
			input:    "first_name: John",
			expected: "first_name: J***",
		},
		{
			name:     "Last name field",
			input:    "last_name: Doe",
			expected: "last_name: D***",
		},
		{
			name:     "Full name field",
			input:    "full_name: John Doe",
			expected: "full_name: J*** D***",
		},
		{
			name:     "Customer name field",
			input:    "customer_name: Jane Smith",
			expected: "customer_name: J*** S***",
		},
		{
			name:     "Name with camelCase field",
			input:    "first_name: John",
			expected: "first_name: J***",
		},
		{
			name:     "Multiple name fields",
			input:    "first_name: Alice, last_name: Johnson",
			expected: "first_name: A***, last_name: J***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskName(tt.input)
			if result != tt.expected {
				t.Errorf("MaskName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMaskAll verifies all masking patterns work together
func TestMaskAll(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name        string
		input       string
		contains    []string // Substrings that should be in the output
		notContains []string // Substrings that should NOT be in the output
	}{
		{
			name:  "User registration log",
			input: "User registered: email=user@example.com, phone=+628123456789, ip=192.168.1.100",
			contains: []string{
				"u***@example.com",
				"******6789",
				"192.***.***.***",
			},
			notContains: []string{
				"user@example.com",
				"+628123456789",
				"192.168.1.100",
			},
		},
		{
			name:  "Authentication log",
			input: "Login attempt from 10.0.0.1 for admin@example.com with token abc123xyz789",
			contains: []string{
				"10.***.***.***",
				"a***@example.com",
				"abc***789",
			},
			notContains: []string{
				"10.0.0.1",
				"admin@example.com",
				"abc123xyz789",
			},
		},
		{
			name:  "Order creation log",
			input: "Order created: customer_name: John Doe, email: john@test.com, phone: 08123456789",
			contains: []string{
				"J*** D***",
				"j***@test.com",
				"******6789",
			},
			notContains: []string{
				"John Doe",
				"john@test.com",
				"08123456789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskAll(tt.input)

			for _, substr := range tt.contains {
				if !contains(result, substr) {
					t.Errorf("MaskAll() result should contain %q, got %q", substr, result)
				}
			}

			for _, substr := range tt.notContains {
				if contains(result, substr) {
					t.Errorf("MaskAll() result should NOT contain %q, got %q", substr, result)
				}
			}
		})
	}
}

// TestMaskSensitiveFields verifies masking of common sensitive field names
func TestMaskSensitiveFields(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:        "Password field",
			input:       `{"password": "secretpass123"}`,
			contains:    []string{`"password": "***"`},
			notContains: []string{"secretpass123"},
		},
		{
			name:        "API key field",
			input:       `{"api_key": "sk_live_abc123xyz"}`,
			contains:    []string{`"api_key": "***"`},
			notContains: []string{"sk_live_abc123xyz"},
		},
		{
			name:        "Credit card field",
			input:       `{"credit_card": "4532123456789012"}`,
			contains:    []string{`"credit_card": "***"`},
			notContains: []string{"4532123456789012"},
		},
		{
			name:  "Multiple sensitive fields",
			input: `{"password": "pass123", "api_key": "key456", "secret": "xyz789"}`,
			contains: []string{
				`"password": "***"`,
				`"api_key": "***"`,
				`"secret": "***"`,
			},
			notContains: []string{"pass123", "key456", "xyz789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskSensitiveFields(tt.input)

			for _, substr := range tt.contains {
				if !contains(result, substr) {
					t.Errorf("MaskSensitiveFields() result should contain %q, got %q", substr, result)
				}
			}

			for _, substr := range tt.notContains {
				if contains(result, substr) {
					t.Errorf("MaskSensitiveFields() result should NOT contain %q, got %q", substr, result)
				}
			}
		})
	}
}

// TestNoMaskingWhenNoMatch verifies text without PII remains unchanged
func TestNoMaskingWhenNoMatch(t *testing.T) {
	masker := utils.NewLogMasker()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Simple message",
			input: "Server started successfully",
		},
		{
			name:  "Numeric data",
			input: "Order total: 150000, items: 3",
		},
		{
			name:  "Technical log",
			input: "Database connection pool size: 10, timeout: 30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskAll(tt.input)
			if result != tt.input {
				t.Errorf("MaskAll() should not modify text without PII: got %v, want %v", result, tt.input)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
