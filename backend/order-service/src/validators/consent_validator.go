package validators

import (
	"fmt"
)

// Required guest consents that MUST be granted for checkout
// These are NOT sent in the payload - they're implied/enforced by backend
var requiredGuestConsents = []string{
	"order_processing",            // Required for order fulfillment
	"payment_processing_midtrans", // Required for payment processing
}

// Optional guest consents that can be granted or declined
var optionalGuestConsents = []string{
	"order_communications",        // Optional order updates
	"promotional_communications",  // Optional promotional messages
}

// ValidateGuestConsents validates that only valid optional consent codes are provided
// Required consents are implicit and not sent in the payload (anti-tampering)
func ValidateGuestConsents(consents []string) error {
	// Create map of valid optional consents for fast lookup
	validOptional := make(map[string]bool)
	for _, code := range optionalGuestConsents {
		validOptional[code] = true
	}
	
	// Validate that all provided consents are valid optional consents
	for _, consent := range consents {
		if !validOptional[consent] {
			return fmt.Errorf("invalid consent code: %s (must be one of: %v)", consent, optionalGuestConsents)
		}
	}
	
	return nil
}

// GetRequiredGuestConsents returns the list of required consent codes
// These will be automatically included in the ConsentGrantedEvent
func GetRequiredGuestConsents() []string {
	return requiredGuestConsents
}

// GetAllGuestConsents returns all consents (required + optional provided)
func GetAllGuestConsents(optionalConsents []string) []string {
	// Start with required consents
	allConsents := make([]string, len(requiredGuestConsents))
	copy(allConsents, requiredGuestConsents)
	
	// Add optional consents that were granted
	allConsents = append(allConsents, optionalConsents...)
	
	return allConsents
}
