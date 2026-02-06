package config

import (
	"fmt"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

var (
	vaultClient     *vault.Client
	vaultClientOnce sync.Once
)

// InitVaultClient initializes HashiCorp Vault client (singleton pattern)
func InitVaultClient(address, token string) error {
	var err error
	vaultClientOnce.Do(func() {
		config := vault.DefaultConfig()
		config.Address = address

		vaultClient, err = vault.NewClient(config)
		if err != nil {
			err = fmt.Errorf("failed to create Vault client: %w", err)
			return
		}

		vaultClient.SetToken(token)
	})
	return err
}

// GetVaultClient returns the singleton Vault client instance
func GetVaultClient() *vault.Client {
	if vaultClient == nil {
		panic("Vault client not initialized. Call InitVaultClient first.")
	}
	return vaultClient
}
