package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// MigrateGuestOrders encrypts existing guest order PII data
func MigrateGuestOrders(config *Config) error {
	log.Println("=== Guest Order PII Encryption Migration (T067) ===")
	log.Println("Purpose: Encrypt existing guest order customer_name, customer_phone, customer_email, ip_address in-place")
	log.Println("Target: guest_orders table columns")
	log.Println()

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	log.Println("✓ Database connection established")

	vaultClient, err := NewVaultClient(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Vault client: %w", err)
	}
	log.Println("✓ Vault client initialized")
	log.Println()

	stats := &MigrationStats{StartTime: time.Now()}
	if err := migrateGuestOrdersData(ctx, db, vaultClient, stats); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	stats.EndTime = time.Now()
	printGuestOrderStats(stats)
	return nil
}

func migrateGuestOrdersData(ctx context.Context, db *sql.DB, vaultClient *VaultClient, stats *MigrationStats) error {
	const batchSize = 100

	query := `
		SELECT id, customer_name, customer_phone, customer_email, ip_address 
		FROM guest_orders 
		WHERE is_anonymized = FALSE
		  AND customer_name IS NOT NULL 
		  AND customer_name NOT LIKE 'vault:v1:%'
		ORDER BY id
		LIMIT $1
	`

	updateQuery := `
		UPDATE guest_orders 
		SET customer_name = $1, 
		    customer_phone = $2, 
		    customer_email = $3,
		    ip_address = $4
		WHERE id = $5
	`

	for {
		rows, err := db.QueryContext(ctx, query, batchSize)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		var orders []struct {
			ID            string
			CustomerName  *string
			CustomerPhone *string
			CustomerEmail *string
			IPAddress     *string
		}

		for rows.Next() {
			var order struct {
				ID            string
				CustomerName  *string
				CustomerPhone *string
				CustomerEmail *string
				IPAddress     *string
			}
			if err := rows.Scan(&order.ID, &order.CustomerName, &order.CustomerPhone, &order.CustomerEmail, &order.IPAddress); err != nil {
				rows.Close()
				return fmt.Errorf("scan failed: %w", err)
			}
			orders = append(orders, order)
			stats.TotalRecords++
		}
		rows.Close()

		if len(orders) == 0 {
			log.Println("No more unencrypted guest orders found")
			break
		}

		log.Printf("Processing batch of %d guest orders...", len(orders))

		for _, order := range orders {
			encryptedFields, err := encryptGuestOrderFields(ctx, vaultClient, order.CustomerName, order.CustomerPhone, order.CustomerEmail, order.IPAddress)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt guest order %s: %v", order.ID, err)
				stats.Errors++
				continue
			}

			_, err = db.ExecContext(ctx, updateQuery,
				encryptedFields.CustomerName,
				encryptedFields.CustomerPhone,
				encryptedFields.CustomerEmail,
				encryptedFields.IPAddress,
				order.ID)
			if err != nil {
				log.Printf("ERROR: Failed to update guest order %s: %v", order.ID, err)
				stats.Errors++
				continue
			}

			stats.Encrypted++
			if stats.Encrypted%10 == 0 {
				log.Printf("Progress: %d/%d guest orders encrypted", stats.Encrypted, stats.TotalRecords)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

type encryptedGuestOrderFields struct {
	CustomerName  *string
	CustomerPhone *string
	CustomerEmail *string
	IPAddress     *string
}

func encryptGuestOrderFields(ctx context.Context, vaultClient *VaultClient, customerName, customerPhone, customerEmail, ipAddress *string) (*encryptedGuestOrderFields, error) {
	result := &encryptedGuestOrderFields{}

	if customerName != nil && *customerName != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *customerName)
		if err != nil {
			return nil, fmt.Errorf("customer_name encryption failed: %w", err)
		}
		result.CustomerName = &encrypted
	}

	if customerPhone != nil && *customerPhone != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *customerPhone)
		if err != nil {
			return nil, fmt.Errorf("customer_phone encryption failed: %w", err)
		}
		result.CustomerPhone = &encrypted
	}

	if customerEmail != nil && *customerEmail != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *customerEmail)
		if err != nil {
			return nil, fmt.Errorf("customer_email encryption failed: %w", err)
		}
		result.CustomerEmail = &encrypted
	}

	if ipAddress != nil && *ipAddress != "" {
		encrypted, err := vaultClient.Encrypt(ctx, *ipAddress)
		if err != nil {
			return nil, fmt.Errorf("ip_address encryption failed: %w", err)
		}
		result.IPAddress = &encrypted
	}

	return result, nil
}

func printGuestOrderStats(stats *MigrationStats) {
	duration := stats.EndTime.Sub(stats.StartTime)

	log.Println()
	log.Println("=== Migration Complete ===")
	log.Printf("Total guest orders processed: %d", stats.TotalRecords)
	log.Printf("Successfully encrypted: %d", stats.Encrypted)
	log.Printf("Already encrypted: %d", stats.AlreadyEncrypted)
	log.Printf("Errors: %d", stats.Errors)
	log.Printf("Duration: %s", duration.Round(time.Second))

	if stats.Errors > 0 {
		log.Println()
		log.Println("⚠️  WARNING: Some guest orders failed to encrypt. Check logs for details.")
	} else {
		log.Println()
		log.Println("✓ All guest orders successfully encrypted")
	}

	log.Println()
	log.Println("Run the following SQL to verify 100% encryption:")
	log.Println("  SELECT COUNT(*) FROM guest_orders WHERE is_anonymized = FALSE AND customer_name NOT LIKE 'vault:v1:%';")
	log.Println("  Expected result: 0")
}
