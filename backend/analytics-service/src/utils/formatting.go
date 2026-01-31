package utils

import (
	"fmt"
	"math"
	"strings"
)

// FormatCurrency formats a float64 value as currency (IDR)
func FormatCurrency(amount float64) string {
	// Round to 2 decimal places
	rounded := math.Round(amount*100) / 100

	// Split into integer and decimal parts
	intPart := int64(rounded)
	decimalPart := int64((rounded - float64(intPart)) * 100)

	// Format integer part with thousand separators
	intStr := formatWithSeparator(intPart, ".")

	// Return formatted string
	if decimalPart > 0 {
		return fmt.Sprintf("Rp %s,%02d", intStr, decimalPart)
	}
	return fmt.Sprintf("Rp %s", intStr)
}

// formatWithSeparator adds thousand separators to a number
func formatWithSeparator(n int64, separator string) string {
	if n < 0 {
		return "-" + formatWithSeparator(-n, separator)
	}

	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result []string
	for i := len(str); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{str[start:i]}, result...)
	}

	return strings.Join(result, separator)
}

// FormatNumber formats a number with thousand separators
func FormatNumber(n int64) string {
	return formatWithSeparator(n, ",")
}

// FormatPercentage formats a percentage value
func FormatPercentage(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", value)
}

// FormatPercentageChange formats a percentage change with + or - sign
func FormatPercentageChange(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0.00%"
	}
	if value >= 0 {
		return fmt.Sprintf("+%.2f%%", value)
	}
	return fmt.Sprintf("%.2f%%", value)
}

// CalculatePercentageChange calculates the percentage change between two values
func CalculatePercentageChange(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100.0 // Undefined, but we return 100% for UI purposes
	}
	return ((current - previous) / previous) * 100
}

// RoundToTwoDecimals rounds a float64 to 2 decimal places
func RoundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}
