package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

type TenantConfigRepository struct {
	db *sql.DB
}

func NewTenantConfigRepository(db *sql.DB) *TenantConfigRepository {
	return &TenantConfigRepository{db: db}
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
created_at,
updated_at
FROM tenant_configs
WHERE tenant_id = $1
`

	var config TenantConfig
	var serviceArea, deliveryFeeConfig []byte

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.TenantID,
		pq.Array(&config.EnabledDeliveryTypes),
		&serviceArea,
		&deliveryFeeConfig,
		&config.AutoCalculateFees,
		&config.MidtransServerKey,
		&config.MidtransClientKey,
		&config.MidtransMerchantID,
		&config.MidtransEnvironment,
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
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
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
enable_delivery_fee_calculation
) VALUES ($1, $2, $3, $4, $5)
`

	_, err = r.db.ExecContext(
		ctx,
		query,
		config.TenantID,
		pq.Array(config.EnabledDeliveryTypes),
		serviceArea,
		deliveryFeeConfig,
		config.AutoCalculateFees,
	)

	if err != nil {
		return fmt.Errorf("failed to create tenant config: %w", err)
	}

	return nil
}

func (r *TenantConfigRepository) Update(ctx context.Context, config *TenantConfig) error {
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
		config.MidtransServerKey,
		config.MidtransClientKey,
		config.MidtransMerchantID,
		config.MidtransEnvironment,
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

	return nil
}
