package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

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

// PaymentNotConfiguredError means the tenant's default payment gateway has no
// complete credentials, so guest ordering cannot take payments.
type PaymentNotConfiguredError struct {
	Status              string
	SubscriptionStatus  string
	PaymentGateway      string
	MidtransEnvironment string
}

func (e *PaymentNotConfiguredError) Error() string {
	return e.PaymentGateway + " not configured"
}

// Reason is the machine readable reason sent to clients, e.g. midtrans_not_configured.
func (e *PaymentNotConfiguredError) Reason() string {
	return e.PaymentGateway + "_not_configured"
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
	PaymentGateway       string                 `json:"payment_gateway"`
	PaymentConfigured    bool                   `json:"payment_configured"`
}

func (s *TenantConfigService) GetDeliveryConfig(ctx context.Context, tenantSlug string) (*DeliveryConfig, error) {
	// Fetch tenant information
	var tenantID, tenantName, status, subscriptionStatus, midtransEnvironment, paymentGateway string
	var midtransConfigured, paymentConfigured bool
	query := `
		SELECT
			t.id,
			t.business_name,
			t.status,
			t.subscription_status,
			` + repository.MidtransConfiguredSQL + ` AS midtrans_configured,
			COALESCE(tc.midtrans_environment, 'sandbox') AS midtrans_environment,
			` + repository.PaymentGatewaySQL + ` AS payment_gateway,
			` + repository.PaymentConfiguredSQL + ` AS payment_configured
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
		&paymentGateway,
		&paymentConfigured,
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
	if !paymentConfigured {
		return nil, &PaymentNotConfiguredError{
			Status:              status,
			SubscriptionStatus:  subscriptionStatus,
			PaymentGateway:      paymentGateway,
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
		PaymentGateway:       paymentGateway,
		PaymentConfigured:    paymentConfigured,
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

	// Keep the payment settings: this endpoint only changes delivery settings.
	existing.EnabledDeliveryTypes = repoConfig.EnabledDeliveryTypes
	existing.ServiceArea = repoConfig.ServiceArea
	existing.DeliveryFeeConfig = repoConfig.DeliveryFeeConfig
	existing.AutoCalculateFees = repoConfig.AutoCalculateFees
	if existing.MidtransEnvironment == "" {
		existing.MidtransEnvironment = "sandbox"
	}

	return s.configRepo.Update(ctx, existing)
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

	return &MidtransConfig{
		TenantID:     tenantID,
		ServerKey:    config.MidtransServerKey,
		ClientKey:    config.MidtransClientKey,
		MerchantID:   config.MidtransMerchantID,
		Environment:  config.MidtransEnvironment,
		IsConfigured: config.MidtransConfigured(),
	}, nil
}

// UpdateMidtransConfig updates Midtrans configuration for a tenant
func (s *TenantConfigService) UpdateMidtransConfig(ctx context.Context, midtransConfig *MidtransConfig) error {
	// Validate environment
	if !validEnvironment(midtransConfig.Environment) {
		return &ValidationError{Message: "invalid environment: must be 'sandbox' or 'production'"}
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

// ValidationError is a client error in a payment configuration request.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func validEnvironment(environment string) bool {
	return environment == "sandbox" || environment == "production"
}

// ManjoConfig is the Manjo configuration shown to tenant owners. The client
// secret and private key are write-only: they are never returned, only whether
// one is stored.
type ManjoConfig struct {
	TenantID        string `json:"tenant_id"`
	ClientID        string `json:"client_id"`
	MerchantID      string `json:"merchant_id"`
	PartnerID       string `json:"partner_id"`
	Environment     string `json:"environment"`
	HasClientSecret bool   `json:"has_client_secret"`
	HasPrivateKey   bool   `json:"has_private_key"`
	IsConfigured    bool   `json:"is_configured"`
}

// UpdateManjoConfigRequest updates the Manjo configuration. An empty
// ClientSecret or PrivateKey keeps the stored value.
type UpdateManjoConfigRequest struct {
	TenantID     string `json:"-"`
	ClientID     string `json:"client_id"`
	MerchantID   string `json:"merchant_id"`
	PartnerID    string `json:"partner_id"`
	Environment  string `json:"environment"`
	ClientSecret string `json:"client_secret"`
	PrivateKey   string `json:"private_key"`
}

// ManjoCredentials is the full Manjo configuration, secrets included, for
// order-service only (internal endpoint, never exposed through the gateway).
type ManjoCredentials struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	PrivateKey   string `json:"private_key"`
	MerchantID   string `json:"merchant_id"`
	PartnerID    string `json:"partner_id"`
	Environment  string `json:"environment"`
	IsConfigured bool   `json:"is_configured"`
}

// GatewayStatus summarises one payment gateway of a tenant.
type GatewayStatus struct {
	Configured  bool   `json:"configured"`
	Environment string `json:"environment"`
}

// PaymentGatewaySettings lists the tenant's gateways and which one is the default.
type PaymentGatewaySettings struct {
	TenantID       string                   `json:"tenant_id"`
	DefaultGateway string                   `json:"default_gateway"`
	Gateways       map[string]GatewayStatus `json:"gateways"`
}

func (s *TenantConfigService) GetManjoConfig(ctx context.Context, tenantID string) (*ManjoConfig, error) {
	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}
	return &ManjoConfig{
		TenantID:        tenantID,
		ClientID:        config.ManjoClientID,
		MerchantID:      config.ManjoMerchantID,
		PartnerID:       config.ManjoPartnerID,
		Environment:     config.ManjoEnvironment,
		HasClientSecret: config.ManjoClientSecret != "",
		HasPrivateKey:   config.ManjoPrivateKey != "",
		IsConfigured:    config.ManjoConfigured(),
	}, nil
}

func (s *TenantConfigService) GetManjoCredentials(ctx context.Context, tenantID string) (*ManjoCredentials, error) {
	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}
	partnerID := config.ManjoPartnerID
	if partnerID == "" {
		partnerID = config.ManjoClientID
	}
	return &ManjoCredentials{
		TenantID:     tenantID,
		ClientID:     config.ManjoClientID,
		ClientSecret: config.ManjoClientSecret,
		PrivateKey:   config.ManjoPrivateKey,
		MerchantID:   config.ManjoMerchantID,
		PartnerID:    partnerID,
		Environment:  config.ManjoEnvironment,
		IsConfigured: config.ManjoConfigured(),
	}, nil
}

func (s *TenantConfigService) UpdateManjoConfig(ctx context.Context, req *UpdateManjoConfigRequest) error {
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.MerchantID = strings.TrimSpace(req.MerchantID)
	req.PartnerID = strings.TrimSpace(req.PartnerID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	req.PrivateKey = strings.TrimSpace(req.PrivateKey)

	if !validEnvironment(req.Environment) {
		return &ValidationError{Message: "invalid environment: must be 'sandbox' or 'production'"}
	}
	if req.ClientID == "" || req.MerchantID == "" {
		return &ValidationError{Message: "client_id and merchant_id are required"}
	}
	if req.PrivateKey != "" {
		if err := validateRSAPrivateKey(req.PrivateKey); err != nil {
			return &ValidationError{Message: err.Error()}
		}
	}

	config, err := s.configRepo.GetByTenantID(ctx, req.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant config: %w", err)
	}

	config.ManjoClientID = req.ClientID
	config.ManjoMerchantID = req.MerchantID
	config.ManjoPartnerID = req.PartnerID
	config.ManjoEnvironment = req.Environment
	if req.ClientSecret != "" {
		config.ManjoClientSecret = req.ClientSecret
	}
	if req.PrivateKey != "" {
		config.ManjoPrivateKey = req.PrivateKey
	}
	if config.ManjoClientSecret == "" || config.ManjoPrivateKey == "" {
		return &ValidationError{Message: "client_secret and private_key are required"}
	}

	if config.CreatedAt == "" {
		return s.configRepo.Create(ctx, config)
	}
	return s.configRepo.Update(ctx, config)
}

func (s *TenantConfigService) GetPaymentGatewaySettings(ctx context.Context, tenantID string) (*PaymentGatewaySettings, error) {
	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}
	gateway := config.PaymentGateway
	if gateway == "" {
		gateway = repository.PaymentGatewayMidtrans
	}
	return &PaymentGatewaySettings{
		TenantID:       tenantID,
		DefaultGateway: gateway,
		Gateways: map[string]GatewayStatus{
			repository.PaymentGatewayMidtrans: {Configured: config.MidtransConfigured(), Environment: config.MidtransEnvironment},
			repository.PaymentGatewayManjo:    {Configured: config.ManjoConfigured(), Environment: config.ManjoEnvironment},
		},
	}, nil
}

// SetDefaultPaymentGateway switches the gateway used for new guest orders. The
// gateway must already be configured, so guest ordering keeps working.
func (s *TenantConfigService) SetDefaultPaymentGateway(ctx context.Context, tenantID, gateway string) error {
	if gateway != repository.PaymentGatewayMidtrans && gateway != repository.PaymentGatewayManjo {
		return &ValidationError{Message: "invalid payment gateway: must be 'midtrans' or 'manjo'"}
	}

	config, err := s.configRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant config: %w", err)
	}

	configured := config.MidtransConfigured()
	if gateway == repository.PaymentGatewayManjo {
		configured = config.ManjoConfigured()
	}
	if !configured {
		return &ValidationError{Message: fmt.Sprintf("%s is not configured yet; save its credentials first", gateway)}
	}

	config.PaymentGateway = gateway
	if config.CreatedAt == "" {
		return s.configRepo.Create(ctx, config)
	}
	return s.configRepo.Update(ctx, config)
}

func validateRSAPrivateKey(pemKey string) error {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return errors.New("private_key must be a PEM encoded RSA private key")
	}
	if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return errors.New("private_key must be a PEM encoded RSA private key")
	}
	if _, ok := key.(*rsa.PrivateKey); !ok {
		return errors.New("private_key must be an RSA key")
	}
	return nil
}
