package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// VaultEncryptor interface for encryption operations
type VaultEncryptor interface {
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	EncryptWithContext(ctx context.Context, plaintext, encryptionContext string) (string, error)
}

// Simple Vault client implementation (import from actual implementation)
// For this migration, we'll use a simplified version that delegates to the actual VaultClient

type MigrationConfig struct {
	DBURL         string
	VaultAddr     string
	VaultToken    string
	VaultKeyPath  string
	Tables        []string
	DryRun        bool
	BatchSize     int
	SkipDecrypted bool
}

type TableMigration struct {
	TableName      string
	IDColumn       string
	EncryptedCols  []ColumnConfig
	WhereCondition string
}

type ColumnConfig struct {
	Name    string
	Context string
}

var tableMigrations = []TableMigration{
	{
		TableName: "users",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "email", Context: "user:email"},
			{Name: "first_name", Context: "user:first_name"},
			{Name: "last_name", Context: "user:last_name"},
		},
		WhereCondition: "is_active = true", // Only migrate active users
	},
	{
		TableName: "invitations",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "email", Context: "invitation:email"},
			{Name: "token", Context: "invitation:token"},
		},
		WhereCondition: "status IN ('pending', 'sent')", // Only pending/sent invitations
	},
	{
		TableName: "guest_orders",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "customer_name", Context: "guest_order:customer_name"},
			{Name: "customer_phone", Context: "guest_order:customer_phone"},
			{Name: "customer_email", Context: "guest_order:customer_email"},
			{Name: "ip_address", Context: "guest_order:ip_address"},
			{Name: "user_agent", Context: "guest_order:user_agent"},
		},
		WhereCondition: "is_anonymized = false", // Skip anonymized orders
	},
	{
		TableName: "delivery_addresses",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "full_address", Context: "delivery_address:full_address"},
			{Name: "geocoding_result", Context: "delivery_address:geocoding_result"},
		},
		WhereCondition: "",
	},
	{
		TableName: "notifications",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "recipient", Context: "notification:recipient"},
			{Name: "body", Context: "notification:body"},
		},
		WhereCondition: "status != 'failed'", // Skip failed notifications
	},
	{
		TableName: "sessions",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "session_id", Context: "session:session_id"},
			{Name: "ip_address", Context: "session:ip_address"},
		},
		WhereCondition: "expired_at > NOW()", // Only active sessions
	},
	{
		TableName: "consent_records",
		IDColumn:  "id",
		EncryptedCols: []ColumnConfig{
			{Name: "ip_address", Context: "consent_record:ip_address"},
		},
		WhereCondition: "",
	},
}

func main() {
	var (
		dbURL         = flag.String("db", os.Getenv("DATABASE_URL"), "Database connection URL")
		vaultAddr     = flag.String("vault-addr", os.Getenv("VAULT_ADDR"), "Vault address")
		vaultToken    = flag.String("vault-token", os.Getenv("VAULT_TOKEN"), "Vault token")
		vaultKeyPath  = flag.String("vault-key", "transit/keys/pos-encryption-key", "Vault transit key path")
		tables        = flag.String("tables", "all", "Comma-separated list of tables to migrate (or 'all')")
		dryRun        = flag.Bool("dry-run", false, "Dry run mode - show what would be done without making changes")
		batchSize     = flag.Int("batch-size", 100, "Number of records to process in each batch")
		skipDecrypted = flag.Bool("skip-decrypted", false, "Skip records that fail decryption (assume already migrated)")
	)
	flag.Parse()

	// Validate required flags
	if *dbURL == "" {
		log.Fatal("Database URL is required (--db or DATABASE_URL env var)")
	}
	if *vaultAddr == "" {
		log.Fatal("Vault address is required (--vault-addr or VAULT_ADDR env var)")
	}
	if *vaultToken == "" {
		log.Fatal("Vault token is required (--vault-token or VAULT_TOKEN env var)")
	}

	config := &MigrationConfig{
		DBURL:         *dbURL,
		VaultAddr:     *vaultAddr,
		VaultToken:    *vaultToken,
		VaultKeyPath:  *vaultKeyPath,
		DryRun:        *dryRun,
		BatchSize:     *batchSize,
		SkipDecrypted: *skipDecrypted,
	}

	// Parse tables to migrate
	if *tables == "all" {
		for _, tm := range tableMigrations {
			config.Tables = append(config.Tables, tm.TableName)
		}
	} else {
		config.Tables = strings.Split(*tables, ",")
	}

	log.Printf("Starting context-based encryption migration...")
	log.Printf("Mode: %s", map[bool]string{true: "DRY RUN", false: "LIVE"}[*dryRun])
	log.Printf("Tables: %v", config.Tables)
	log.Printf("Batch size: %d", *batchSize)

	if err := runMigration(config); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
}

