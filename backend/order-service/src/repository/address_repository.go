package repository

import (
"context"
"database/sql"
"time"

"github.com/rs/zerolog/log"

"github.com/point-of-sale-system/order-service/src/models"
)

// AddressRepository handles database operations for delivery addresses
type AddressRepository struct {
db *sql.DB
}

// NewAddressRepository creates a new address repository
func NewAddressRepository(db *sql.DB) *AddressRepository {
return &AddressRepository{db: db}
}

// Create creates a new delivery address record
// Implements T071: Create delivery address repository
func (r *AddressRepository) Create(ctx context.Context, address *models.DeliveryAddress) error {
query := `
INSERT INTO delivery_addresses (
id, order_id, tenant_id, full_address, latitude, longitude,
geocoding_result, service_area_validated, calculated_fee,
distance_km, zone_id, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

now := time.Now()
address.CreatedAt = now
address.UpdatedAt = now

_, err := r.db.ExecContext(ctx, query,
address.ID,
address.OrderID,
address.TenantID,
address.FullAddress,
address.Latitude,
address.Longitude,
address.GeocodingResult,
address.ServiceAreaValidated,
address.CalculatedFee,
address.DistanceKm,
address.ZoneID,
address.CreatedAt,
address.UpdatedAt,
)

if err != nil {
log.Error().
Err(err).
Str("order_id", address.OrderID).
Str("tenant_id", address.TenantID).
Msg("Failed to create delivery address")
return err
}

log.Info().
Str("address_id", address.ID).
Str("order_id", address.OrderID).
Float64("latitude", address.Latitude).
Float64("longitude", address.Longitude).
Msg("Delivery address created")

return nil
}

// GetByOrderID retrieves a delivery address by order ID
func (r *AddressRepository) GetByOrderID(ctx context.Context, orderID string) (*models.DeliveryAddress, error) {
query := `
SELECT id, order_id, tenant_id, full_address, latitude, longitude,
       geocoding_result, service_area_validated, calculated_fee,
       distance_km, zone_id, created_at, updated_at
FROM delivery_addresses
WHERE order_id = $1
`

var address models.DeliveryAddress
err := r.db.QueryRowContext(ctx, query, orderID).Scan(
&address.ID,
&address.OrderID,
&address.TenantID,
&address.FullAddress,
&address.Latitude,
&address.Longitude,
&address.GeocodingResult,
&address.ServiceAreaValidated,
&address.CalculatedFee,
&address.DistanceKm,
&address.ZoneID,
&address.CreatedAt,
&address.UpdatedAt,
)

if err == sql.ErrNoRows {
return nil, nil
}

if err != nil {
log.Error().
Err(err).
Str("order_id", orderID).
Msg("Failed to get delivery address")
return nil, err
}

return &address, nil
}

// Update updates an existing delivery address
func (r *AddressRepository) Update(ctx context.Context, address *models.DeliveryAddress) error {
query := `
UPDATE delivery_addresses
SET full_address = $1,
    latitude = $2,
    longitude = $3,
    geocoding_result = $4,
    service_area_validated = $5,
    calculated_fee = $6,
    distance_km = $7,
    zone_id = $8,
    updated_at = $9
WHERE id = $10
`

address.UpdatedAt = time.Now()

result, err := r.db.ExecContext(ctx, query,
address.FullAddress,
address.Latitude,
address.Longitude,
address.GeocodingResult,
address.ServiceAreaValidated,
address.CalculatedFee,
address.DistanceKm,
address.ZoneID,
address.UpdatedAt,
address.ID,
)

if err != nil {
log.Error().
Err(err).
Str("address_id", address.ID).
Msg("Failed to update delivery address")
return err
}

rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
log.Warn().
Str("address_id", address.ID).
Msg("No rows affected when updating delivery address")
}

return nil
}

// Delete deletes a delivery address by ID
func (r *AddressRepository) Delete(ctx context.Context, addressID string) error {
query := `DELETE FROM delivery_addresses WHERE id = $1`

result, err := r.db.ExecContext(ctx, query, addressID)
if err != nil {
log.Error().
Err(err).
Str("address_id", addressID).
Msg("Failed to delete delivery address")
return err
}

rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
log.Warn().
Str("address_id", addressID).
Msg("No rows affected when deleting delivery address")
}

return nil
}
