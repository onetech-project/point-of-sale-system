package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/pos/tenant-service/src/utils"
)

type TenantConfigRepository struct {
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher *utils.AuditPublisher
}

// NewTenantConfigRepository creates a repository with custom encryptor (for testing)
func NewTenantConfigRepository(db *sql.DB, encryptor utils.Encryptor, auditPublisher *utils.AuditPublisher) *TenantConfigRepository {
	return &TenantConfigRepository{
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// NewTenantConfigRepositoryWithVault creates a repository with Vault encryption (production)
func NewTenantConfigRepositoryWithVault(db *sql.DB, auditPublisher *utils.AuditPublisher) (*TenantConfigRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	return &TenantConfigRepository{
		db:             db,
		encryptor:      vaultClient,
		auditPublisher: auditPublisher,
	}, nil
}

type TenantConfig struct {
	TenantID             string                 `json:"tenant_id"`
	EnabledDeliveryTypes []string               `json:"enabled_delivery_types"`
	ServiceArea          map[string]interface{} `json:"service_area"`
	DeliveryFeeConfig    map[string]interface{} `json:"delivery_fee_config"`
	AutoCalculateFees    bool                   `json:"auto_calculate_fees"`
	MidtransServerKey    string                 `json:"midtrans_server_key,omitempty"`
	MidtransClientKey    string                 `json:"midtrans_client_key,omitempty"`
	MidtransMerchantID   string                 `json:"midtrans_merchant_id,omitempty"`
	MidtransEnvironment  string                 `json:"midtrans_environment"`
	PaymentGateway       string                 `json:"payment_gateway"`
	ManjoClientID        string                 `json:"manjo_client_id,omitempty"`
	ManjoClientSecret    string                 `json:"manjo_client_secret,omitempty"`
	ManjoMerchantID      string                 `json:"manjo_merchant_id,omitempty"`
	ManjoPartnerID       string                 `json:"manjo_partner_id,omitempty"`
	ManjoEnvironment     string                 `json:"manjo_environment"`
	CreatedAt            string                 `json:"created_at"`
	UpdatedAt            string                 `json:"updated_at"`
}

func (r *TenantConfigRepository) GetByTenantID(ctx context.Context, tenantID string) (*TenantConfig, error) {
	query := `
		SELECT 
			tenant_id,
			enabled_delivery_types,
			COALESCE(service_area_data, '{}'::jsonb) as service_area,
			COALESCE(delivery_fee_config, '{}'::jsonb) as delivery_fee_config,
			COALESCE(enable_delivery_fee_calculation, false) as auto_calculate_fees,
			COALESCE(midtrans_server_key, '') as midtrans_server_key,
			COALESCE(midtrans_client_key, '') as midtrans_client_key,
			COALESCE(midtrans_merchant_id, '') as midtrans_merchant_id,
			COALESCE(midtrans_environment, 'sandbox') as midtrans_environment,
			COALESCE(payment_gateway, 'midtrans') as payment_gateway,
			COALESCE(manjo_client_id, '') as manjo_client_id,
			COALESCE(manjo_client_secret, '') as manjo_client_secret,
			COALESCE(manjo_merchant_id, '') as manjo_merchant_id,
			COALESCE(manjo_partner_id, '') as manjo_partner_id,
			COALESCE(manjo_environment, 'sandbox') as manjo_environment,
			created_at,
			updated_at
		FROM tenant_configs
		WHERE tenant_id = $1
	`

	var config TenantConfig
	var serviceArea, deliveryFeeConfig []byte
	var encryptedServerKey, encryptedClientKey string
	var encryptedManjoSecret string

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.TenantID,
		pq.Array(&config.EnabledDeliveryTypes),
		&serviceArea,
		&deliveryFeeConfig,
		&config.AutoCalculateFees,
		&encryptedServerKey,
		&encryptedClientKey,
		&config.MidtransMerchantID,
		&config.MidtransEnvironment,
		&config.PaymentGateway,
		&config.ManjoClientID,
		&encryptedManjoSecret,
		&config.ManjoMerchantID,
		&config.ManjoPartnerID,
		&config.ManjoEnvironment,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default configuration if none exists
		return &TenantConfig{
			TenantID:             tenantID,
			EnabledDeliveryTypes: []string{"pickup", "delivery", "dine_in"},
			ServiceArea:          map[string]interface{}{},
			DeliveryFeeConfig:    map[string]interface{}{},
			AutoCalculateFees:    false,
			MidtransEnvironment:  "sandbox",
			PaymentGateway:       PaymentGatewayMidtrans,
			ManjoEnvironment:     "sandbox",
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	// Decrypt Midtrans keys with context
	if encryptedServerKey != "" {
		config.MidtransServerKey, err = r.encryptor.DecryptWithContext(ctx, encryptedServerKey, "tenant_config:midtrans_server_key")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt midtrans_server_key: %w", err)
		}
	}

	if encryptedClientKey != "" {
		config.MidtransClientKey, err = r.encryptor.DecryptWithContext(ctx, encryptedClientKey, "tenant_config:midtrans_client_key")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt midtrans_client_key: %w", err)
		}
	}

	if config.ManjoClientSecret, err = r.decrypt(ctx, encryptedManjoSecret, "manjo_client_secret"); err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(serviceArea, &config.ServiceArea); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service_area: %w", err)
	}

	if err := json.Unmarshal(deliveryFeeConfig, &config.DeliveryFeeConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delivery_fee_config: %w", err)
	}

	return &config, nil
}

