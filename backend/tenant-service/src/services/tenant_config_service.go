package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pos/tenant-service/src/repository"
)

var ErrTenantNotFound = errors.New("tenant not found")

type TenantUnavailableError struct {
	Status             string
	SubscriptionStatus string
}

func (e *TenantUnavailableError) Error() string {
	return "tenant unavailable"
}

type MidtransNotConfiguredError struct {
	Status              string
	SubscriptionStatus  string
	MidtransEnvironment string
}

func (e *MidtransNotConfiguredError) Error() string {
	return "midtrans not configured"
}

type TenantConfigService struct {
	configRepo *repository.TenantConfigRepository
	db         *sql.DB
}

func NewTenantConfigService(configRepo *repository.TenantConfigRepository, db *sql.DB) *TenantConfigService {
	return &TenantConfigService{
		configRepo: configRepo,
		db:         db,
	}
}

type DeliveryConfig struct {
	TenantID             string                 `json:"tenant_id"`
	TenantName           string                 `json:"tenant_name,omitempty"`
	LogoURL              string                 `json:"logo_url,omitempty"`
	Description          string                 `json:"description,omitempty"`
	EnabledDeliveryTypes []string               `json:"enabled_delivery_types"`
	ServiceArea          map[string]interface{} `json:"service_area,omitempty"`
	DeliveryFeeConfig    map[string]interface{} `json:"delivery_fee_config,omitempty"`
	AutoCalculateFees    bool                   `json:"auto_calculate_fees"`
	DefaultDeliveryFee   int                    `json:"default_delivery_fee,omitempty"`
	MinOrderAmount       int                    `json:"min_order_amount,omitempty"`
	EstimatedPrepTime    int                    `json:"estimated_prep_time,omitempty"`
	ChargeDeliveryFee    bool                   `json:"charge_delivery_fee"`
	MidtransConfigured   bool                   `json:"midtrans_configured"`
	MidtransEnvironment  string                 `json:"midtrans_environment"`
}

func (s *TenantConfigService) GetDeliveryConfig(ctx context.Context, tenantSlug string) (*DeliveryConfig, error) {
	// Fetch tenant information
	var tenantID, tenantName, status, subscriptionStatus, midtransEnvironment string
	var midtransConfigured bool
	query := `
		SELECT
			t.id,
			t.business_name,
			t.status,
			t.subscription_status,
			(COALESCE(tc.midtrans_server_key, '') <> '' AND COALESCE(tc.midtrans_client_key, '') <> '') AS midtrans_configured,
			COALESCE(tc.midtrans_environment, 'sandbox') AS midtrans_environment
		FROM tenants t
		LEFT JOIN tenant_configs tc ON tc.tenant_id = t.id
		WHERE t.slug = $1 AND t.status != 'deleted'`
	err := s.db.QueryRowContext(ctx, query, tenantSlug).Scan(
		&tenantID,
		&tenantName,
		&status,
		&subscriptionStatus,
		&midtransConfigured,
		&midtransEnvironment,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant info: %w", err)
	}

	if subscriptionStatus == "" {
		subscriptionStatus = "trial"
	}
	if status != "active" || (subscriptionStatus != "trial" && subscriptionStatus != "active") {
		return nil, &TenantUnavailableError{
			Status:             status,
			SubscriptionStatus: subscriptionStatus,
		}
	}
	if !midtransConfigured {
		return nil, &MidtransNotConfiguredError{
			Status:              status,
			SubscriptionStatus:  subscriptionStatus,
			MidtransEnvironment: midtransEnvironment,
		}
	}

	// Fetch order settings from order_settings table
	var deliveryEnabled, pickupEnabled, dineInEnabled, chargeDeliveryFee bool
	var defaultDeliveryFee, minOrderAmount, estimatedPrepTime sql.NullInt64

	orderSettingsQuery := `
		SELECT delivery_enabled, pickup_enabled, dine_in_enabled, 
		       default_delivery_fee, min_order_amount, estimated_prep_time,
		       charge_delivery_fee
		FROM order_settings 
		WHERE tenant_id = $1`

	err = s.db.QueryRowContext(ctx, orderSettingsQuery, tenantID).Scan(
		&deliveryEnabled, &pickupEnabled, &dineInEnabled,
		&defaultDeliveryFee, &minOrderAmount, &estimatedPrepTime,
		&chargeDeliveryFee,
	)

	// Build enabled delivery types array
	enabledTypes := []string{}
	if err == sql.ErrNoRows {
		// No settings found, return defaults
		enabledTypes = []string{"pickup", "delivery", "dine_in"}
		chargeDeliveryFee = true
	} else if err != nil {
		return nil, fmt.Errorf("failed to get order settings: %w", err)
	} else {
		// Use settings from database
		if deliveryEnabled {
			enabledTypes = append(enabledTypes, "delivery")
		}
		if pickupEnabled {
			enabledTypes = append(enabledTypes, "pickup")
		}
		if dineInEnabled {
			enabledTypes = append(enabledTypes, "dine_in")
		}
	}

	return &DeliveryConfig{
		TenantID:             tenantID,
		TenantName:           tenantName,
		EnabledDeliveryTypes: enabledTypes,
		ServiceArea:          map[string]interface{}{},
		DeliveryFeeConfig:    map[string]interface{}{},
		AutoCalculateFees:    false,
		DefaultDeliveryFee:   int(defaultDeliveryFee.Int64),
		MinOrderAmount:       int(minOrderAmount.Int64),
		EstimatedPrepTime:    int(estimatedPrepTime.Int64),
		ChargeDeliveryFee:    chargeDeliveryFee,
		MidtransConfigured:   midtransConfigured,
		MidtransEnvironment:  midtransEnvironment,
	}, nil
}

