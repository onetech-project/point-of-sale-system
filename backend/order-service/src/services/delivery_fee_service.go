package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
) // DeliveryFeeConfig represents delivery fee configuration for a tenant
type DeliveryFeeConfig struct {
	Type           string         `json:"type"` // "distance" or "zone"
	DistanceTiers  []DistanceTier `json:"distance_tiers,omitempty"`
	ZoneFees       map[string]int `json:"zone_fees,omitempty"`
	BaseFee        int            `json:"base_fee"`
	FreeDeliveryKm *float64       `json:"free_delivery_km,omitempty"`
}

// DistanceTier represents a distance-based pricing tier
type DistanceTier struct {
	MaxDistanceKm float64 `json:"max_distance_km"`
	FeeAmount     int     `json:"fee_amount"`
}

// DeliveryFeeService handles delivery fee calculation
// Implements T077-T079: Delivery fee service with distance and zone-based pricing
type DeliveryFeeService struct {
}

// NewDeliveryFeeService creates a new delivery fee service
func NewDeliveryFeeService() *DeliveryFeeService {
	return &DeliveryFeeService{}
}

// CalculateFee calculates the delivery fee based on distance or zone
// Implements T077-T079: Fee calculation with distance tiers and zone-based pricing
func (s *DeliveryFeeService) CalculateFee(ctx context.Context, distanceKm float64, zoneID *string, config *DeliveryFeeConfig) (int, error) {
	if config == nil {
		return 0, errors.New("delivery fee config is not configured")
	}

	// Check for free delivery
	if config.FreeDeliveryKm != nil && distanceKm <= *config.FreeDeliveryKm {
		log.Info().
			Float64("distance_km", distanceKm).
			Float64("free_delivery_threshold", *config.FreeDeliveryKm).
			Msg("Free delivery applied")
		return 0, nil
	}

	// T078: Distance-based tier matching
	if config.Type == "distance" {
		fee, err := s.calculateDistanceBasedFee(distanceKm, config)
		if err != nil {
			return 0, err
		}

		log.Info().
			Float64("distance_km", distanceKm).
			Int("calculated_fee", fee).
			Str("method", "distance").
			Msg("Delivery fee calculated")

		return fee, nil
	}

	// T079: Zone-based fee lookup
	if config.Type == "zone" {
		if zoneID == nil {
			return 0, errors.New("zone_id is required for zone-based pricing")
		}

		fee, err := s.calculateZoneBasedFee(*zoneID, config)
		if err != nil {
			return 0, err
		}

		log.Info().
			Str("zone_id", *zoneID).
			Int("calculated_fee", fee).
			Str("method", "zone").
			Msg("Delivery fee calculated")

		return fee, nil
	}

	return 0, fmt.Errorf("unsupported delivery fee type: %s", config.Type)
}

// calculateDistanceBasedFee calculates fee based on distance tiers
// Implements T078: Distance-based tier matching
func (s *DeliveryFeeService) calculateDistanceBasedFee(distanceKm float64, config *DeliveryFeeConfig) (int, error) {
	if len(config.DistanceTiers) == 0 {
		log.Warn().Msg("No distance tiers configured, using base fee")
		return config.BaseFee, nil
	}

	// Find the matching tier
	for _, tier := range config.DistanceTiers {
		if distanceKm <= tier.MaxDistanceKm {
			log.Debug().
				Float64("distance_km", distanceKm).
				Float64("tier_max_distance", tier.MaxDistanceKm).
				Int("tier_fee", tier.FeeAmount).
				Msg("Matched distance tier")
			return tier.FeeAmount, nil
		}
	}

	// If distance exceeds all tiers, use the highest tier fee
	highestTier := config.DistanceTiers[len(config.DistanceTiers)-1]
	log.Warn().
		Float64("distance_km", distanceKm).
		Float64("highest_tier_max", highestTier.MaxDistanceKm).
		Int("fee", highestTier.FeeAmount).
		Msg("Distance exceeds all tiers, using highest tier fee")

	return highestTier.FeeAmount, nil
}

// calculateZoneBasedFee calculates fee based on delivery zone
// Implements T079: Zone-based fee lookup
func (s *DeliveryFeeService) calculateZoneBasedFee(zoneID string, config *DeliveryFeeConfig) (int, error) {
	if config.ZoneFees == nil || len(config.ZoneFees) == 0 {
		log.Warn().Msg("No zone fees configured, using base fee")
		return config.BaseFee, nil
	}

	fee, exists := config.ZoneFees[zoneID]
	if !exists {
		log.Warn().
			Str("zone_id", zoneID).
			Msg("Zone not found in fee configuration, using base fee")
		return config.BaseFee, nil
	}

	log.Debug().
		Str("zone_id", zoneID).
		Int("zone_fee", fee).
		Msg("Matched zone fee")

	return fee, nil
}

// ValidateConfig validates the delivery fee configuration
func (s *DeliveryFeeService) ValidateConfig(config *DeliveryFeeConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	if config.Type != "distance" && config.Type != "zone" {
		return fmt.Errorf("invalid type: must be 'distance' or 'zone', got '%s'", config.Type)
	}

	if config.Type == "distance" && len(config.DistanceTiers) == 0 {
		return errors.New("distance_tiers cannot be empty for distance-based pricing")
	}

	if config.Type == "zone" && len(config.ZoneFees) == 0 {
		return errors.New("zone_fees cannot be empty for zone-based pricing")
	}

	// Validate distance tiers are sorted
	if config.Type == "distance" {
		for i := 1; i < len(config.DistanceTiers); i++ {
			if config.DistanceTiers[i].MaxDistanceKm <= config.DistanceTiers[i-1].MaxDistanceKm {
				return fmt.Errorf("distance tiers must be sorted in ascending order")
			}
		}
	}

	return nil
}
