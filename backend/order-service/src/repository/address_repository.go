package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// AddressRepository handles database operations for delivery addresses
type AddressRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewAddressRepository creates a new address repository with dependency injection (for testing)
func NewAddressRepository(db *sql.DB, encryptor utils.Encryptor) *AddressRepository {
	return &AddressRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewAddressRepositoryWithVault creates a repository with real VaultClient (for production)
func NewAddressRepositoryWithVault(db *sql.DB) (*AddressRepository, error) {
	vaultEncryptor, err := utils.NewVaultEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultEncryptor: %w", err)
	}
	return NewAddressRepository(db, vaultEncryptor), nil
}

// encryptStringPtr encrypts a pointer to string (handles nil values)
func (r *AddressRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.Encrypt(ctx, *value)
}

// decryptToStringPtr decrypts to a pointer to string (handles empty values)
func (r *AddressRepository) decryptToStringPtr(ctx context.Context, encrypted string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// Create creates a new delivery address record with encrypted PII
// Encrypts: FullAddress, GeocodingResult
// Note: Latitude/Longitude remain plaintext for geocoding queries
func (r *AddressRepository) Create(ctx context.Context, address *models.DeliveryAddress) error {
	// Encrypt PII fields
	encryptedAddress, err := r.encryptor.Encrypt(ctx, address.FullAddress)
	if err != nil {
		return fmt.Errorf("failed to encrypt full_address: %w", err)
	}

	encryptedGeocodingResult, err := r.encryptStringPtr(ctx, address.GeocodingResult)
	if err != nil {
		return fmt.Errorf("failed to encrypt geocoding_result: %w", err)
	}

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

	_, err = r.db.ExecContext(ctx, query,
		address.ID,
		address.OrderID,
		address.TenantID,
		encryptedAddress,
		address.Latitude,
		address.Longitude,
		encryptedGeocodingResult,
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

// GetByOrderID retrieves a delivery address by order ID with decrypted PII
func (r *AddressRepository) GetByOrderID(ctx context.Context, orderID string) (*models.DeliveryAddress, error) {
	query := `
		SELECT id, order_id, tenant_id, full_address, latitude, longitude,
		       geocoding_result, service_area_validated, calculated_fee,
		       distance_km, zone_id, created_at, updated_at
		FROM delivery_addresses
		WHERE order_id = $1
	`

	var address models.DeliveryAddress
	var encryptedAddress, encryptedGeocodingResult string

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&address.ID,
		&address.OrderID,
		&address.TenantID,
		&encryptedAddress,
		&address.Latitude,
		&address.Longitude,
		&encryptedGeocodingResult,
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

	// Decrypt PII fields
	address.FullAddress, err = r.encryptor.Decrypt(ctx, encryptedAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt full_address: %w", err)
	}

	address.GeocodingResult, err = r.decryptToStringPtr(ctx, encryptedGeocodingResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt geocoding_result: %w", err)
	}

	return &address, nil
}

// Update updates an existing delivery address with encrypted PII
func (r *AddressRepository) Update(ctx context.Context, address *models.DeliveryAddress) error {
	// Encrypt PII fields
	encryptedAddress, err := r.encryptor.Encrypt(ctx, address.FullAddress)
	if err != nil {
		return fmt.Errorf("failed to encrypt full_address: %w", err)
	}

	encryptedGeocodingResult, err := r.encryptStringPtr(ctx, address.GeocodingResult)
	if err != nil {
		return fmt.Errorf("failed to encrypt geocoding_result: %w", err)
	}

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
		encryptedAddress,
		address.Latitude,
		address.Longitude,
		encryptedGeocodingResult,
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
