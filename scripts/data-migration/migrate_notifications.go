package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// migrateNotifications encrypts sensitive fields in notifications table metadata
func migrateNotifications(db *sql.DB, encryptor *VaultClient) error {
	ctx := context.Background()

	log.Println("Starting notifications table migration...")

	// Get all notifications with recipient, body, and metadata
	query := `
		SELECT id, recipient, body, metadata
		FROM notifications
		ORDER BY created_at ASC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var processedCount, encryptedCount, skippedCount int

	// Prepare update statement
	updateStmt, err := db.PrepareContext(ctx, `
		UPDATE notifications
		SET recipient = $1, body = $2, metadata = $3, updated_at = NOW()
		WHERE id = $4`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	for rows.Next() {
		var id string
		var recipient, body sql.NullString
		var metadataJSON []byte

		if err := rows.Scan(&id, &recipient, &body, &metadataJSON); err != nil {
			log.Printf("ERROR: Failed to scan notification %s: %v", id, err)
			continue
		}

		processedCount++

		// Check if recipient or body are already encrypted
		recipientAlreadyEncrypted := false
		bodyAlreadyEncrypted := false

		if recipient.Valid && len(recipient.String) > 8 && recipient.String[:8] == "vault:v1" {
			recipientAlreadyEncrypted = true
		}
		if body.Valid && len(body.String) > 8 && body.String[:8] == "vault:v1" {
			bodyAlreadyEncrypted = true
		}

		// Parse metadata
		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				log.Printf("ERROR: Failed to unmarshal metadata for notification %s: %v", id, err)
				continue
			}
		}

		// Check if metadata fields are already encrypted
		metadataAlreadyEncrypted := false
		sensitiveFields := []string{
			"email", "name", "inviter_name", "token", "invitation_token",
			"ip_address", "user_agent", "customer_name", "customer_email", "customer_phone",
		}

		for _, field := range sensitiveFields {
			if val, ok := metadata[field].(string); ok && len(val) > 8 {
				if val[:8] == "vault:v1" {
					metadataAlreadyEncrypted = true
					break
				}
			}
		}

		// Skip if everything is already encrypted
		if recipientAlreadyEncrypted && bodyAlreadyEncrypted && metadataAlreadyEncrypted {
			skippedCount++
			if processedCount%100 == 0 {
				log.Printf("Progress: %d processed, %d encrypted, %d skipped (already encrypted)", processedCount, encryptedCount, skippedCount)
			}
			continue
		}

		modified := false

		// Encrypt recipient if not already encrypted
		var encryptedRecipient string
		if recipient.Valid && !recipientAlreadyEncrypted && recipient.String != "" {
			encrypted, err := encryptor.Encrypt(ctx, recipient.String)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt recipient for notification %s: %v", id, err)
				encryptedRecipient = recipient.String // Keep original on error
			} else {
				encryptedRecipient = encrypted
				modified = true
			}
		} else if recipient.Valid {
			encryptedRecipient = recipient.String
		}

		// Encrypt body if not already encrypted
		var encryptedBody string
		if body.Valid && !bodyAlreadyEncrypted && body.String != "" {
			encrypted, err := encryptor.Encrypt(ctx, body.String)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt body for notification %s: %v", id, err)
				encryptedBody = body.String // Keep original on error
			} else {
				encryptedBody = encrypted
				modified = true
			}
		} else if body.Valid {
			encryptedBody = body.String
		}

		// Encrypt sensitive metadata fields if not already encrypted
		if !metadataAlreadyEncrypted && metadata != nil {
			for _, field := range sensitiveFields {
				if val, ok := metadata[field].(string); ok && val != "" {
					encrypted, err := encryptor.Encrypt(ctx, val)
					if err != nil {
						log.Printf("ERROR: Failed to encrypt %s for notification %s: %v", field, id, err)
						continue
					}
					metadata[field] = encrypted
					modified = true
				}
			}
		}

		if !modified {
			skippedCount++
			if processedCount%100 == 0 {
				log.Printf("Progress: %d processed, %d encrypted, %d skipped (no data to encrypt)", processedCount, encryptedCount, skippedCount)
			}
			continue
		}

		// Marshal metadata back to JSON
		var updatedMetadataJSON []byte
		if metadata != nil {
			updatedMetadataJSON, err = json.Marshal(metadata)
			if err != nil {
				log.Printf("ERROR: Failed to marshal metadata for notification %s: %v", id, err)
				continue
			}
		}

		// Update database
		if _, err := updateStmt.ExecContext(ctx, encryptedRecipient, encryptedBody, updatedMetadataJSON, id); err != nil {
			log.Printf("ERROR: Failed to update notification %s: %v", id, err)
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
		return fmt.Errorf("error iterating notifications: %w", err)
	}

	log.Printf("✓ Notifications migration completed: %d processed, %d encrypted, %d skipped", processedCount, encryptedCount, skippedCount)
	return nil
}
