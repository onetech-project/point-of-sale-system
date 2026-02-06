package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

// Config holds all migration configuration
type Config struct {
	DatabaseURL     string
	VaultAddr       string
	VaultToken      string
	VaultTransitKey string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		VaultAddr:       os.Getenv("VAULT_ADDR"),
		VaultToken:      os.Getenv("VAULT_TOKEN"),
		VaultTransitKey: os.Getenv("VAULT_TRANSIT_KEY"),
	}

	// Validate required fields
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}
	if config.VaultAddr == "" {
		return nil, fmt.Errorf("VAULT_ADDR environment variable not set")
	}
	if config.VaultToken == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable not set")
	}
	if config.VaultTransitKey == "" {
		return nil, fmt.Errorf("VAULT_TRANSIT_KEY environment variable not set")
	}

	return config, nil
}

// VaultClient handles encryption/decryption via Vault Transit Engine
type VaultClient struct {
	client     *vault.Client
	transitKey string
	hmacSecret []byte
	mu         sync.RWMutex
}

var (
	vaultClientInstance *VaultClient
	vaultClientOnce     sync.Once
)

// NewVaultClient creates a singleton Vault client instance
func NewVaultClient(config *Config) (*VaultClient, error) {
	var initErr error
	vaultClientOnce.Do(func() {
		vaultConfig := vault.DefaultConfig()
		vaultConfig.Address = config.VaultAddr

		client, err := vault.NewClient(vaultConfig)
		if err != nil {
			initErr = fmt.Errorf("failed to create Vault client: %w", err)
			return
		}

		client.SetToken(config.VaultToken)

		// Generate HMAC secret from transit key (for integrity verification)
		hmacSecret := sha256.Sum256([]byte(config.VaultTransitKey + "-hmac-secret"))

		vaultClientInstance = &VaultClient{
			client:     client,
			transitKey: config.VaultTransitKey,
			hmacSecret: hmacSecret[:],
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return vaultClientInstance, nil
}

// Encrypt encrypts plaintext using Vault Transit Engine
func (vc *VaultClient) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	path := fmt.Sprintf("transit/encrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	secret, err := vc.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", fmt.Errorf("vault encrypt failed: %w", err)
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		return "", fmt.Errorf("vault encrypt returned no ciphertext")
	}

	ciphertext := secret.Data["ciphertext"].(string)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using Vault Transit Engine
// Verifies HMAC integrity if present (format: vault:v1:ciphertext:hmac)
func (vc *VaultClient) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Parse vault:v1:data:hmac format
	// HMAC is optional - only present if string ends with :HEXSTRING (64 hex chars)
	var vaultCiphertext, providedHmac string
	lastColonIdx := strings.LastIndex(ciphertext, ":")

	// Check if suffix after last colon looks like HMAC (64 hex characters)
	isHmacPresent := false
	if lastColonIdx != -1 && lastColonIdx < len(ciphertext)-1 {
		suffix := ciphertext[lastColonIdx+1:]
		// HMAC is 64 hex chars (SHA256 = 32 bytes = 64 hex)
		if len(suffix) == 64 {
			// Check if all chars are hex
			allHex := true
			for _, c := range suffix {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
					allHex = false
					break
				}
			}
			isHmacPresent = allHex
		}
	}

	if isHmacPresent {
		vaultCiphertext = ciphertext[:lastColonIdx]
		providedHmac = ciphertext[lastColonIdx+1:]
	} else {
		// No HMAC, entire string is vault ciphertext
		vaultCiphertext = ciphertext
	}

	if vaultCiphertext == "" {
		return "", fmt.Errorf("invalid ciphertext format")
	}

	// Verify HMAC integrity if present
	if providedHmac != "" {
		mac := hmac.New(sha256.New, vc.hmacSecret)
		mac.Write([]byte(vaultCiphertext))
		expectedHmac := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(providedHmac), []byte(expectedHmac)) {
			return "", fmt.Errorf("HMAC integrity verification failed - data tampering detected")
		}
	}

	// Call Vault Transit Engine Decrypt API
	path := fmt.Sprintf("transit/decrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"ciphertext": vaultCiphertext,
	}

	secret, err := vc.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", fmt.Errorf("vault decrypt failed: %w", err)
	}

	if secret == nil || secret.Data["plaintext"] == nil {
		return "", fmt.Errorf("vault decrypt returned no plaintext")
	}

	plaintextBase64 := secret.Data["plaintext"].(string)
	plaintext, err := base64.StdEncoding.DecodeString(plaintextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode plaintext: %w", err)
	}

	return string(plaintext), nil
}

// GetTransitKey returns the transit key name
func (vc *VaultClient) GetTransitKey() string {
	return vc.transitKey
}

// GetClient returns the underlying Vault client
func (vc *VaultClient) GetClient() *vault.Client {
	return vc.client
}

// ComputeHMAC computes HMAC for integrity verification
func ComputeHMAC(key, data string) string {
	h := sha256.New()
	h.Write([]byte(key + data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
