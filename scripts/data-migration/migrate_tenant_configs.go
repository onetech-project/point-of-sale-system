package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// MigrateTenantConfigs encrypts existing tenant payment credentials
func MigrateTenantConfigs(config *Config) error {
	log.Println("=== Tenant Config Payment Credentials Encryption Migration (T068) ===")
	log.Println("Purpose: Encrypt existing midtrans_server_key and midtrans_client_key in-place")
	log.Println("Target: tenant_configs table columns")
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
	if err := migrateTenantConfigsData(ctx, db, vaultClient, stats); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	stats.EndTime = time.Now()
	printTenantConfigStats(stats)
	return nil
}

func migrateTenantConfigsData(ctx context.Context, db *sql.DB, vaultClient *VaultClient, stats *MigrationStats) error {
	const batchSize = 100

	query := `
		SELECT id, tenant_id, midtrans_server_key, midtrans_client_key 
		FROM tenant_configs 
		WHERE (midtrans_server_key IS NOT NULL AND midtrans_server_key != '' AND midtrans_server_key NOT LIKE 'vault:v1:%')
		   OR (midtrans_client_key IS NOT NULL AND midtrans_client_key != '' AND midtrans_client_key NOT LIKE 'vault:v1:%')
		ORDER BY tenant_id
		LIMIT $1
	`

	updateQuery := `
		UPDATE tenant_configs 
		SET midtrans_server_key = $1, 
		    midtrans_client_key = $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	for {
		rows, err := db.QueryContext(ctx, query, batchSize)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		var configs []struct {
			ID                string
			TenantID          string
			MidtransServerKey *string
			MidtransClientKey *string
		}

		for rows.Next() {
			var config struct {
				ID                string
				TenantID          string
				MidtransServerKey *string
				MidtransClientKey *string
			}
			if err := rows.Scan(&config.ID, &config.TenantID, &config.MidtransServerKey, &config.MidtransClientKey); err != nil {
				rows.Close()
				return fmt.Errorf("scan failed: %w", err)
			}
			configs = append(configs, config)
			stats.TotalRecords++
		}
		rows.Close()

		if len(configs) == 0 {
			log.Println("No more unencrypted tenant configs found")
			break
		}

		log.Printf("Processing batch of %d tenant configs...", len(configs))

		for _, config := range configs {
			encryptedServerKey, encryptedClientKey, err := encryptPaymentCredentials(ctx, vaultClient, config.MidtransServerKey, config.MidtransClientKey)
			if err != nil {
				log.Printf("ERROR: Failed to encrypt tenant config %s: %v", config.TenantID, err)
				stats.Errors++
				continue
			}

			_, err = db.ExecContext(ctx, updateQuery, encryptedServerKey, encryptedClientKey, config.ID)
			if err != nil {
				log.Printf("ERROR: Failed to update tenant config %s: %v", config.TenantID, err)
				stats.Errors++
				continue
			}

			stats.Encrypted++
			if stats.Encrypted%10 == 0 {
				log.Printf("Progress: %d/%d tenant configs encrypted", stats.Encrypted, stats.TotalRecords)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func encryptPaymentCredentials(ctx context.Context, vaultClient *VaultClient, serverKey, clientKey *string) (*string, *string, error) {
	var encryptedServerKey, encryptedClientKey *string

	if serverKey != nil && *serverKey != "" && !isAlreadyEncrypted(*serverKey) {
		encrypted, err := vaultClient.Encrypt(ctx, *serverKey)
		if err != nil {
			return nil, nil, fmt.Errorf("midtrans_server_key encryption failed: %w", err)
		}
		encryptedServerKey = &encrypted
	} else {
		encryptedServerKey = serverKey
	}

	if clientKey != nil && *clientKey != "" && !isAlreadyEncrypted(*clientKey) {
		encrypted, err := vaultClient.Encrypt(ctx, *clientKey)
		if err != nil {
			return nil, nil, fmt.Errorf("midtrans_client_key encryption failed: %w", err)
		}
		encryptedClientKey = &encrypted
	} else {
		encryptedClientKey = clientKey
	}

	return encryptedServerKey, encryptedClientKey, nil
}

func isAlreadyEncrypted(value string) bool {
	return len(value) > 9 && value[:9] == "vault:v1:"
}

func printTenantConfigStats(stats *MigrationStats) {
	duration := stats.EndTime.Sub(stats.StartTime)

	log.Println()
	log.Println("=== Migration Complete ===")
	log.Printf("Total tenant configs processed: %d", stats.TotalRecords)
	log.Printf("Successfully encrypted: %d", stats.Encrypted)
	log.Printf("Already encrypted: %d", stats.AlreadyEncrypted)
	log.Printf("Errors: %d", stats.Errors)
	log.Printf("Duration: %s", duration.Round(time.Second))

	if stats.Errors > 0 {
		log.Println()
		log.Println("⚠️  WARNING: Some tenant configs failed to encrypt. Check logs for details.")
	} else {
		log.Println()
		log.Println("✓ All tenant configs successfully encrypted")
	}

	log.Println()
	log.Println("Run the following SQL to verify 100% encryption:")
	log.Println("  SELECT COUNT(*) FROM tenant_configs WHERE")
	log.Println("    (midtrans_server_key IS NOT NULL AND midtrans_server_key != '' AND midtrans_server_key NOT LIKE 'vault:v1:%')")
	log.Println("    OR (midtrans_client_key IS NOT NULL AND midtrans_client_key != '' AND midtrans_client_key NOT LIKE 'vault:v1:%');")
	log.Println("  Expected result: 0")
}
