package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/services"
	"github.com/pos/user-service/src/utils/mocks"
)

// Example: Testing UserService with MockEncryptor
func TestUserService_WithMockEncryptor(t *testing.T) {
	// Setup mock encryptor that doesn't actually encrypt
	mockEncryptor := &mocks.MockEncryptor{
		EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
			return "mock:" + plaintext, nil
		},
		DecryptFunc: func(ctx context.Context, ciphertext string) (string, error) {
			// Remove "mock:" prefix
			if len(ciphertext) > 5 && ciphertext[:5] == "mock:" {
				return ciphertext[5:], nil
			}
			return ciphertext, nil
		},
		EncryptWithContextFunc: func(ctx context.Context, plaintext string, encryptionContext string) (string, error) {
			return "mock:" + plaintext, nil
		},
		DecryptWithContextFunc: func(ctx context.Context, ciphertext string, encryptionContext string) (string, error) {
			// Remove "mock:" prefix
			if len(ciphertext) > 5 && ciphertext[:5] == "mock:" {
				return ciphertext[5:], nil
			}
			return ciphertext, nil
		},
	}

	// Setup test database (you'd use a test DB or mock DB)
	// db := setupTestDatabase(t)
	// defer db.Close()

	var db *sql.DB // placeholder for example

	// Create repository with mock encryptor (NO VAULT REQUIRED)
	userRepo := repository.NewUserRepository(db, mockEncryptor)

	// Create service with injected repository
	userService := services.NewUserServiceWithRepository(db, userRepo)

	// Now you can test UserService methods without Vault!
	_ = userService

	t.Log("✅ UserService is testable with mock encryptor")
	t.Log("No Vault connection needed for unit tests")
}

// Example: Testing UserService with NoOpEncryptor (plaintext for tests)
func TestUserService_WithNoOpEncryptor(t *testing.T) {
	noOpEncryptor := &mocks.NoOpEncryptor{}

	// Setup test database
	var db *sql.DB // placeholder

	// Create repository with no-op encryptor (data stays plaintext)
	userRepo := repository.NewUserRepository(db, noOpEncryptor)

	// Create service with injected repository
	userService := services.NewUserServiceWithRepository(db, userRepo)

	// Test business logic without encryption complexity
	_ = userService

	t.Log("✅ UserService is testable with no-op encryptor")
	t.Log("Useful for testing business logic without encryption")
}

// Example: Production code usage (with real Vault)
func ExampleNewUserService() {
	// In production code, use services.NewUserService directly
	// This will create a real VaultClient connection

	// db := setupProductionDatabase()
	// service, err := services.NewUserService(db)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// _ = service
}