func (r *TenantConfigRepository) Create(ctx context.Context, config *TenantConfig) error {
	// Encrypt Midtrans keys with context
	var encryptedServerKey, encryptedClientKey string
	var err error

	if config.MidtransServerKey != "" {
		encryptedServerKey, err = r.encryptor.EncryptWithContext(ctx, config.MidtransServerKey, "tenant_config:midtrans_server_key")
		if err != nil {
			return fmt.Errorf("failed to encrypt midtrans_server_key: %w", err)
		}
	}

	if config.MidtransClientKey != "" {
		encryptedClientKey, err = r.encryptor.EncryptWithContext(ctx, config.MidtransClientKey, "tenant_config:midtrans_client_key")
		if err != nil {
			return fmt.Errorf("failed to encrypt midtrans_client_key: %w", err)
		}
	}

	manjo, err := r.encryptManjoSecrets(ctx, config)
	if err != nil {
		return err
	}

	serviceArea, err := json.Marshal(config.ServiceArea)
	if err != nil {
		return fmt.Errorf("failed to marshal service_area: %w", err)
	}

	deliveryFeeConfig, err := json.Marshal(config.DeliveryFeeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery_fee_config: %w", err)
	}

	query := `
		INSERT INTO tenant_configs (
			tenant_id,
			enabled_delivery_types,
			service_area_data,
			delivery_fee_config,
			enable_delivery_fee_calculation,
			midtrans_server_key,
			midtrans_client_key,
			midtrans_merchant_id,
			midtrans_environment,
			payment_gateway,
			manjo_client_id,
			manjo_client_secret,
			manjo_merchant_id,
			manjo_partner_id,
			manjo_environment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		config.TenantID,
		pq.Array(config.EnabledDeliveryTypes),
		serviceArea,
		deliveryFeeConfig,
		config.AutoCalculateFees,
		encryptedServerKey,
		encryptedClientKey,
		config.MidtransMerchantID,
		config.MidtransEnvironment,
		paymentGatewayOrDefault(config.PaymentGateway),
		config.ManjoClientID,
		manjo.secret,
		config.ManjoMerchantID,
		config.ManjoPartnerID,
		environmentOrDefault(config.ManjoEnvironment),
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant config: %w", err)
	}

	return nil
}

func (r *TenantConfigRepository) Update(ctx context.Context, config *TenantConfig) error {
	// Encrypt Midtrans keys with context
	var encryptedServerKey, encryptedClientKey string
	var err error

	if config.MidtransServerKey != "" {
		encryptedServerKey, err = r.encryptor.EncryptWithContext(ctx, config.MidtransServerKey, "tenant_config:midtrans_server_key")
		if err != nil {
			return fmt.Errorf("failed to encrypt midtrans_server_key: %w", err)
		}
	}

	if config.MidtransClientKey != "" {
		encryptedClientKey, err = r.encryptor.EncryptWithContext(ctx, config.MidtransClientKey, "tenant_config:midtrans_client_key")
		if err != nil {
			return fmt.Errorf("failed to encrypt midtrans_client_key: %w", err)
		}
	}

	manjo, err := r.encryptManjoSecrets(ctx, config)
	if err != nil {
		return err
	}

	serviceArea, err := json.Marshal(config.ServiceArea)
	if err != nil {
		return fmt.Errorf("failed to marshal service_area: %w", err)
	}

	deliveryFeeConfig, err := json.Marshal(config.DeliveryFeeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery_fee_config: %w", err)
	}

	query := `
		UPDATE tenant_configs
		SET
			enabled_delivery_types = $2,
			service_area_data = $3,
			delivery_fee_config = $4,
			enable_delivery_fee_calculation = $5,
			midtrans_server_key = $6,
			midtrans_client_key = $7,
			midtrans_merchant_id = $8,
			midtrans_environment = $9,
			payment_gateway = $10,
			manjo_client_id = $11,
			manjo_client_secret = $12,
			manjo_merchant_id = $13,
			manjo_partner_id = $14,
			manjo_environment = $15,
			updated_at = NOW()
		WHERE tenant_id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		config.TenantID,
		pq.Array(config.EnabledDeliveryTypes),
		serviceArea,
		deliveryFeeConfig,
		config.AutoCalculateFees,
		encryptedServerKey,
		encryptedClientKey,
		config.MidtransMerchantID,
		config.MidtransEnvironment,
		paymentGatewayOrDefault(config.PaymentGateway),
		config.ManjoClientID,
		manjo.secret,
		config.ManjoMerchantID,
		config.ManjoPartnerID,
		environmentOrDefault(config.ManjoEnvironment),
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant config not found")
	}

	// T102: Publish ConfigUpdatedEvent when payment credentials changed
	if r.auditPublisher != nil && (config.MidtransServerKey != "" || config.MidtransClientKey != "" || config.ManjoClientID != "") {
		afterValue := map[string]interface{}{
			"midtrans_server_key":  encryptedServerKey,
			"midtrans_client_key":  encryptedClientKey,
			"midtrans_merchant_id": config.MidtransMerchantID,
			"midtrans_environment": config.MidtransEnvironment,
			"payment_gateway":      paymentGatewayOrDefault(config.PaymentGateway),
			"manjo_client_id":      config.ManjoClientID,
			"manjo_client_secret":  manjo.secret,
			"manjo_merchant_id":    config.ManjoMerchantID,
			"manjo_partner_id":     config.ManjoPartnerID,
			"manjo_environment":    environmentOrDefault(config.ManjoEnvironment),
		}

		auditEvent := &utils.AuditEvent{
			TenantID:     config.TenantID,
			ActorType:    "system",
			Action:       "UPDATE",
			ResourceType: "tenant_config",
			ResourceID:   config.TenantID,
			AfterValue:   afterValue,
			Metadata: map[string]interface{}{
				"config_type": "payment_credentials",
			},
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			fmt.Printf("Failed to publish tenant config update audit event: %v\n", err)
		}
	}

	return nil
}

