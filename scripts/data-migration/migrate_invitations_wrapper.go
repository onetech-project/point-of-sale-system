package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// MigrateInvitations encrypts email and token fields in invitations table
func MigrateInvitations() error {
	log.Println("=== Invitation Encryption Migration ===")
	log.Println("Purpose: Encrypt email and token fields in invitations table")
	log.Println("Target: invitations table (email, token columns)")
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
	if err := migrateInvitations(db, vaultClient); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
