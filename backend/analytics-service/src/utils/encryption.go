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

// Encryptor defines the interface for encryption/decryption operations
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error)
	DecryptBatch(ctx context.Context, ciphertexts []string, encryptionContexts []string) ([]string, error)
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
func NewVaultClient() (*VaultClient, error) {
	var initErr error
	vaultClientOnce.Do(func() {
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

// Encrypt encrypts plaintext using Vault Transit Engine
func (vc *VaultClient) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return vc.EncryptWithContext(ctx, plaintext, "")
}

// EncryptWithContext encrypts plaintext with convergent encryption context
func (vc *VaultClient) EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	path := fmt.Sprintf("transit/encrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	if encryptionContext != "" {
		data["context"] = base64.StdEncoding.EncodeToString([]byte(encryptionContext))
	}

	secret, err := vc.client.Logical().Write(path, data)
	if err != nil {
		return "", fmt.Errorf("vault encrypt failed: %w", err)
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		return "", fmt.Errorf("vault encrypt returned no ciphertext")
	}

	ciphertext := secret.Data["ciphertext"].(string)

	// Generate HMAC for integrity verification
	mac := hmac.New(sha256.New, vc.hmacSecret)
	mac.Write([]byte(ciphertext))
	hmacHex := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s:%s", ciphertext, hmacHex), nil
}

// Decrypt decrypts ciphertext using Vault Transit Engine
func (vc *VaultClient) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return vc.DecryptWithContext(ctx, ciphertext, "")
}

// DecryptWithContext decrypts ciphertext with convergent encryption context
func (vc *VaultClient) DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Parse vault:v1:data:hmac format
	var vaultCiphertext, providedHmac string
	lastColonIdx := strings.LastIndex(ciphertext, ":")

	// Check if suffix after last colon looks like HMAC (64 hex characters)
	isHmacPresent := false
	if lastColonIdx != -1 && lastColonIdx < len(ciphertext)-1 {
		suffix := ciphertext[lastColonIdx+1:]
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
		vaultCiphertext = ciphertext[:lastColonIdx]
		providedHmac = ciphertext[lastColonIdx+1:]
	} else {
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

	// Call Vault Transit Engine Decrypt API with context
	path := fmt.Sprintf("transit/decrypt/%s", vc.transitKey)
	data := map[string]interface{}{
		"ciphertext": vaultCiphertext,
	}

	if encryptionContext != "" {
		data["context"] = base64.StdEncoding.EncodeToString([]byte(encryptionContext))
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

// DecryptBatch decrypts multiple ciphertexts in a single Vault API call
// encryptionContexts: slice of encryption contexts (one per ciphertext). Pass empty string for no context.
func (vc *VaultClient) DecryptBatch(ctx context.Context, ciphertexts []string, encryptionContexts []string) ([]string, error) {
	if len(ciphertexts) == 0 {
		return []string{}, nil
	}

	// Validate that contexts match ciphertexts length
	if len(encryptionContexts) != len(ciphertexts) {
		return nil, fmt.Errorf("encryption contexts length (%d) must match ciphertexts length (%d)", len(encryptionContexts), len(ciphertexts))
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

		// Parse and verify HMAC
		var vaultCiphertext, providedHmac string
		lastColonIdx := strings.LastIndex(ct, ":")

		// Check if suffix after last colon looks like HMAC (64 hex characters)
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

		if providedHmac != "" {
			mac := hmac.New(sha256.New, vc.hmacSecret)
			mac.Write([]byte(vaultCiphertext))
			expectedHmac := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(providedHmac), []byte(expectedHmac)) {
				return nil, fmt.Errorf("HMAC integrity verification failed for item %d", i)
			}
		}

		batchItem := map[string]interface{}{
			"ciphertext": vaultCiphertext,
		}

		// Add encryption context if provided
		if encryptionContexts[i] != "" {
			batchItem["context"] = base64.StdEncoding.EncodeToString([]byte(encryptionContexts[i]))
		}

		batchInput[i] = batchItem
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

		if resultMap["plaintext"] == nil {
			plaintexts[i] = ""
			continue
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
	return nil
}

// HashForSearch creates a deterministic HMAC-SHA256 hash for searching encrypted fields
func HashForSearch(value string) string {
	secretKey := GetEnv("SEARCH_HASH_SECRET")
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}