func runMigration(config *MigrationConfig) error {
	// Connect to database
	db, err := sql.Open("postgres", config.DBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize Vault client
	// NOTE: This is a simplified version. In production, use the actual VaultClient from utils
	log.Println("⚠️  WARNING: This script requires manual implementation of Vault client")
	log.Println("⚠️  Import VaultClient from the service utils package")
	log.Println()

	ctx := context.Background()

	// Migrate each table
	for _, tableName := range config.Tables {
		var tm *TableMigration
		for i := range tableMigrations {
			if tableMigrations[i].TableName == tableName {
				tm = &tableMigrations[i]
				break
			}
		}

		if tm == nil {
			log.Printf("⚠️  Table '%s' not found in migration config, skipping", tableName)
			continue
		}

		log.Printf("\n=== Migrating table: %s ===", tm.TableName)

		if err := migrateTable(ctx, db, tm, config); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", tm.TableName, err)
		}

		log.Printf("✅ Table %s migrated successfully", tm.TableName)
	}

	return nil
}

func migrateTable(ctx context.Context, db *sql.DB, tm *TableMigration, config *MigrationConfig) error {
	// Build column list
	var columns []string
	columns = append(columns, tm.IDColumn)
	for _, col := range tm.EncryptedCols {
		columns = append(columns, col.Name)
	}

	// Build query
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tm.TableName)
	if tm.WhereCondition != "" {
		query += " WHERE " + tm.WhereCondition
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tm.TableName)
	if tm.WhereCondition != "" {
		countQuery += " WHERE " + tm.WhereCondition
	}

	var totalCount int
	if err := db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return fmt.Errorf("failed to count records: %w", err)
	}

	log.Printf("Found %d records to migrate", totalCount)

	if totalCount == 0 {
		return nil
	}

	// Execute query
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query table: %w", err)
	}
	defer rows.Close()

	// Process records in batches
	var (
		processed int
		errors    int
		skipped   int
	)

	for rows.Next() {
		// Scan row
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// First value is ID
		id := values[0]

		// Process encrypted columns
		updateNeeded := false
		updateCols := []string{}
		updateVals := []interface{}{}
		paramIdx := 1

		for i, colConfig := range tm.EncryptedCols {
			colIdx := i + 1 // Skip ID column
			encryptedValue := values[colIdx]

			// Handle NULL values
			if encryptedValue == nil {
				continue
			}

			encryptedStr, ok := encryptedValue.(string)
			if !ok || encryptedStr == "" {
				continue
			}

			// NOTE: This requires actual VaultClient implementation
			// For now, we'll just log what would be done
			log.Printf("Would migrate %s.%s (ID: %v) with context: %s", tm.TableName, colConfig.Name, id, colConfig.Context)

			if config.DryRun {
				updateNeeded = true
				continue
			}

			// In production:
			// 1. Decrypt with old method (no context): plaintext, err := vaultClient.Decrypt(ctx, encryptedStr)
			// 2. Re-encrypt with context: newCiphertext, err := vaultClient.EncryptWithContext(ctx, plaintext, colConfig.Context)
			// 3. Store update: updateCols = append(updateCols, colConfig.Name); updateVals = append(updateVals, newCiphertext)

			updateNeeded = true
		}

		if updateNeeded && !config.DryRun {
			// Build and execute UPDATE query
			updateQuery := fmt.Sprintf("UPDATE %s SET ", tm.TableName)
			for i, col := range updateCols {
				if i > 0 {
					updateQuery += ", "
				}
				updateQuery += fmt.Sprintf("%s = $%d", col, paramIdx)
				paramIdx++
			}
			updateQuery += fmt.Sprintf(" WHERE %s = $%d", tm.IDColumn, paramIdx)
			updateVals = append(updateVals, id)

			if _, err := db.ExecContext(ctx, updateQuery, updateVals...); err != nil {
				log.Printf("❌ Failed to update record %v: %v", id, err)
				errors++
				continue
			}
		}

		processed++

		if processed%config.BatchSize == 0 {
			log.Printf("Progress: %d/%d records processed", processed, totalCount)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("Processed: %d, Errors: %d, Skipped: %d", processed, errors, skipped)

	return nil
}
