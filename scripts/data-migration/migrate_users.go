package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// MigrateUsers encrypts existing user PII data
func MigrateUsers(config *Config) error {
	log.Println("=== User PII Encryption Migration (T066) ===")
	log.Println("Purpose: Encrypt existing user email, first_name, last_name in-place")
	log.Println("Target: users table columns")
	log.Println()

	// Initialize database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test database connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	log.Println("✓ Database connection established")

	// Initialize Vault encryption client
	vaultClient, err := NewVaultClient(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Vault client: %w", err)
	}
	log.Println("✓ Vault client initialized")
	log.Println()

	// Run migration
	stats := &MigrationStats{StartTime: time.Now()}
	if err := migrateUsersData(ctx, db, vaultClient, stats); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	stats.EndTime = time.Now()
	printUserStats(stats)
	return nil
}

func migrateUsersData(ctx context.Context, db *sql.DB, vaultClient *VaultClient, stats *MigrationStats) error {
	const batchSize = 100

	query := `
		SELECT id, email, first_name, last_name 
		FROM users 
		WHERE email IS NOT NULL 
		  AND email NOT LIKE 'vault:v1:%'
		ORDER BY id
		LIMIT $1
	`

	updateQuery := `
		UPDATE users 
		SET email = $1, 
		    first_name = $2, 
		    last_name = $3,
		    updated_at = NOW()
		WHERE id = $4
	`

	for {
		rows, err := db.QueryContext(ctx, query, batchSize)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		var users []struct {
			ID        string
			Email     *string
			FirstName *string
			LastName  *string
		}

		for rows.Next() {
			var user struct {
				ID        string
				Email     *string
				FirstName *string
				LastName  *string
			}
			if err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName); err != nil {
				rows.Close()
				return fmt.Errorf("scan failed: %w", err)
			}
			users = append(users, user)
			stats.TotalRecords++
		}
		rows.Close()

		if len(users) == 0 {
			log.Println("No more unencrypted users found")
			break
		}

		log.Printf("Processing batch of %d users...", len(users))

		for _, user := range users {
			encryptedEmail, encryptedFirstName, encryptedLastName, err := encryptUserFields(ctx, vaultClient, user.Email, user.FirstName, user.LastName)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt user %s: %v", user.ID, err)
				stats.Errors++
				continue
			}

			_, err = db.ExecContext(ctx, updateQuery, encryptedEmail, encryptedFirstName, encryptedLastName, user.ID)
			if err != nil {
				log.Printf("ERROR: Failed to update user %s: %v", user.ID, err)
				stats.Errors++
				continue
			}

			stats.Encrypted++
			if stats.Encrypted%10 == 0 {
				log.Printf("Progress: %d/%d users encrypted", stats.Encrypted, stats.TotalRecords)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func encryptUserFields(ctx context.Context, vaultClient *VaultClient, email, firstName, lastName *string) (string, *string, *string, error) {
	var encryptedEmail string
	var encryptedFirstName, encryptedLastName *string

	if email != nil && *email != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *email)
		if err != nil {
			return "", nil, nil, fmt.Errorf("email encryption failed: %w", err)
		}
		encryptedEmail = encrypted
	}

	if firstName != nil && *firstName != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *firstName)
		if err != nil {
			return "", nil, nil, fmt.Errorf("first_name encryption failed: %w", err)
		}
		encryptedFirstName = &encrypted
	}

	if lastName != nil && *lastName != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *lastName)
		if err != nil {
			return "", nil, nil, fmt.Errorf("last_name encryption failed: %w", err)
		}
		encryptedLastName = &encrypted
	}

	return encryptedEmail, encryptedFirstName, encryptedLastName, nil
}

func printUserStats(stats *MigrationStats) {
	duration := stats.EndTime.Sub(stats.StartTime)

	log.Println()
	log.Println("=== Migration Complete ===")
	log.Printf("Total users processed: %d", stats.TotalRecords)
	log.Printf("Successfully encrypted: %d", stats.Encrypted)
	log.Printf("Already encrypted: %d", stats.AlreadyEncrypted)
	log.Printf("Errors: %d", stats.Errors)
	log.Printf("Duration: %s", duration.Round(time.Second))

	if stats.Errors > 0 {
		log.Println()
		log.Println("⚠️  WARNING: Some users failed to encrypt. Check logs for details.")
	} else {
		log.Println()
		log.Println("✓ All users successfully encrypted")
	}

	log.Println()
	log.Println("Run the following SQL to verify 100% encryption:")
	log.Println("  SELECT COUNT(*) FROM users WHERE email NOT LIKE 'vault:v1:%';")
	log.Println("  Expected result: 0")
}