func (s *TenantConfigService) IsDeliveryTypeEnabled(ctx context.Context, tenantID, deliveryType string) (bool, error) {
	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get tenant config: %w", err)
	}

	for _, enabled := range config.EnabledDeliveryTypes {
		if enabled == deliveryType {
			return true, nil
		}
	}

	return false, nil
}

func (s *TenantConfigService) UpdateDeliveryConfig(ctx context.Context, config *DeliveryConfig) error {
	// Validate delivery types
	validTypes := map[string]bool{
		"pickup":   true,
		"delivery": true,
		"dine_in":  true,
	}

	for _, dt := range config.EnabledDeliveryTypes {
		if !validTypes[dt] {
			return fmt.Errorf("invalid delivery type: %s", dt)
		}
	}

	repoConfig := &repository.TenantConfig{
		TenantID:             config.TenantID,
		EnabledDeliveryTypes: config.EnabledDeliveryTypes,
		ServiceArea:          config.ServiceArea,
		DeliveryFeeConfig:    config.DeliveryFeeConfig,
		AutoCalculateFees:    config.AutoCalculateFees,
		MidtransEnvironment:  "sandbox",
	}

	// Try to get existing config first
	existing, err := s.configRepo.GetByTenantID(ctx, config.TenantID)
	if err != nil {
		return fmt.Errorf("failed to check existing config: %w", err)
	}

	// If no created_at, it's a default config, so create it
	if existing.CreatedAt == "" {
		return s.configRepo.Create(ctx, repoConfig)
	}

	repoConfig.MidtransServerKey = existing.MidtransServerKey
	repoConfig.MidtransClientKey = existing.MidtransClientKey
	repoConfig.MidtransMerchantID = existing.MidtransMerchantID
	repoConfig.MidtransEnvironment = existing.MidtransEnvironment
	if repoConfig.MidtransEnvironment == "" {
		repoConfig.MidtransEnvironment = "sandbox"
	}

	return s.configRepo.Update(ctx, repoConfig)
}

// MidtransConfig represents Midtrans payment configuration for a tenant
type MidtransConfig struct {
	TenantID     string `json:"tenant_id"`
	ServerKey    string `json:"server_key"`
	ClientKey    string `json:"client_key"`
	MerchantID   string `json:"merchant_id"`
	Environment  string `json:"environment"` // sandbox or production
	IsConfigured bool   `json:"is_configured"`
}

// GetMidtransConfig retrieves Midtrans configuration for a tenant
func (s *TenantConfigService) GetMidtransConfig(ctx context.Context, tenantID string) (*MidtransConfig, error) {
	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	isConfigured := config.MidtransServerKey != "" && config.MidtransClientKey != ""

	return &MidtransConfig{
		TenantID:     tenantID,
		ServerKey:    config.MidtransServerKey,
		ClientKey:    config.MidtransClientKey,
		MerchantID:   config.MidtransMerchantID,
		Environment:  config.MidtransEnvironment,
		IsConfigured: isConfigured,
	}, nil
}

// UpdateMidtransConfig updates Midtrans configuration for a tenant
func (s *TenantConfigService) UpdateMidtransConfig(ctx context.Context, midtransConfig *MidtransConfig) error {
	// Validate environment
	if midtransConfig.Environment != "sandbox" && midtransConfig.Environment != "production" {
		return fmt.Errorf("invalid environment: must be 'sandbox' or 'production'")
	}

	// Get existing config
	config, err := s.configRepo.GetByTenantID(ctx, midtransConfig.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant config: %w", err)
	}

	// Update Midtrans fields
	config.MidtransServerKey = midtransConfig.ServerKey
	config.MidtransClientKey = midtransConfig.ClientKey
	config.MidtransMerchantID = midtransConfig.MerchantID
	config.MidtransEnvironment = midtransConfig.Environment

	// If no created_at, it's a default config, so create it
	if config.CreatedAt == "" {
		return s.configRepo.Create(ctx, config)
	}

	return s.configRepo.Update(ctx, config)
}
