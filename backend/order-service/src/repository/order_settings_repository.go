package repository

import (
	"context"
	"database/sql"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/rs/zerolog/log"
)

// OrderSettingsRepository handles database operations for order settings
type OrderSettingsRepository struct {
	db *sql.DB
}

// NewOrderSettingsRepository creates a new order settings repository
func NewOrderSettingsRepository(db *sql.DB) *OrderSettingsRepository {
	return &OrderSettingsRepository{db: db}
}

// GetByTenantID retrieves order settings for a tenant
func (r *OrderSettingsRepository) GetByTenantID(ctx context.Context, tenantID string) (*models.OrderSettings, error) {
	query := `
		SELECT id, tenant_id, delivery_enabled, pickup_enabled, dine_in_enabled,
		       default_delivery_fee, min_order_amount, max_delivery_distance,
		       estimated_prep_time, auto_accept_orders, require_phone_verification,
		       charge_delivery_fee, created_at, updated_at
		FROM order_settings
		WHERE tenant_id = $1
	`

	var settings models.OrderSettings
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&settings.ID,
		&settings.TenantID,
		&settings.DeliveryEnabled,
		&settings.PickupEnabled,
		&settings.DineInEnabled,
		&settings.DefaultDeliveryFee,
		&settings.MinOrderAmount,
		&settings.MaxDeliveryDistance,
		&settings.EstimatedPrepTime,
		&settings.AutoAcceptOrders,
		&settings.RequirePhoneVerification,
		&settings.ChargeDeliveryFee,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &settings, nil
}

// Create creates default order settings for a tenant
func (r *OrderSettingsRepository) Create(ctx context.Context, tenantID string) (*models.OrderSettings, error) {
	query := `
		INSERT INTO order_settings (
			tenant_id, delivery_enabled, pickup_enabled, dine_in_enabled,
			default_delivery_fee, min_order_amount, max_delivery_distance,
			estimated_prep_time, auto_accept_orders, require_phone_verification,
			charge_delivery_fee
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, tenant_id, delivery_enabled, pickup_enabled, dine_in_enabled,
		          default_delivery_fee, min_order_amount, max_delivery_distance,
		          estimated_prep_time, auto_accept_orders, require_phone_verification,
		          charge_delivery_fee, created_at, updated_at
	`

	var settings models.OrderSettings
	err := r.db.QueryRowContext(ctx, query,
		tenantID,
		true,  // delivery_enabled
		true,  // pickup_enabled
		false, // dine_in_enabled
		10000, // default_delivery_fee
		20000, // min_order_amount
		10.0,  // max_delivery_distance
		30,    // estimated_prep_time
		false, // auto_accept_orders
		false, // require_phone_verification
		true,  // charge_delivery_fee
	).Scan(
		&settings.ID,
		&settings.TenantID,
		&settings.DeliveryEnabled,
		&settings.PickupEnabled,
		&settings.DineInEnabled,
		&settings.DefaultDeliveryFee,
		&settings.MinOrderAmount,
		&settings.MaxDeliveryDistance,
		&settings.EstimatedPrepTime,
		&settings.AutoAcceptOrders,
		&settings.RequirePhoneVerification,
		&settings.ChargeDeliveryFee,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// Update updates order settings for a tenant
func (r *OrderSettingsRepository) Update(ctx context.Context, tenantID string, req *models.UpdateOrderSettingsRequest) (*models.OrderSettings, error) {
	// First, get existing settings
	existing, err := r.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// If no settings exist, create them first
	if existing == nil {
		existing, err = r.Create(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	}

	// Build update query dynamically based on what's provided
	query := `
		UPDATE order_settings
		SET 
			delivery_enabled = COALESCE($2, delivery_enabled),
			pickup_enabled = COALESCE($3, pickup_enabled),
			dine_in_enabled = COALESCE($4, dine_in_enabled),
			default_delivery_fee = COALESCE($5, default_delivery_fee),
			min_order_amount = COALESCE($6, min_order_amount),
			max_delivery_distance = COALESCE($7, max_delivery_distance),
			estimated_prep_time = COALESCE($8, estimated_prep_time),
			auto_accept_orders = COALESCE($9, auto_accept_orders),
			require_phone_verification = COALESCE($10, require_phone_verification),
			charge_delivery_fee = COALESCE($11, charge_delivery_fee),
			updated_at = NOW()
		WHERE tenant_id = $1
		RETURNING id, tenant_id, delivery_enabled, pickup_enabled, dine_in_enabled,
		          default_delivery_fee, min_order_amount, max_delivery_distance,
		          estimated_prep_time, auto_accept_orders, require_phone_verification,
		          charge_delivery_fee, created_at, updated_at
	`

	var settings models.OrderSettings
	err = r.db.QueryRowContext(ctx, query,
		tenantID,
		req.DeliveryEnabled,
		req.PickupEnabled,
		req.DineInEnabled,
		req.DefaultDeliveryFee,
		req.MinOrderAmount,
		req.MaxDeliveryDistance,
		req.EstimatedPrepTime,
		req.AutoAcceptOrders,
		req.RequirePhoneVerification,
		req.ChargeDeliveryFee,
	).Scan(
		&settings.ID,
		&settings.TenantID,
		&settings.DeliveryEnabled,
		&settings.PickupEnabled,
		&settings.DineInEnabled,
		&settings.DefaultDeliveryFee,
		&settings.MinOrderAmount,
		&settings.MaxDeliveryDistance,
		&settings.EstimatedPrepTime,
		&settings.AutoAcceptOrders,
		&settings.RequirePhoneVerification,
		&settings.ChargeDeliveryFee,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to update order settings")
		return nil, err
	}

	return &settings, nil
}

// GetOrCreate retrieves settings or creates default ones if they don't exist
func (r *OrderSettingsRepository) GetOrCreate(ctx context.Context, tenantID string) (*models.OrderSettings, error) {
	settings, err := r.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return r.Create(ctx, tenantID)
	}

	return settings, nil
}
