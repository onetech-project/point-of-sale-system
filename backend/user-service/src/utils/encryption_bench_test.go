package utils

import (
	"context"
	"testing"
)

// BenchmarkEncryptSmall benchmarks encryption of small data (50 bytes)
func BenchmarkEncryptSmall(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()
	plaintext := "small_data_50_bytes_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // ~50 bytes

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Encrypt(ctx, plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptSmall benchmarks decryption of small data (50 bytes)
func BenchmarkDecryptSmall(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()
	plaintext := "small_data_50_bytes_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	// Pre-encrypt data
	ciphertext, err := encryptionService.Encrypt(ctx, plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Decrypt(ctx, ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncryptMedium benchmarks encryption of medium data (500 bytes)
func BenchmarkEncryptMedium(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Generate 500 bytes of data
	plaintext := ""
	for len(plaintext) < 500 {
		plaintext += "This is a medium-sized data block for encryption benchmarking. "
	}
	plaintext = plaintext[:500] // Trim to exactly 500 bytes

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Encrypt(ctx, plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptMedium benchmarks decryption of medium data (500 bytes)
func BenchmarkDecryptMedium(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Generate 500 bytes of data
	plaintext := ""
	for len(plaintext) < 500 {
		plaintext += "This is a medium-sized data block for encryption benchmarking. "
	}
	plaintext = plaintext[:500]

	// Pre-encrypt data
	ciphertext, err := encryptionService.Encrypt(ctx, plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Decrypt(ctx, ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncryptLarge benchmarks encryption of large data (5KB)
func BenchmarkEncryptLarge(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Generate 5KB of data
	plaintext := ""
	for len(plaintext) < 5120 {
		plaintext += "This is a large data block for encryption benchmarking. It simulates larger payloads like JSON documents or detailed records. "
	}
	plaintext = plaintext[:5120] // Trim to exactly 5KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Encrypt(ctx, plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptLarge benchmarks decryption of large data (5KB)
func BenchmarkDecryptLarge(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Generate 5KB of data
	plaintext := ""
	for len(plaintext) < 5120 {
		plaintext += "This is a large data block for encryption benchmarking. It simulates larger payloads like JSON documents or detailed records. "
	}
	plaintext = plaintext[:5120]

	// Pre-encrypt data
	ciphertext, err := encryptionService.Encrypt(ctx, plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.Decrypt(ctx, ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncryptBatch benchmarks batch encryption (10 items, 100 bytes each)
func BenchmarkEncryptBatch(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Create batch of 10 items
	plaintexts := make([]string, 10)
	for i := 0; i < 10; i++ {
		plaintexts[i] = "This is a test data item for batch encryption benchmarking that is about one hundred bytes long xxxx"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.EncryptBatch(ctx, plaintexts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptBatch benchmarks batch decryption (10 items)
func BenchmarkDecryptBatch(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()

	// Create batch of 10 items
	plaintexts := make([]string, 10)
	for i := 0; i < 10; i++ {
		plaintexts[i] = "This is a test data item for batch encryption benchmarking that is about one hundred bytes long xxxx"
	}

	// Pre-encrypt batch
	ciphertexts, err := encryptionService.EncryptBatch(ctx, plaintexts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptionService.DecryptBatch(ctx, ciphertexts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncryptParallel benchmarks concurrent encryption
func BenchmarkEncryptParallel(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()
	plaintext := "parallel_test_data_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := encryptionService.Encrypt(ctx, plaintext)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDecryptParallel benchmarks concurrent decryption
func BenchmarkDecryptParallel(b *testing.B) {
	encryptionService, err := NewVaultClient()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}
	ctx := context.Background()
	plaintext := "parallel_test_data_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	// Pre-encrypt data
	ciphertext, err := encryptionService.Encrypt(ctx, plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := encryptionService.Decrypt(ctx, ciphertext)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
