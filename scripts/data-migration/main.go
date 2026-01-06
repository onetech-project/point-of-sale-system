package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// MigrationStats tracks progress and results
type MigrationStats struct {
	TotalRecords     int
	Encrypted        int
	AlreadyEncrypted int
	Errors           int
	StartTime        time.Time
	EndTime          time.Time
}

func main() {
	// Define command-line flags
	migrationType := flag.String("type", "", "Migration type: users, guest-orders, tenant-configs, notifications, or all")
	flag.Parse()

	if *migrationType == "" {
		fmt.Println("Usage: go run main.go -type=<migration-type>")
		fmt.Println()
		fmt.Println("Available migration types:")
		fmt.Println("  users              - Encrypt user PII (email, first_name, last_name)")
		fmt.Println("  guest-orders       - Encrypt guest order PII (customer_name, phone, email, ip_address)")
		fmt.Println("  tenant-configs     - Encrypt tenant payment credentials (midtrans keys)")
		fmt.Println("  notifications      - Encrypt notification recipient, body, and metadata sensitive fields")
		fmt.Println("  invitations        - Encrypt invitation email and token")
		fmt.Println("  search-hashes      - Populate searchable HMAC hashes for encrypted fields")
		fmt.Println("  encrypt-plaintext  - Encrypt plaintext PII data with context-based encryption")
		fmt.Println("  all                - Run all migrations sequentially")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  go run main.go -type=users")
		fmt.Println("  go run main.go -type=encrypt-plaintext")
		fmt.Println("  go run main.go -type=all")
		os.Exit(1)
	}

	// Load configuration from environment variables
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Run the specified migration(s)
	var migrationErr error
	switch *migrationType {
	case "users":
		migrationErr = MigrateUsers(config)
	case "guest-orders":
		migrationErr = MigrateGuestOrders(config)
	case "tenant-configs":
		migrationErr = MigrateTenantConfigs(config)
	case "notifications":
		migrationErr = MigrateNotifications(config)
	case "invitations":
		migrationErr = MigrateInvitations()
	case "search-hashes":
		migrationErr = PopulateSearchHashes()
	case "encrypt-plaintext":
		migrationErr = EncryptPlaintextDataWrapper(config)
	case "all":
		log.Println("Running all migrations sequentially...")
		log.Println()

		if err := MigrateUsers(config); err != nil {
			log.Printf("Users migration failed: %v", err)
			migrationErr = err
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := MigrateGuestOrders(config); err != nil {
			log.Printf("Guest orders migration failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := MigrateTenantConfigs(config); err != nil {
			log.Printf("Tenant configs migration failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := MigrateNotifications(config); err != nil {
			log.Printf("Notifications migration failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := MigrateInvitations(); err != nil {
			log.Printf("Invitations migration failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := PopulateSearchHashes(); err != nil {
			log.Printf("Search hash population failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		log.Println()
		log.Println("---")
		log.Println()

		if err := EncryptPlaintextDataWrapper(config); err != nil {
			log.Printf("Plaintext encryption failed: %v", err)
			if migrationErr == nil {
				migrationErr = err
			}
		}

		if migrationErr != nil {
			log.Println()
			log.Println("⚠️  One or more migrations encountered errors. Review the logs above.")
		} else {
			log.Println()
			log.Println("✓ All migrations completed successfully!")
		}
	default:
		log.Fatalf("Unknown migration type: %s. Use 'users', 'guest-orders', 'tenant-configs', 'notifications', 'invitations', 'search-hashes', 'encrypt-plaintext', or 'all'", *migrationType)
	}

	if migrationErr != nil {
		log.Fatalf("Migration failed: %v", migrationErr)
	}
}
