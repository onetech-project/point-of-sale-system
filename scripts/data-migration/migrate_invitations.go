package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// migrateInvitations encrypts email and token fields in invitations table
func migrateInvitations(db *sql.DB, encryptor *VaultClient) error {
	ctx := context.Background()

	log.Println("Starting invitations table migration...")

	// Get all invitations
	query := `
		SELECT id, email, token
		FROM invitations
		ORDER BY created_at ASC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query invitations: %w", err)
	}
	defer rows.Close()

	var processedCount, encryptedCount, skippedCount int

	// Prepare update statement
	updateStmt, err := db.PrepareContext(ctx, `
		UPDATE invitations
		SET email = $1, token = $2, updated_at = NOW()
		WHERE id = $3`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	for rows.Next() {
		var id, email, token string

		if err := rows.Scan(&id, &email, &token); err != nil {
			log.Printf("ERROR: Failed to scan invitation %s: %v", id, err)
			continue
		}

		processedCount++

		// Check if already encrypted
		emailAlreadyEncrypted := len(email) > 8 && email[:8] == "vault:v1"
		tokenAlreadyEncrypted := len(token) > 8 && token[:8] == "vault:v1"

		if emailAlreadyEncrypted && tokenAlreadyEncrypted {
			skippedCount++
			if processedCount%100 == 0 {
				log.Printf("Progress: %d processed, %d encrypted, %d skipped (already encrypted)", processedCount, encryptedCount, skippedCount)
			}
			continue
		}

		// Encrypt email if not already encrypted
		var encryptedEmail string
		if !emailAlreadyEncrypted && email != "" {
			encrypted, err := encryptor.Encrypt(ctx, email)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt email for invitation %s: %v", id, err)
				encryptedEmail = email // Keep original on error
			} else {
				encryptedEmail = encrypted
			}
		} else {
			encryptedEmail = email
		}

		// Encrypt token if not already encrypted
		var encryptedToken string
		if !tokenAlreadyEncrypted && token != "" {
			encrypted, err := encryptor.Encrypt(ctx, token)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt token for invitation %s: %v", id, err)
				encryptedToken = token // Keep original on error
			} else {
				encryptedToken = encrypted
			}
		} else {
			encryptedToken = token
		}

		// Update database
		if _, err := updateStmt.ExecContext(ctx, encryptedEmail, encryptedToken, id); err != nil {
			log.Printf("ERROR: Failed to update invitation %s: %v", id, err)
			continue
		}

		encryptedCount++

		// Progress logging
		if processedCount%100 == 0 {
			log.Printf("Progress: %d processed, %d encrypted, %d skipped", processedCount, encryptedCount, skippedCount)
		}

		// Rate limiting to avoid overwhelming Vault
		if encryptedCount%50 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating invitations: %w", err)
	}

	log.Printf("✓ Invitations migration completed: %d processed, %d encrypted, %d skipped", processedCount, encryptedCount, skippedCount)
	return nil
}
