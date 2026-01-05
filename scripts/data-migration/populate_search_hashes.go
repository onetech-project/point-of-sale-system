package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// hashForSearch creates HMAC-SHA256 hash for searching
func hashForSearch(value string) string {
	secretKey := os.Getenv("SEARCH_HASH_SECRET")
	if secretKey == "" {
		panic("SEARCH_HASH_SECRET environment variable is not set")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

// populateSearchHashes populates hash columns for existing encrypted data
func populateSearchHashes(db *sql.DB, encryptor *VaultClient) error {
	ctx := context.Background()

	log.Println("Starting search hash population...")

	// 1. Populate users.email_hash
	log.Println("Populating users.email_hash...")
	if err := populateUsersEmailHash(ctx, db, encryptor); err != nil {
		return fmt.Errorf("failed to populate users.email_hash: %w", err)
	}

	// 2. Populate invitations.email_hash and token_hash
	log.Println("Populating invitations.email_hash and token_hash...")
	if err := populateInvitationsHashes(ctx, db, encryptor); err != nil {
		return fmt.Errorf("failed to populate invitations hashes: %w", err)
	}

	log.Println("✓ Search hash population completed")
	return nil
}

func populateUsersEmailHash(ctx context.Context, db *sql.DB, encryptor *VaultClient) error {
	query := `SELECT id, email FROM users WHERE email_hash IS NULL OR email_hash = ''`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	updateStmt, err := db.PrepareContext(ctx, `UPDATE users SET email_hash = $1 WHERE id = $2`)
	if err != nil {
		return err
	}
	defer updateStmt.Close()

	var processed, updated, skipped int
	for rows.Next() {
		var id, encryptedEmail string
		if err := rows.Scan(&id, &encryptedEmail); err != nil {
			log.Printf("ERROR: Failed to scan user %s: %v", id, err)
			continue
		}

		processed++

		// Decrypt email
		email, err := encryptor.Decrypt(ctx, encryptedEmail)
		if err != nil {
			log.Printf("ERROR: Failed to decrypt email for user %s: %v", id, err)
			skipped++
			continue
		}

		// Generate hash
		emailHash := hashForSearch(email)

		// Update hash
		if _, err := updateStmt.ExecContext(ctx, emailHash, id); err != nil {
			log.Printf("ERROR: Failed to update user %s: %v", id, err)
			skipped++
			continue
		}

		updated++
		if processed%100 == 0 {
			log.Printf("Progress: %d processed, %d updated, %d skipped", processed, updated, skipped)
		}

		if updated%50 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("✓ Users email_hash: %d processed, %d updated, %d skipped", processed, updated, skipped)
	return rows.Err()
}

func populateInvitationsHashes(ctx context.Context, db *sql.DB, encryptor *VaultClient) error {
	query := `SELECT id, email, token FROM invitations WHERE email_hash IS NULL OR email_hash = '' OR token_hash IS NULL OR token_hash = ''`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	updateStmt, err := db.PrepareContext(ctx, `UPDATE invitations SET email_hash = $1, token_hash = $2 WHERE id = $3`)
	if err != nil {
		return err
	}
	defer updateStmt.Close()

	var processed, updated, skipped int
	for rows.Next() {
		var id, encryptedEmail, encryptedToken string
		if err := rows.Scan(&id, &encryptedEmail, &encryptedToken); err != nil {
			log.Printf("ERROR: Failed to scan invitation %s: %v", id, err)
			continue
		}

		processed++

		// Decrypt email and token
		email, err := encryptor.Decrypt(ctx, encryptedEmail)
		if err != nil {
			log.Printf("ERROR: Failed to decrypt email for invitation %s: %v", id, err)
			skipped++
			continue
		}

		token, err := encryptor.Decrypt(ctx, encryptedToken)
		if err != nil {
			log.Printf("ERROR: Failed to decrypt token for invitation %s: %v", id, err)
			skipped++
			continue
		}

		// Generate hashes
		emailHash := hashForSearch(email)
		tokenHash := hashForSearch(token)

		// Update hashes
		if _, err := updateStmt.ExecContext(ctx, emailHash, tokenHash, id); err != nil {
			log.Printf("ERROR: Failed to update invitation %s: %v", id, err)
			skipped++
			continue
		}

		updated++
		if processed%100 == 0 {
			log.Printf("Progress: %d processed, %d updated, %d skipped", processed, updated, skipped)
		}

		if updated%50 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("✓ Invitations hashes: %d processed, %d updated, %d skipped", processed, updated, skipped)
	return rows.Err()
}
