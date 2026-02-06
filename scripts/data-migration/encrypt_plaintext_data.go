package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// vaultClientEncrypt is a simple Vault client for encryption operations
type vaultClientEncrypt struct {
	address    string
	token      string
	keyName    string
	httpClient *http.Client
	hmacSecret string
}

func newVaultClientEncrypt(address, token, keyName, hmacSecret string) *vaultClientEncrypt {
	return &vaultClientEncrypt{
		address: address,
		token:   token,
		keyName: keyName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		hmacSecret: hmacSecret,
	}
}

func (v *vaultClientEncrypt) EncryptWithContext(ctx context.Context, plaintext, encryptionContext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	if encryptionContext != "" {
		data["context"] = base64.StdEncoding.EncodeToString([]byte(encryptionContext))
	}

	reqBody, _ := json.Marshal(data)
	url := fmt.Sprintf("%s/v1/transit/encrypt/%s", v.address, v.keyName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault encrypt request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("vault encrypt failed: %s", string(body))
	}

	var result struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	ciphertext := result.Data.Ciphertext

	// Append HMAC
	if v.hmacSecret != "" {
		h := hmac.New(sha256.New, []byte(v.hmacSecret))
		h.Write([]byte(plaintext))
		hmacValue := hex.EncodeToString(h.Sum(nil))
		ciphertext = fmt.Sprintf("%s:%s", ciphertext, hmacValue)
	}

	return ciphertext, nil
}

// EncryptPlaintextData encrypts plaintext PII data with context-based encryption
func EncryptPlaintextData(config *Config) error {
	log.Printf("=== Encrypt Plaintext Data ===")

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get HMAC secret from environment
	hmacSecret := ""
	if os.Getenv("ENCRYPTION_HMAC_SECRET") != "" {
		hmacSecret = os.Getenv("ENCRYPTION_HMAC_SECRET")
	}

	vault := newVaultClientEncrypt(config.VaultAddr, config.VaultToken, config.VaultTransitKey, hmacSecret)
	ctx := context.Background()

	// Encrypt users
	log.Println("\n=== Users ===")
	if err := encryptUsersTablePlaintext(ctx, db, vault); err != nil {
		return fmt.Errorf("users: %w", err)
	}

	// Encrypt invitations
	log.Println("\n=== Invitations ===")
	if err := encryptInvitationsTablePlaintext(ctx, db, vault); err != nil {
		return fmt.Errorf("invitations: %w", err)
	}

	// Encrypt guest_orders
	log.Println("\n=== Guest Orders ===")
	if err := encryptGuestOrdersTablePlaintext(ctx, db, vault); err != nil {
		return fmt.Errorf("guest_orders: %w", err)
	}

	// Encrypt notifications
	log.Println("\n=== Notifications ===")
	if err := encryptNotificationsTablePlaintext(ctx, db, vault); err != nil {
		return fmt.Errorf("notifications: %w", err)
	}

	log.Println("\n✅ Plaintext encryption complete!")
	return nil
}

func encryptUsersTablePlaintext(ctx context.Context, db *sql.DB, vault *vaultClientEncrypt) error {
	rows, err := db.QueryContext(ctx, "SELECT id, email, first_name, last_name FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, email, firstName, lastName string
		if err := rows.Scan(&id, &email, &firstName, &lastName); err != nil {
			return err
		}

		encEmail, _ := vault.EncryptWithContext(ctx, email, "user:email")
		encFirstName, _ := vault.EncryptWithContext(ctx, firstName, "user:first_name")
		encLastName, _ := vault.EncryptWithContext(ctx, lastName, "user:last_name")

		_, err := db.ExecContext(ctx,
			"UPDATE users SET email = $1, first_name = $2, last_name = $3 WHERE id = $4",
			encEmail, encFirstName, encLastName, id)
		if err != nil {
			return err
		}
		count++
	}

	log.Printf("Processed %d users", count)
	return nil
}

func encryptInvitationsTablePlaintext(ctx context.Context, db *sql.DB, vault *vaultClientEncrypt) error {
	rows, err := db.QueryContext(ctx, "SELECT id, email, token FROM invitations")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, email, token string
		if err := rows.Scan(&id, &email, &token); err != nil {
			return err
		}

		encEmail, _ := vault.EncryptWithContext(ctx, email, "invitation:email")
		encToken, _ := vault.EncryptWithContext(ctx, token, "invitation:token")

		_, err := db.ExecContext(ctx,
			"UPDATE invitations SET email = $1, token = $2 WHERE id = $3",
			encEmail, encToken, id)
		if err != nil {
			return err
		}
		count++
	}

	log.Printf("Processed %d invitations", count)
	return nil
}

func encryptGuestOrdersTablePlaintext(ctx context.Context, db *sql.DB, vault *vaultClientEncrypt) error {
	rows, err := db.QueryContext(ctx, "SELECT id, customer_name, customer_phone, COALESCE(customer_email, ''), COALESCE(ip_address, ''), COALESCE(user_agent, '') FROM guest_orders WHERE is_anonymized = false")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, name, phone, email, ip, ua string
		if err := rows.Scan(&id, &name, &phone, &email, &ip, &ua); err != nil {
			return err
		}

		encName, _ := vault.EncryptWithContext(ctx, name, "guest_order:customer_name")
		encPhone, _ := vault.EncryptWithContext(ctx, phone, "guest_order:customer_phone")
		encEmail, _ := vault.EncryptWithContext(ctx, email, "guest_order:customer_email")
		encIP, _ := vault.EncryptWithContext(ctx, ip, "guest_order:ip_address")
		encUA, _ := vault.EncryptWithContext(ctx, ua, "guest_order:user_agent")

		_, err := db.ExecContext(ctx,
			"UPDATE guest_orders SET customer_name = $1, customer_phone = $2, customer_email = NULLIF($3, ''), ip_address = NULLIF($4, ''), user_agent = NULLIF($5, '') WHERE id = $6",
			encName, encPhone, encEmail, encIP, encUA, id)
		if err != nil {
			return err
		}
		count++
	}

	log.Printf("Processed %d guest orders", count)
	return nil
}

func encryptNotificationsTablePlaintext(ctx context.Context, db *sql.DB, vault *vaultClientEncrypt) error {
	rows, err := db.QueryContext(ctx, "SELECT id, recipient, body FROM notifications")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, recipient, body string
		if err := rows.Scan(&id, &recipient, &body); err != nil {
			return err
		}

		encRecipient, _ := vault.EncryptWithContext(ctx, recipient, "notification:recipient")
		encBody, _ := vault.EncryptWithContext(ctx, body, "notification:body")

		_, err := db.ExecContext(ctx,
			"UPDATE notifications SET recipient = $1, body = $2 WHERE id = $3",
			encRecipient, encBody, id)
		if err != nil {
			return err
		}
		count++
	}

	log.Printf("Processed %d notifications", count)
	return nil
}
