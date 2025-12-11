package utils

import (
	"testing"
)

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   int
		expected string
	}{
		{
			name:     "Small amount",
			amount:   500,
			expected: "500",
		},
		{
			name:     "Thousand",
			amount:   1000,
			expected: "1.000",
		},
		{
			name:     "Fifty thousand",
			amount:   50000,
			expected: "50.000",
		},
		{
			name:     "Million",
			amount:   1000000,
			expected: "1.000.000",
		},
		{
			name:     "Complex amount",
			amount:   1234567,
			expected: "1.234.567",
		},
		{
			name:     "Zero",
			amount:   0,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCurrency(tt.amount)
			if result != tt.expected {
				t.Errorf("FormatCurrency(%d) = %s; want %s", tt.amount, result, tt.expected)
			}
		})
	}
}
