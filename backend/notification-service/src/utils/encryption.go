package utils

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	vault "github.com/hashicorp/vault/api"
)

// Encryptor defines the interface for encryption operations
// Enables dependency injection for testing
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)
	DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)
}

// VaultClient handles encryption/decryption via Vault Transit Engine
// Implements FR-009: Secure key storage outside primary data storage
// Implements FR-012: HMAC integrity verification
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
// POST /transit/encrypt/:key_name, POST /transit/decrypt/:key_name
func NewVaultClient() (*VaultClient, error) {
	var initErr error
	vaultClientOnce.Do(func() {
		// All environment variables are mandatory - will panic if not set
		vaultAddr := GetEnv("VAULT_ADDR")
		vaultToken := GetEnv("VAULT_TOKEN")
		transitKey := GetEnv("VAULT_TRANSIT_KEY")

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
			client:     client,
			transitKey: transitKey,
			hmacSecret: hmacSecret[:],
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return vaultClientInstance, nil
}

// Encrypt encrypts plaintext using Vault Transit Engine Encrypt API
// Returns base64-encoded ciphertext with HMAC for integrity verification
// Format: "vault:v1:<base64_ciphertext>:<hex_hmac>"
func (vc *VaultClient) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // Don't encrypt empty strings
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Call Vault Transit Engine Encrypt API
	path := fmt.Sprintf("transit/encrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	secret, err := vc.client.Logical().Write(path, data)
	if err != nil {
		return "", fmt.Errorf("vault encrypt failed: %w", err)
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		return "", fmt.Errorf("vault encrypt returned no ciphertext")
	}

	ciphertext := secret.Data["ciphertext"].(string)

	// Generate HMAC for integrity verification (FR-012)
	mac := hmac.New(sha256.New, vc.hmacSecret)
	mac.Write([]byte(ciphertext))
	hmacHex := hex.EncodeToString(mac.Sum(nil))

	// Return format: ciphertext:hmac
	return fmt.Sprintf("%s:%s", ciphertext, hmacHex), nil
}

// Decrypt decrypts ciphertext using Vault Transit Engine Decrypt API
// Verifies HMAC integrity before decryption (FR-012)
func (vc *VaultClient) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil // Don't decrypt empty strings
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

	// Verify HMAC integrity if present (FR-012)
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

	secret, err := vc.client.Logical().Write(path, data)
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

// EncryptBatch encrypts multiple plaintexts in a single Vault API call (performance optimization)
func (vc *VaultClient) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if len(plaintexts) == 0 {
		return []string{}, nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Prepare batch items
	batchInput := make([]map[string]interface{}, len(plaintexts))
	for i, pt := range plaintexts {
		if pt == "" {
			batchInput[i] = map[string]interface{}{"plaintext": ""}
			continue
		}
		batchInput[i] = map[string]interface{}{
			"plaintext": base64.StdEncoding.EncodeToString([]byte(pt)),
		}
	}

	path := fmt.Sprintf("transit/encrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"batch_input": batchInput,
	}

	secret, err := vc.client.Logical().Write(path, data)
	if err != nil {
		return nil, fmt.Errorf("vault batch encrypt failed: %w", err)
	}

	if secret == nil || secret.Data["batch_results"] == nil {
		return nil, fmt.Errorf("vault batch encrypt returned no results")
	}

	batchResults := secret.Data["batch_results"].([]interface{})
	ciphertexts := make([]string, len(batchResults))

	for i, result := range batchResults {
		resultMap := result.(map[string]interface{})
		if resultMap["error"] != nil {
			return nil, fmt.Errorf("batch encrypt item %d failed: %v", i, resultMap["error"])
		}

		ciphertext := resultMap["ciphertext"].(string)

		// Generate HMAC
		mac := hmac.New(sha256.New, vc.hmacSecret)
		mac.Write([]byte(ciphertext))
		hmacHex := hex.EncodeToString(mac.Sum(nil))

		ciphertexts[i] = fmt.Sprintf("%s:%s", ciphertext, hmacHex)
	}

	return ciphertexts, nil
}

// DecryptBatch decrypts multiple ciphertexts in a single Vault API call (performance optimization)
func (vc *VaultClient) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if len(ciphertexts) == 0 {
		return []string{}, nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Prepare batch items and verify HMACs
	batchInput := make([]map[string]interface{}, len(ciphertexts))
	for i, ct := range ciphertexts {
		if ct == "" {
			batchInput[i] = map[string]interface{}{"ciphertext": ""}
			continue
		}

		// Parse vault:v1:data:hmac format
		var vaultCiphertext, providedHmac string
		lastColonIdx := strings.LastIndex(ct, ":")

		// Check if suffix looks like HMAC (64 hex chars)
		isHmacPresent := false
		if lastColonIdx != -1 && lastColonIdx < len(ct)-1 {
			suffix := ct[lastColonIdx+1:]
			if len(suffix) == 64 {
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
			vaultCiphertext = ct[:lastColonIdx]
			providedHmac = ct[lastColonIdx+1:]
		} else {
			vaultCiphertext = ct
		}

		// Verify HMAC if present
		if providedHmac != "" {
			mac := hmac.New(sha256.New, vc.hmacSecret)
			mac.Write([]byte(vaultCiphertext))
			expectedHmac := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(providedHmac), []byte(expectedHmac)) {
				return nil, fmt.Errorf("HMAC integrity verification failed for item %d", i)
			}
		}

		batchInput[i] = map[string]interface{}{
			"ciphertext": vaultCiphertext,
		}
	}

	path := fmt.Sprintf("transit/decrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"batch_input": batchInput,
	}

	secret, err := vc.client.Logical().Write(path, data)
	if err != nil {
		return nil, fmt.Errorf("vault batch decrypt failed: %w", err)
	}

	if secret == nil || secret.Data["batch_results"] == nil {
		return nil, fmt.Errorf("vault batch decrypt returned no results")
	}

	batchResults := secret.Data["batch_results"].([]interface{})
	plaintexts := make([]string, len(batchResults))

	for i, result := range batchResults {
		resultMap := result.(map[string]interface{})
		if resultMap["error"] != nil {
			return nil, fmt.Errorf("batch decrypt item %d failed: %v", i, resultMap["error"])
		}

		plaintextBase64 := resultMap["plaintext"].(string)
		plaintext, err := base64.StdEncoding.DecodeString(plaintextBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode plaintext for item %d: %w", i, err)
		}

		plaintexts[i] = string(plaintext)
	}

	return plaintexts, nil
}

// Close closes the Vault client connection
func (vc *VaultClient) Close() error {
	// Vault client doesn't require explicit cleanup
	return nil
}
