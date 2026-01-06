package mocks

import (
	"context"
	"fmt"
)

// MockEncryptor is a mock implementation of the Encryptor interface for testing
// It simulates encryption by adding a prefix/suffix instead of actual encryption
type MockEncryptor struct {
	EncryptFunc            func(ctx context.Context, plaintext string) (string, error)
	DecryptFunc            func(ctx context.Context, ciphertext string) (string, error)
	EncryptBatchFunc       func(ctx context.Context, plaintexts []string) ([]string, error)
	DecryptBatchFunc       func(ctx context.Context, ciphertexts []string) ([]string, error)
	EncryptWithContextFunc func(ctx context.Context, plaintext string, encryptionContext string) (string, error)
	DecryptWithContextFunc func(ctx context.Context, ciphertext string, encryptionContext string) (string, error)
}

// Encrypt mock implementation
func (m *MockEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, plaintext)
	}
	// Default behavior: add prefix to simulate encryption
	if plaintext == "" {
		return "", nil
	}
	return fmt.Sprintf("mock:encrypted:%s", plaintext), nil
}

// Decrypt mock implementation
func (m *MockEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, ciphertext)
	}
	// Default behavior: remove prefix to simulate decryption
	if ciphertext == "" {
		return "", nil
	}
	var plaintext string
	fmt.Sscanf(ciphertext, "mock:encrypted:%s", &plaintext)
	return plaintext, nil
}

// EncryptBatch mock implementation
func (m *MockEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if m.EncryptBatchFunc != nil {
		return m.EncryptBatchFunc(ctx, plaintexts)
	}
	// Default behavior: encrypt each item
	result := make([]string, len(plaintexts))
	for i, pt := range plaintexts {
		encrypted, err := m.Encrypt(ctx, pt)
		if err != nil {
			return nil, err
		}
		result[i] = encrypted
	}
	return result, nil
}

// DecryptBatch mock implementation
func (m *MockEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if m.DecryptBatchFunc != nil {
		return m.DecryptBatchFunc(ctx, ciphertexts)
	}
	// Default behavior: decrypt each item
	result := make([]string, len(ciphertexts))
	for i, ct := range ciphertexts {
		decrypted, err := m.Decrypt(ctx, ct)
		if err != nil {
			return nil, err
		}
		result[i] = decrypted
	}
	return result, nil
}

// Encrypt With Context mock implementation
func (m *MockEncryptor) EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error) {
	return m.Encrypt(ctx, plaintext)
}

// Decrypt With Context mock implementation
func (m *MockEncryptor) DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error) {
	return m.Decrypt(ctx, ciphertext)
}

// NoOpEncryptor is a pass-through encryptor for testing without encryption
type NoOpEncryptor struct{}

func (n *NoOpEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return plaintext, nil
}

func (n *NoOpEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return ciphertext, nil
}

func (n *NoOpEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	return plaintexts, nil
}

func (n *NoOpEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	return ciphertexts, nil
}

func (n *NoOpEncryptor) EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error) {
	return plaintext, nil
}

func (n *NoOpEncryptor) DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error) {
	return ciphertext, nil
}
