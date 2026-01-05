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

	"github.com/point-of-sale-system/order-service/src/config"
)

var mu sync.RWMutex

// Encryptor interface enables dependency injection for encryption/decryption operations
// This allows repositories to accept mock encryptors for testing without Vault dependency
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)
	DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)
}

// VaultEncryptor wraps VaultClient to implement the Encryptor interface
type VaultEncryptor struct {
	vaultClient *config.VaultClient
}

// NewVaultEncryptor creates a new VaultEncryptor instance
func NewVaultEncryptor() (*VaultEncryptor, error) {
	vaultClient := config.GetVaultClient()
	if vaultClient == nil {
		return nil, fmt.Errorf("vault client not initialized")
	}
	return &VaultEncryptor{vaultClient: vaultClient}, nil
}

// Encrypt implements Encryptor interface
func (v *VaultEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return Encrypt(ctx, plaintext)
}

// Decrypt implements Encryptor interface
func (v *VaultEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return Decrypt(ctx, ciphertext)
}

// EncryptBatch implements Encryptor interface
func (v *VaultEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	return EncryptBatch(ctx, plaintexts)
}

// DecryptBatch implements Encryptor interface
func (v *VaultEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	return DecryptBatch(ctx, ciphertexts)
}

// Encrypt encrypts plaintext using Vault Transit Engine Encrypt API
// Returns base64-encoded ciphertext with HMAC for integrity verification
// Format: "vault:v1:<base64_ciphertext>:<hex_hmac>"
func Encrypt(ctx context.Context, plaintext string) (string, error) {
	vc := config.GetVaultClient()
	if vc == nil {
		return "", fmt.Errorf("vault client not initialized")
	}

	if plaintext == "" {
		return "", nil // Don't encrypt empty strings
	}

	mu.RLock()
	defer mu.RUnlock()

	// Call Vault Transit Engine Encrypt API
	path := fmt.Sprintf("transit/encrypt/%s", vc.TransitKey)
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	secret, err := vc.Client.Logical().Write(path, data)
	if err != nil {
		return "", fmt.Errorf("vault encrypt failed: %w", err)
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		return "", fmt.Errorf("vault encrypt returned no ciphertext")
	}

	ciphertext := secret.Data["ciphertext"].(string)

	// Generate HMAC for integrity verification (FR-012)
	mac := hmac.New(sha256.New, vc.HmacSecret)
	mac.Write([]byte(ciphertext))
	hmacHex := hex.EncodeToString(mac.Sum(nil))

	// Return format: ciphertext:hmac
	return fmt.Sprintf("%s:%s", ciphertext, hmacHex), nil
}

// Decrypt decrypts ciphertext using Vault Transit Engine Decrypt API
// Verifies HMAC integrity before decryption (FR-012)
func Decrypt(ctx context.Context, ciphertext string) (string, error) {
	vc := config.GetVaultClient()
	if vc == nil {
		return "", fmt.Errorf("vault client not initialized")
	}

	if ciphertext == "" {
		return "", nil // Don't decrypt empty strings
	}

	mu.RLock()
	defer mu.RUnlock()

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
		mac := hmac.New(sha256.New, vc.HmacSecret)
		mac.Write([]byte(vaultCiphertext))
		expectedHmac := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(providedHmac), []byte(expectedHmac)) {
			return "", fmt.Errorf("HMAC integrity verification failed - data tampering detected")
		}
	}

	// Call Vault Transit Engine Decrypt API
	path := fmt.Sprintf("transit/decrypt/%s", vc.TransitKey)
	data := map[string]interface{}{
		"ciphertext": vaultCiphertext,
	}

	secret, err := vc.Client.Logical().Write(path, data)
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
func EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	vc := config.GetVaultClient()
	if vc == nil {
		return nil, fmt.Errorf("vault client not initialized")
	}

	if len(plaintexts) == 0 {
		return []string{}, nil
	}

	mu.RLock()
	defer mu.RUnlock()

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

	path := fmt.Sprintf("transit/encrypt/%s", vc.TransitKey)
	data := map[string]interface{}{
		"batch_input": batchInput,
	}

	secret, err := vc.Client.Logical().Write(path, data)
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
		mac := hmac.New(sha256.New, vc.HmacSecret)
		mac.Write([]byte(ciphertext))
		hmacHex := hex.EncodeToString(mac.Sum(nil))

		ciphertexts[i] = fmt.Sprintf("%s:%s", ciphertext, hmacHex)
	}

	return ciphertexts, nil
}

// DecryptBatch decrypts multiple ciphertexts in a single Vault API call (performance optimization)
func DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	vc := config.GetVaultClient()
	if vc == nil {
		return nil, fmt.Errorf("vault client not initialized")
	}

	if len(ciphertexts) == 0 {
		return []string{}, nil
	}

	mu.RLock()
	defer mu.RUnlock()

	// Prepare batch items and verify HMACs
	batchInput := make([]map[string]interface{}, len(ciphertexts))
	for i, ct := range ciphertexts {
		if ct == "" {
			batchInput[i] = map[string]interface{}{"ciphertext": ""}
			continue
		}

		// Parse and verify HMAC
		var vaultCiphertext, providedHmac string
		fmt.Sscanf(ct, "%[^:]:%s", &vaultCiphertext, &providedHmac)

		if providedHmac != "" {
			mac := hmac.New(sha256.New, vc.HmacSecret)
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

	path := fmt.Sprintf("transit/decrypt/%s", vc.TransitKey)
	data := map[string]interface{}{
		"batch_input": batchInput,
	}

	secret, err := vc.Client.Logical().Write(path, data)
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
