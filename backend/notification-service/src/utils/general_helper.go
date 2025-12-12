package utils

import "fmt"

// FormatCurrencyIDR formats an amount in IDR currency with thousand separators
// Example: 50000 -> "50.000"
func FormatCurrencyIDR(amount int) string {
	// Simple formatting for Indonesian Rupiah
	if amount < 0 {
		return fmt.Sprintf("-%s", FormatCurrencyIDR(-amount))
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	// Add thousand separators
	var result string
	for i, c := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}

	return result
}
