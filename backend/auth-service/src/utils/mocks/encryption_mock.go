package mocks

import (
	"context"
	"fmt"
)

// MockEncryptor is a mock implementation of Encryptor for testing
type MockEncryptor struct {
	EncryptFunc      func(ctx context.Context, plaintext string) (string, error)
	DecryptFunc      func(ctx context.Context, ciphertext string) (string, error)
	EncryptBatchFunc func(ctx context.Context, plaintexts []string) ([]string, error)
	DecryptBatchFunc func(ctx context.Context, ciphertexts []string) ([]string, error)
}

// Encrypt calls the injected EncryptFunc
func (m *MockEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, plaintext)
	}
	return "encrypted:" + plaintext, nil
}

// Decrypt calls the injected DecryptFunc
func (m *MockEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, ciphertext)
	}
	// Strip "encrypted:" prefix for simple mock
	if len(ciphertext) > 10 && ciphertext[:10] == "encrypted:" {
		return ciphertext[10:], nil
	}
	return ciphertext, nil
}

// EncryptBatch calls the injected EncryptBatchFunc
func (m *MockEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if m.EncryptBatchFunc != nil {
		return m.EncryptBatchFunc(ctx, plaintexts)
	}
	encrypted := make([]string, len(plaintexts))
	for i, pt := range plaintexts {
		encrypted[i] = "encrypted:" + pt
	}
	return encrypted, nil
}

// DecryptBatch calls the injected DecryptBatchFunc
func (m *MockEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if m.DecryptBatchFunc != nil {
		return m.DecryptBatchFunc(ctx, ciphertexts)
	}
	decrypted := make([]string, len(ciphertexts))
	for i, ct := range ciphertexts {
		if len(ct) > 10 && ct[:10] == "encrypted:" {
			decrypted[i] = ct[10:]
		} else {
			decrypted[i] = ct
		}
	}
	return decrypted, nil
}

// NoOpEncryptor is a pass-through encryptor for testing (no encryption)
type NoOpEncryptor struct{}

// Encrypt returns plaintext unchanged
func (n *NoOpEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return plaintext, nil
}

// Decrypt returns ciphertext unchanged
func (n *NoOpEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return ciphertext, nil
}

// EncryptBatch returns plaintexts unchanged
func (n *NoOpEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	return plaintexts, nil
}

// DecryptBatch returns ciphertexts unchanged
func (n *NoOpEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	return ciphertexts, nil
}

// ErrorEncryptor always returns errors for testing error handling
type ErrorEncryptor struct {
	EncryptError      error
	DecryptError      error
	EncryptBatchError error
	DecryptBatchError error
}

// Encrypt returns the configured error
func (e *ErrorEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if e.EncryptError != nil {
		return "", e.EncryptError
	}
	return "", fmt.Errorf("encryption error")
}

// Decrypt returns the configured error
func (e *ErrorEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if e.DecryptError != nil {
		return "", e.DecryptError
	}
	return "", fmt.Errorf("decryption error")
}

// EncryptBatch returns the configured error
func (e *ErrorEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if e.EncryptBatchError != nil {
		return nil, e.EncryptBatchError
	}
	return nil, fmt.Errorf("batch encryption error")
}

// DecryptBatch returns the configured error
func (e *ErrorEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if e.DecryptBatchError != nil {
		return nil, e.DecryptBatchError
	}
	return nil, fmt.Errorf("batch decryption error")
}
