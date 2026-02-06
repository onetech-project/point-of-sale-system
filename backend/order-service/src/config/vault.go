package config

import (
	"crypto/sha256"
	"fmt"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

// VaultClient handles encryption/decryption via Vault Transit Engine
// Implements FR-009: Secure key storage outside primary data storage
// Implements FR-012: HMAC integrity verification
type VaultClient struct {
	Client     *vault.Client
	TransitKey string
	HmacSecret []byte
}

var (
	vaultClientInstance *VaultClient
	vaultClientOnce     sync.Once
)

// InitVaultClient creates a singleton Vault client instance
// POST /transit/encrypt/:key_name, POST /transit/decrypt/:key_name
func InitVaultClient() (*VaultClient, error) {
	var initErr error
	vaultClientOnce.Do(func() {
		// All environment variables are mandatory - will panic if not set
		vaultAddr := GetEnvAsString("VAULT_ADDR")
		vaultToken := GetEnvAsString("VAULT_TOKEN")
		transitKey := GetEnvAsString("VAULT_TRANSIT_KEY")

		config := vault.DefaultConfig()
		config.Address = vaultAddr

		client, err := vault.NewClient(config)
		if err != nil {
			initErr = fmt.Errorf("failed to create Vault client: %w", err)
			return
		}

		client.SetToken(vaultToken)

		// Generate HMAC secret from transit key (for integrity verification)
		hmacSecret := sha256.Sum256([]byte(transitKey + "-hmac-secret"))

		vaultClientInstance = &VaultClient{
			Client:     client,
			TransitKey: transitKey,
			HmacSecret: hmacSecret[:],
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return vaultClientInstance, nil
}

// GetVaultClient returns the singleton Vault client instance
func GetVaultClient() *VaultClient {
	return vaultClientInstance
}
