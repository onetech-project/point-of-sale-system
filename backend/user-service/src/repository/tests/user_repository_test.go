package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pos/user-service/src/utils/mocks"
)

// Example test demonstrating dependency injection with MockEncryptor
func TestUserRepository_Create_WithMockEncryptor(t *testing.T) {
	// This test demonstrates how to inject a mock encryptor for testing
	// In real tests, you'd use a test database or mock the database as well

	// Setup mock encryptor
	mockEncryptor := &mocks.MockEncryptor{
		EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
			if plaintext == "" {
				return "", nil
			}
			return "encrypted:" + plaintext, nil
		},
		DecryptFunc: func(ctx context.Context, ciphertext string) (string, error) {
			if ciphertext == "" {
				return "", nil
			}
			// Remove "encrypted:" prefix
			if len(ciphertext) > 10 {
				return ciphertext[10:], nil
			}
			return ciphertext, nil
		},
	}

	// Note: In a real test, you'd also mock/setup a test database
	// This is just to show the encryptor injection pattern
	// db := setupTestDB(t) // your test DB setup
	// defer db.Close()

	// Create repository with injected mock encryptor
	// repo := repository.NewUserRepository(db, mockEncryptor)

	// Suppress unused variable warning
	_ = mockEncryptor

	t.Log("Example test showing MockEncryptor injection")
	t.Log("In production code, use repository.NewUserRepositoryWithVault(db)")
	t.Log("In test code, use repository.NewUserRepository(db, mockEncryptor)")
}

// Example test with NoOpEncryptor (no encryption for testing)
func TestUserRepository_Create_WithNoOpEncryptor(t *testing.T) {
	noOpEncryptor := &mocks.NoOpEncryptor{}

	// Create repository with no-op encryptor
	// repo := repository.NewUserRepository(db, noOpEncryptor)

	// Suppress unused variable warning
	_ = noOpEncryptor

	t.Log("Example test showing NoOpEncryptor for testing without encryption")
}

// Example showing how to test encryption errors
func TestUserRepository_Create_EncryptionError(t *testing.T) {
	mockEncryptor := &mocks.MockEncryptor{
		EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
			// Simulate encryption failure
			return "", sql.ErrConnDone // simulate an error
		},
	}

	// repo := repository.NewUserRepository(db, mockEncryptor)
	// user := &models.User{Email: "test@example.com"}
	// err := repo.Create(context.Background(), user)
	// if err == nil {
	//     t.Error("Expected error from encryption failure")
	// }

	// Suppress unused variable warning
	_ = mockEncryptor

	t.Log("Example test showing encryption error handling")
}
