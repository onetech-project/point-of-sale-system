package validators

import (
	"fmt"
)

// Required tenant consents that MUST be granted for registration
// These are NOT sent in the payload - they're implied/enforced by backend
var requiredTenantConsents = []string{
	"operational",           // Required for account functionality
	"third_party_midtrans",  // Required for payment processing
}

// Optional tenant consents that can be granted or declined
var optionalTenantConsents = []string{
	"analytics",    // Optional analytics consent
	"advertising",  // Optional advertising consent
}

// ValidateTenantConsents validates that only valid optional consent codes are provided
// Required consents are implicit and not sent in the payload (anti-tampering)
func ValidateTenantConsents(consents []string) error {
	// Create map of valid optional consents for fast lookup
	validOptional := make(map[string]bool)
	for _, code := range optionalTenantConsents {
		validOptional[code] = true
	}
	
	// Validate that all provided consents are valid optional consents
	for _, consent := range consents {
		if !validOptional[consent] {
			return fmt.Errorf("invalid consent code: %s (must be one of: %v)", consent, optionalTenantConsents)
		}
	}
	
	return nil
}

// GetRequiredTenantConsents returns the list of required consent codes
// These will be automatically included in the ConsentGrantedEvent
func GetRequiredTenantConsents() []string {
	return requiredTenantConsents
}

// GetAllTenantConsents returns all consents (required + optional provided)
func GetAllTenantConsents(optionalConsents []string) []string {
	// Start with required consents
	allConsents := make([]string, len(requiredTenantConsents))
	copy(allConsents, requiredTenantConsents)
	
	// Add optional consents that were granted
	allConsents = append(allConsents, optionalConsents...)
	
	return allConsents
}
