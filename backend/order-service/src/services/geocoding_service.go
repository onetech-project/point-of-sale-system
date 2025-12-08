package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"googlemaps.github.io/maps"

	"github.com/point-of-sale-system/order-service/src/models"
)

const (
	// Cache TTL for geocoding results: 7 days
	geocodingCacheTTL = 7 * 24 * time.Hour

	// Earth radius in kilometers (for Haversine calculation)
	earthRadiusKm = 6371.0
)

// GeocodingService handles address geocoding and service area validation
// Implements T072-T076: Geocoding service with Google Maps API
type GeocodingService struct {
	mapsClient  *maps.Client
	redisClient *redis.Client
}

// NewGeocodingService creates a new geocoding service
func NewGeocodingService(mapsClient *maps.Client, redisClient *redis.Client) *GeocodingService {
	return &GeocodingService{
		mapsClient:  mapsClient,
		redisClient: redisClient,
	}
}

// GeocodingResult represents the result of geocoding an address
type GeocodingResult struct {
	FormattedAddress string  `json:"formatted_address"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	PlaceID          string  `json:"place_id"`
}

// GeocodeAddress geocodes an address to lat/lng coordinates
// Implements T073: Address geocoding with caching
func (s *GeocodingService) GeocodeAddress(ctx context.Context, address string) (*GeocodingResult, error) {
	// Check cache first (T074: Redis caching with 7-day TTL)
	cacheKey := s.getCacheKey(address)
	cachedResult, err := s.getFromCache(ctx, cacheKey)
	if err == nil && cachedResult != nil {
		log.Info().
			Str("address", address).
			Float64("latitude", cachedResult.Latitude).
			Float64("longitude", cachedResult.Longitude).
			Msg("Geocoding result retrieved from cache")
		return cachedResult, nil
	}

	// Call Google Maps Geocoding API
	req := &maps.GeocodingRequest{
		Address: address,
	}

	results, err := s.mapsClient.Geocode(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to geocode address")
		return nil, fmt.Errorf("geocoding failed: %w", err)
	}

	if len(results) == 0 {
		log.Warn().
			Str("address", address).
			Msg("No geocoding results found")
		return nil, errors.New("address not found")
	}

	// Take the first result (most relevant)
	firstResult := results[0]
	geocodingResult := &GeocodingResult{
		FormattedAddress: firstResult.FormattedAddress,
		Latitude:         firstResult.Geometry.Location.Lat,
		Longitude:        firstResult.Geometry.Location.Lng,
		PlaceID:          firstResult.PlaceID,
	}

	// Cache the result
	if err := s.saveToCache(ctx, cacheKey, geocodingResult); err != nil {
		log.Warn().
			Err(err).
			Str("address", address).
			Msg("Failed to cache geocoding result")
		// Non-fatal error, continue
	}

	log.Info().
		Str("address", address).
		Str("formatted_address", geocodingResult.FormattedAddress).
		Float64("latitude", geocodingResult.Latitude).
		Float64("longitude", geocodingResult.Longitude).
		Msg("Address geocoded successfully")

	return geocodingResult, nil
}

// ValidateServiceArea checks if an address is within the tenant's service area
// Implements T075-T076: Service area validation with Haversine distance and point-in-polygon
func (s *GeocodingService) ValidateServiceArea(ctx context.Context, latitude, longitude float64, serviceArea *models.ServiceArea) (bool, float64, error) {
	if serviceArea == nil {
		return false, 0, errors.New("service area is not configured")
	}

	// T075: Haversine distance calculation for radius-based areas
	if serviceArea.Type == "radius" {
		distance := s.calculateHaversineDistance(
			latitude, longitude,
			serviceArea.CenterLatitude, serviceArea.CenterLongitude,
		)

		isWithinArea := distance <= serviceArea.RadiusKm

		log.Info().
			Float64("distance_km", distance).
			Float64("radius_km", serviceArea.RadiusKm).
			Bool("within_area", isWithinArea).
			Msg("Service area validation (radius)")

		return isWithinArea, distance, nil
	}

	// T076: Point-in-polygon validation for zone-based areas
	if serviceArea.Type == "polygon" {
		isWithinArea := s.isPointInPolygon(latitude, longitude, serviceArea.PolygonPoints)

		// For polygon, calculate distance to centroid for delivery fee calculation
		distance := s.calculateDistanceToCentroid(latitude, longitude, serviceArea.PolygonPoints)

		log.Info().
			Bool("within_area", isWithinArea).
			Float64("distance_to_centroid_km", distance).
			Msg("Service area validation (polygon)")

		return isWithinArea, distance, nil
	}

	return false, 0, fmt.Errorf("unsupported service area type: %s", serviceArea.Type)
}

// calculateHaversineDistance calculates the distance between two lat/lng points using Haversine formula
// Implements T075: Haversine distance calculation
func (s *GeocodingService) calculateHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusKm * c
	return distance
}

// isPointInPolygon checks if a point is inside a polygon using ray-casting algorithm
// Implements T076: Point-in-polygon validation
func (s *GeocodingService) isPointInPolygon(lat, lng float64, polygonPoints []models.LatLng) bool {
	if len(polygonPoints) < 3 {
		return false
	}

	inside := false
	j := len(polygonPoints) - 1

	for i := 0; i < len(polygonPoints); i++ {
		xi := polygonPoints[i].Latitude
		yi := polygonPoints[i].Longitude
		xj := polygonPoints[j].Latitude
		yj := polygonPoints[j].Longitude

		intersect := ((yi > lng) != (yj > lng)) &&
			(lat < (xj-xi)*(lng-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}

// calculateDistanceToCentroid calculates the distance from a point to the polygon's centroid
func (s *GeocodingService) calculateDistanceToCentroid(lat, lng float64, polygonPoints []models.LatLng) float64 {
	if len(polygonPoints) == 0 {
		return 0
	}

	// Calculate centroid
	var sumLat, sumLng float64
	for _, point := range polygonPoints {
		sumLat += point.Latitude
		sumLng += point.Longitude
	}

	centroidLat := sumLat / float64(len(polygonPoints))
	centroidLng := sumLng / float64(len(polygonPoints))

	// Calculate distance to centroid
	return s.calculateHaversineDistance(lat, lng, centroidLat, centroidLng)
}

// getCacheKey generates a cache key for an address
func (s *GeocodingService) getCacheKey(address string) string {
	hash := sha256.Sum256([]byte(address))
	return fmt.Sprintf("geocoding:%s", hex.EncodeToString(hash[:]))
}

// getFromCache retrieves a geocoding result from Redis cache
// Implements T074: Geocoding result caching
func (s *GeocodingService) getFromCache(ctx context.Context, key string) (*GeocodingResult, error) {
	val, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.New("cache miss")
	}
	if err != nil {
		return nil, err
	}

	var result GeocodingResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// saveToCache stores a geocoding result in Redis cache with 7-day TTL
// Implements T074: Geocoding result caching with TTL
func (s *GeocodingService) saveToCache(ctx context.Context, key string, result *GeocodingResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, data, geocodingCacheTTL).Err()
}
