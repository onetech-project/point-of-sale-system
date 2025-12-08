package models

import (
	"time"
)

// DeliveryAddress represents a geocoded delivery address
type DeliveryAddress struct {
	ID                   string    `json:"id"`
	OrderID              string    `json:"order_id"`
	TenantID             string    `json:"tenant_id"`
	FullAddress          string    `json:"full_address"`
	Latitude             float64   `json:"latitude"`
	Longitude            float64   `json:"longitude"`
	GeocodingResult      *string   `json:"geocoding_result,omitempty"`
	ServiceAreaValidated bool      `json:"service_area_validated"`
	CalculatedFee        int       `json:"calculated_fee"`
	DistanceKm           *float64  `json:"distance_km,omitempty"`
	ZoneID               *string   `json:"zone_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// HasCoordinates checks if the address has been geocoded
func (da *DeliveryAddress) HasCoordinates() bool {
	return da.Latitude != 0 && da.Longitude != 0
}

// ServiceArea represents a tenant's delivery service area configuration
type ServiceArea struct {
	Type            string   `json:"type"` // "radius" or "polygon"
	CenterLatitude  float64  `json:"center_latitude,omitempty"`
	CenterLongitude float64  `json:"center_longitude,omitempty"`
	RadiusKm        float64  `json:"radius_km,omitempty"`
	PolygonPoints   []LatLng `json:"polygon_points,omitempty"`
}

// LatLng represents a geographic coordinate
type LatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
