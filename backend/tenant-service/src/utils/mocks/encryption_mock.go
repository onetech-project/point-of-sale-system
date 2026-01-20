package mocks

import (
	"context"
	"fmt"
)

// MockEncryptor is a test implementation of the Encryptor interface
type MockEncryptor struct {
	EncryptFunc      func(ctx context.Context, plaintext string) (string, error)
	DecryptFunc      func(ctx context.Context, ciphertext string) (string, error)
	EncryptBatchFunc func(ctx context.Context, plaintexts []string) ([]string, error)
	DecryptBatchFunc func(ctx context.Context, ciphertexts []string) ([]string, error)
}

func (m *MockEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, plaintext)
	}
	return "encrypted_" + plaintext, nil
}

func (m *MockEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, ciphertext)
	}
	// Remove "encrypted_" prefix if present
	if len(ciphertext) > 10 && ciphertext[:10] == "encrypted_" {
		return ciphertext[10:], nil
	}
	return ciphertext, nil
}

func (m *MockEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if m.EncryptBatchFunc != nil {
		return m.EncryptBatchFunc(ctx, plaintexts)
	}
	result := make([]string, len(plaintexts))
	for i, pt := range plaintexts {
		result[i] = "encrypted_" + pt
	}
	return result, nil
}

func (m *MockEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if m.DecryptBatchFunc != nil {
		return m.DecryptBatchFunc(ctx, ciphertexts)
	}
	result := make([]string, len(ciphertexts))
	for i, ct := range ciphertexts {
		if len(ct) > 10 && ct[:10] == "encrypted_" {
			result[i] = ct[10:]
		} else {
			result[i] = ct
		}
	}
	return result, nil
}

// NoOpEncryptor is a pass-through encryptor for testing (no encryption)
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

// ErrorEncryptor always returns errors (for testing error handling)
type ErrorEncryptor struct{}

func (e *ErrorEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return "", fmt.Errorf("mock encryption error")
}

func (e *ErrorEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return "", fmt.Errorf("mock decryption error")
}

func (e *ErrorEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	return nil, fmt.Errorf("mock batch encryption error")
}

func (e *ErrorEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	return nil, fmt.Errorf("mock batch decryption error")
}
