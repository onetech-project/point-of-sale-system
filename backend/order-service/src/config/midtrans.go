package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/rs/zerolog/log"
)

type MidtransConfig struct {
	ServerKey   string
	ClientKey   string
	Environment midtrans.EnvironmentType
	MerchantID  string
}

// TenantMidtransConfig represents the response from tenant-service
type TenantMidtransConfig struct {
	TenantID     string `json:"tenant_id"`
	ServerKey    string `json:"server_key"`
	ClientKey    string `json:"client_key"`
	MerchantID   string `json:"merchant_id"`
	Environment  string `json:"environment"`
	IsConfigured bool   `json:"is_configured"`
}

var tenantServiceURL string

// InitMidtrans initializes the connection to tenant-service
func InitMidtrans() error {
	tenantServiceURL = os.Getenv("TENANT_SERVICE_URL")
	if tenantServiceURL == "" {
		tenantServiceURL = "http://localhost:8084" // Default for development
	}

	log.Info().
		Str("tenant_service_url", tenantServiceURL).
		Msg("Midtrans config initialized - will fetch per-tenant configuration from tenant-service")

	return nil
}

// GetSnapClientForTenant creates a Snap client for a specific tenant
func GetSnapClientForTenant(ctx context.Context, tenantID string) (*snap.Client, error) {
	config, err := fetchTenantMidtransConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant Midtrans config: %w", err)
	}

	if !config.IsConfigured {
		return nil, fmt.Errorf("Midtrans is not configured for tenant: %s", tenantID)
	}

	env := midtrans.Sandbox
	if config.Environment == "production" {
		env = midtrans.Production
	}

	var snapClient snap.Client
	snapClient.New(config.ServerKey, env)

	return &snapClient, nil
}

// GetCoreAPIClientForTenant creates a Core API client for a specific tenant
func GetCoreAPIClientForTenant(ctx context.Context, tenantID string) (*coreapi.Client, error) {
	config, err := fetchTenantMidtransConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant Midtrans config: %w", err)
	}

	if !config.IsConfigured {
		return nil, fmt.Errorf("Midtrans is not configured for tenant: %s", tenantID)
	}

	env := midtrans.Sandbox
	if config.Environment == "production" {
		env = midtrans.Production
	}

	var coreAPIClient coreapi.Client
	coreAPIClient.New(config.ServerKey, env)

	return &coreAPIClient, nil
}

// GetMidtransServerKeyForTenant returns the Midtrans server key for a specific tenant
func GetMidtransServerKeyForTenant(ctx context.Context, tenantID string) (string, error) {
	config, err := fetchTenantMidtransConfig(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tenant Midtrans config: %w", err)
	}

	if !config.IsConfigured {
		return "", fmt.Errorf("Midtrans is not configured for tenant: %s", tenantID)
	}

	return config.ServerKey, nil
}

// GetMidtransConfigForTenant returns the full Midtrans config for a tenant
func GetMidtransConfigForTenant(ctx context.Context, tenantID string) (*TenantMidtransConfig, error) {
	return fetchTenantMidtransConfig(ctx, tenantID)
}

// GetWebhookURL returns the webhook URL for payment notifications
func GetWebhookURL() string {
	webhookURL := os.Getenv("MIDTRANS_WEBHOOK_URL")
	if webhookURL == "" {
		// Default to API gateway for development
		webhookURL = "http://localhost:8080/api/v1/webhooks/payments/midtrans/notification"
	}
	return webhookURL
}

// fetchTenantMidtransConfig fetches Midtrans configuration from tenant-service
func fetchTenantMidtransConfig(ctx context.Context, tenantID string) (*TenantMidtransConfig, error) {
	url := fmt.Sprintf("%s/api/v1/admin/tenants/%s/midtrans-config", tenantServiceURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenant-service returned status: %d", resp.StatusCode)
	}

	var config TenantMidtransConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &config, nil
}

// Legacy functions for backward compatibility (deprecated)
// These will be removed in future versions

var SnapClient snap.Client
var CoreAPIClient coreapi.Client

// GetSnapClient returns a deprecated global Snap client
// Deprecated: Use GetSnapClientForTenant instead
func GetSnapClient() *snap.Client {
	return &SnapClient
}

// GetCoreAPIClient returns a deprecated global Core API client
// Deprecated: Use GetCoreAPIClientForTenant instead
func GetCoreAPIClient() *coreapi.Client {
	return &CoreAPIClient
}

// GetMidtransServerKey returns a deprecated global server key
// Deprecated: Use GetMidtransServerKeyForTenant instead
func GetMidtransServerKey() string {
	return os.Getenv("MIDTRANS_SERVER_KEY")
}
