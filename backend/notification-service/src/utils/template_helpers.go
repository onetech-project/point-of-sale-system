package utils

import (
	"fmt"
	"strings"
	"text/template"
)

// GetTemplateFuncMap returns custom template functions for email templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatCurrency": FormatCurrency,
		"formatDate":     FormatDate,
		"formatTime":     FormatTime,
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"title":          strings.Title,
	}
}

// FormatCurrency formats an integer amount (in smallest currency unit) to a readable string
// Example: 50000 -> "50.000"
func FormatCurrency(amount int) string {
	// Convert to string
	amountStr := fmt.Sprintf("%d", amount)

	// Add thousand separators
	var result strings.Builder
	length := len(amountStr)

	for i, digit := range amountStr {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteRune('.')
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// FormatDate formats a date string to a readable format
func FormatDate(dateStr string) string {
	// This is a simple implementation, you may want to parse and format properly
	return dateStr
}

// FormatTime formats a time string to a readable format
func FormatTime(timeStr string) string {
	// This is a simple implementation, you may want to parse and format properly
	return timeStr
}
