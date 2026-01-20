package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// PopulateSearchHashes populates hash columns for existing encrypted data
func PopulateSearchHashes() error {
	log.Println("=== Search Hash Population Migration ===")
	log.Println("Purpose: Generate searchable hashes for encrypted fields")
	log.Println("Target: users.email_hash, invitations.email_hash, invitations.token_hash")
	log.Println()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
	if err := populateSearchHashes(db, vaultClient); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