// Payment gateways a tenant can choose as default for guest QRIS payments.
const (
	PaymentGatewayMidtrans = "midtrans"
	PaymentGatewayManjo    = "manjo"
)

func paymentGatewayOrDefault(gateway string) string {
	if gateway == "" {
		return PaymentGatewayMidtrans
	}
	return gateway
}

func environmentOrDefault(environment string) string {
	if environment == "" {
		return "sandbox"
	}
	return environment
}

type encryptedManjoSecrets struct {
	secret string
}

func (r *TenantConfigRepository) encryptManjoSecrets(ctx context.Context, config *TenantConfig) (encryptedManjoSecrets, error) {
	var out encryptedManjoSecrets
	var err error
	if out.secret, err = r.encrypt(ctx, config.ManjoClientSecret, "manjo_client_secret"); err != nil {
		return out, err
	}
	return out, nil
}

// encrypt encrypts a tenant_configs column value with Vault, bound to the column name.
func (r *TenantConfigRepository) encrypt(ctx context.Context, plaintext, column string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ciphertext, err := r.encryptor.EncryptWithContext(ctx, plaintext, "tenant_config:"+column)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt %s: %w", column, err)
	}
	return ciphertext, nil
}

func (r *TenantConfigRepository) decrypt(ctx context.Context, ciphertext, column string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	plaintext, err := r.encryptor.DecryptWithContext(ctx, ciphertext, "tenant_config:"+column)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt %s: %w", column, err)
	}
	return plaintext, nil
}

// SQL expressions over tenant_configs aliased as tc, shared by every query that
// reports whether guest ordering can take payments.
const (
	MidtransConfiguredSQL = `(COALESCE(tc.midtrans_server_key, '') <> '' AND COALESCE(tc.midtrans_client_key, '') <> '')`
	ManjoConfiguredSQL    = `(COALESCE(tc.manjo_client_id, '') <> '' AND COALESCE(tc.manjo_client_secret, '') <> '' AND COALESCE(tc.manjo_merchant_id, '') <> '')`
	PaymentGatewaySQL     = `COALESCE(tc.payment_gateway, 'midtrans')`
	// PaymentConfiguredSQL is true when the tenant's default gateway has credentials.
	PaymentConfiguredSQL = `(CASE WHEN ` + PaymentGatewaySQL + ` = 'manjo' THEN ` + ManjoConfiguredSQL + ` ELSE ` + MidtransConfiguredSQL + ` END)`
)

// MidtransConfigured reports whether the Midtrans credentials are complete.
func (c *TenantConfig) MidtransConfigured() bool {
	return c.MidtransServerKey != "" && c.MidtransClientKey != ""
}

// ManjoConfigured reports whether the Manjo credentials are complete.
func (c *TenantConfig) ManjoConfigured() bool {
	return c.ManjoClientID != "" && c.ManjoClientSecret != "" && c.ManjoMerchantID != ""
}
