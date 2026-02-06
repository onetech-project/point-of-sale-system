package config

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"googlemaps.github.io/maps"
)

type GoogleMapsConfig struct {
	APIKey string
}

var MapsClient *maps.Client

// InitGoogleMaps initializes the Google Maps API client
func InitGoogleMaps() error {
	cfg := loadGoogleMapsConfig()

	if cfg.APIKey == "" {
		return fmt.Errorf("GOOGLE_MAPS_API_KEY is required")
	}

	client, err := maps.NewClient(maps.WithAPIKey(cfg.APIKey))
	if err != nil {
		return fmt.Errorf("failed to create Google Maps client: %w", err)
	}

	MapsClient = client

	log.Info().Msg("Google Maps client initialized")

	return nil
}

// GetMapsClient returns the Google Maps client
func GetMapsClient() *maps.Client {
	return MapsClient
}

// CloseGoogleMaps closes the Google Maps client (if needed)
func CloseGoogleMaps() {
	// Google Maps client doesn't require explicit closing
	log.Info().Msg("Google Maps client resources released")
}

func loadGoogleMapsConfig() GoogleMapsConfig {
	return GoogleMapsConfig{
		APIKey: GetEnvAsString("GOOGLE_MAPS_API_KEY"),
	}
}
