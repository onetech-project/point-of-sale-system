package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// MigrateNotifications encrypts sensitive fields in notification metadata
func MigrateNotifications(config *Config) error {
	log.Println("=== Notification Metadata Encryption Migration ===")
	log.Println("Purpose: Encrypt sensitive PII fields within notification metadata")
	log.Println("Target: notifications table metadata column")
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
	if err := migrateNotifications(db, vaultClient); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
