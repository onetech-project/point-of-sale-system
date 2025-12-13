package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pos/backend/product-service/src/config"
)

// StorageService handles object storage operations (S3/MinIO)
type StorageService struct {
	client *minio.Client
	config *config.StorageConfig
}

// NewStorageService creates a new StorageService with MinIO client
func NewStorageService(cfg *config.StorageConfig) (*StorageService, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &StorageService{
		client: client,
		config: cfg,
	}, nil
}

// InitializeBucket ensures the bucket exists, creates it if it doesn't
func (s *StorageService) InitializeBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.config.BucketName, minio.MakeBucketOptions{
			Region: s.config.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// UploadPhoto uploads a photo to object storage
func (s *StorageService) UploadPhoto(ctx context.Context, storageKey string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(
		ctx,
		s.config.BucketName,
		storageKey,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to upload photo to storage: %w", err)
	}

	return nil
}

// GetPhotoURL generates a presigned URL for photo access
func (s *StorageService) GetPhotoURL(ctx context.Context, storageKey string) (string, error) {
	ttl := time.Duration(s.config.PresignedURLTTLSeconds) * time.Second

	url, err := s.client.PresignedGetObject(ctx, s.config.BucketName, storageKey, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// DeletePhoto removes a photo from object storage
func (s *StorageService) DeletePhoto(ctx context.Context, storageKey string) error {
	err := s.client.RemoveObject(ctx, s.config.BucketName, storageKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete photo from storage: %w", err)
	}

	return nil
}

// GetPhoto retrieves a photo from object storage
func (s *StorageService) GetPhoto(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.config.BucketName, storageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get photo from storage: %w", err)
	}

	return object, nil
}

// GenerateStorageKey creates a unique storage key for a photo
// Format: photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.{ext}
func GenerateStorageKey(tenantID, productID, photoID uuid.UUID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg" // Default extension
	}

	timestamp := time.Now().Unix()
	return fmt.Sprintf("photos/%s/%s/%s_%d%s", tenantID, productID, photoID, timestamp, ext)
}

// SanitizeFilename removes potentially dangerous characters from filenames
func SanitizeFilename(filename string) string {
	// Remove path traversal attempts
	filename = filepath.Base(filename)

	// Replace potentially dangerous characters
	replacer := strings.NewReplacer(
		"..", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)

	return replacer.Replace(filename)
}

// HealthCheck verifies connectivity to object storage
func (s *StorageService) HealthCheck(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.config.BucketName)
	if err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	if !exists {
		return fmt.Errorf("bucket %s does not exist", s.config.BucketName)
	}

	return nil
}
