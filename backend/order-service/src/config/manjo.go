package config

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/point-of-sale-system/order-service/src/manjo"
)

// Payment gateways a tenant can choose for guest QRIS payments.
const (
	PaymentGatewayMidtrans = "midtrans"
	PaymentGatewayManjo    = "manjo"
)

// TenantManjoConfig is the response of tenant-service's internal
// /internal/tenants/:id/manjo-credentials endpoint.
type TenantManjoConfig struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	MerchantID   string `json:"merchant_id"`
	PartnerID    string `json:"partner_id"`
	Environment  string `json:"environment"`
	IsConfigured bool   `json:"is_configured"`
}

// Credentials converts the tenant config to client credentials.
func (c *TenantManjoConfig) Credentials() manjo.Credentials {
	return manjo.Credentials{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		MerchantID:   c.MerchantID,
		PartnerID:    c.PartnerID,
		Environment:  c.Environment,
	}
}

var tenantConfigHTTPClient = &http.Client{Timeout: 10 * time.Second}

// GetManjoConfigForTenant fetches the tenant's Manjo credentials from tenant-service.
func GetManjoConfigForTenant(ctx context.Context, tenantID string) (*TenantManjoConfig, error) {
	var cfg TenantManjoConfig
	url := fmt.Sprintf("%s/internal/tenants/%s/manjo-credentials", tenantServiceURL, tenantID)
	if err := getTenantServiceJSON(ctx, url, &cfg); err != nil {
		return nil, fmt.Errorf("failed to fetch tenant Manjo config: %w", err)
	}
	return &cfg, nil
}

// GetPaymentGatewayForTenant returns the tenant's default payment gateway.
func GetPaymentGatewayForTenant(ctx context.Context, tenantID string) (string, error) {
	var status struct {
		PaymentGateway string `json:"payment_gateway"`
	}
	url := fmt.Sprintf("%s/internal/tenants/%s/status", tenantServiceURL, tenantID)
	if err := getTenantServiceJSON(ctx, url, &status); err != nil {
		return "", fmt.Errorf("failed to fetch tenant payment gateway: %w", err)
	}
	if status.PaymentGateway == "" {
		return PaymentGatewayMidtrans, nil
	}
	return status.PaymentGateway, nil
}

func getTenantServiceJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := tenantConfigHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tenant-service returned status: %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var (
	manjoClientsMu sync.Mutex
	manjoClients   = map[string]*manjo.Client{}
)

// GetManjoClient returns the shared Manjo client for an environment
// ("sandbox" or "production"). Base URLs come from MANJO_SANDBOX_BASE_URL and
// MANJO_PRODUCTION_BASE_URL; access token requests are signed with the
// platform key in MANJO_PRIVATE_KEY. Clients are shared so access tokens stay
// cached.
func GetManjoClient(environment string) (*manjo.Client, error) {
	envVar := "MANJO_SANDBOX_BASE_URL"
	if environment == "production" {
		envVar = "MANJO_PRODUCTION_BASE_URL"
	}
	baseURL := strings.TrimSpace(os.Getenv(envVar))
	if baseURL == "" {
		return nil, fmt.Errorf("%s is not set", envVar)
	}

	manjoClientsMu.Lock()
	defer manjoClientsMu.Unlock()
	if client, ok := manjoClients[baseURL]; ok {
		return client, nil
	}
	privateKey, err := manjoPrivateKey()
	if err != nil {
		return nil, err
	}
	opts := []manjo.Option{manjo.WithPrivateKey(privateKey)}
	if ip := strings.TrimSpace(os.Getenv("MANJO_MERCHANT_IP")); ip != "" {
		opts = append(opts, manjo.WithIPAddress(ip))
	}
	client := manjo.NewClient(baseURL, opts...)
	manjoClients[baseURL] = client
	return client, nil
}

func manjoPrivateKey() (*rsa.PrivateKey, error) {
	value := os.Getenv("MANJO_PRIVATE_KEY")
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("MANJO_PRIVATE_KEY is not set")
	}
	return manjo.ParsePrivateKey(value)
}

// ManjoPublicKeys returns the Manjo public keys that verify the access token
// requests Manjo sends us before a payment notification, per environment, from
// MANJO_SANDBOX_PUBLIC_KEY and MANJO_PRODUCTION_PUBLIC_KEY. Unset keys are
// left out; a key that is set but invalid is an error.
func ManjoPublicKeys() (map[string]*rsa.PublicKey, error) {
	keys := map[string]*rsa.PublicKey{}
	for environment, envVar := range map[string]string{
		"sandbox":    "MANJO_SANDBOX_PUBLIC_KEY",
		"production": "MANJO_PRODUCTION_PUBLIC_KEY",
	} {
		value := os.Getenv(envVar)
		if strings.TrimSpace(value) == "" {
			continue
		}
		key, err := manjo.ParsePublicKey(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", envVar, err)
		}
		keys[environment] = key
	}
	return keys, nil
}
