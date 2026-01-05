package utils

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Encryptor interface for encryption operations
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
}

// VaultClient implements Encryptor using HashiCorp Vault Transit Engine
type VaultClient struct {
	address    string
	token      string
	transitKey string
	httpClient *http.Client
	hmacSecret []byte
}

// NewVaultClient creates a new Vault client for encryption
func NewVaultClient() (*VaultClient, error) {
	address := os.Getenv("VAULT_ADDR")
	if address == "" {
		address = "http://vault:8200"
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable not set")
	}

	transitKey := os.Getenv("VAULT_TRANSIT_KEY")
	if transitKey == "" {
		transitKey = "pos-encryption-key"
	}

	// Generate HMAC secret from transit key
	h := sha256.New()
	h.Write([]byte(transitKey + "-hmac-secret"))
	hmacSecret := h.Sum(nil)

	return &VaultClient{
		address:    address,
		token:      token,
		transitKey: transitKey,
		httpClient: &http.Client{},
		hmacSecret: hmacSecret,
	}, nil
}

// Encrypt encrypts plaintext using Vault Transit Engine with HMAC
func (v *VaultClient) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	url := fmt.Sprintf("%s/v1/transit/encrypt/%s", v.address, v.transitKey)

	payload := map[string]interface{}{
		"plaintext": plaintext,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal encrypt request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create encrypt request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault encrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault encrypt failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode encrypt response: %w", err)
	}

	// Generate HMAC for integrity verification
	mac := hmac.New(sha256.New, v.hmacSecret)
	mac.Write([]byte(result.Data.Ciphertext))
	hmacHex := hex.EncodeToString(mac.Sum(nil))

	// Return ciphertext with HMAC appended
	return result.Data.Ciphertext + ":" + hmacHex, nil
}

// Decrypt decrypts ciphertext using Vault Transit Engine with HMAC verification
func (v *VaultClient) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Check for HMAC suffix (64 hex characters after last colon)
	var vaultCiphertext, providedHmac string
	lastColonIdx := strings.LastIndex(ciphertext, ":")
	if lastColonIdx != -1 {
		suffix := ciphertext[lastColonIdx+1:]
		if len(suffix) == 64 && isAllHex(suffix) {
			vaultCiphertext = ciphertext[:lastColonIdx]
			providedHmac = suffix
		} else {
			vaultCiphertext = ciphertext
		}
	} else {
		vaultCiphertext = ciphertext
	}

	// Verify HMAC if present
	if providedHmac != "" {
		mac := hmac.New(sha256.New, v.hmacSecret)
		mac.Write([]byte(vaultCiphertext))
		expectedHmac := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(providedHmac), []byte(expectedHmac)) {
			return "", fmt.Errorf("HMAC verification failed: data may have been tampered with")
		}
	}

	url := fmt.Sprintf("%s/v1/transit/decrypt/%s", v.address, v.transitKey)

	payload := map[string]interface{}{
		"ciphertext": vaultCiphertext,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal decrypt request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create decrypt request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault decrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault decrypt failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Plaintext string `json:"plaintext"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode decrypt response: %w", err)
	}

	return result.Data.Plaintext, nil
}

// isAllHex checks if a string contains only hexadecimal characters
func isAllHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
