package config

import (
	"fmt"

	"github.com/pos/backend/product-service/src/utils"
)

// StorageConfig holds configuration for object storage (S3/MinIO)
type StorageConfig struct {
	// Connection settings
	Endpoint        string // S3 endpoint (e.g., "localhost:9000" for MinIO, "s3.amazonaws.com" for AWS)
	PublicEndpoint  string // Public S3 endpoint (e.g., "localhost:9000" for MinIO, "s3.amazonaws.com" for AWS)
	AccessKeyID     string // S3 access key
	SecretAccessKey string // S3 secret key
	BucketName      string // S3 bucket name
	Region          string // S3 region (e.g., "us-east-1")
	UseSSL          bool   // Whether to use HTTPS
	ForcePathStyle  bool   // Force path-style URLs (required for MinIO)

	// Storage limits
	MaxPhotoSizeBytes        int64 // Maximum photo file size in bytes (default: 10MB)
	MaxPhotosPerProduct      int   // Maximum photos per product (default: 5)
	DefaultStorageQuotaBytes int64 // Default storage quota per tenant (default: 5GB)
	PresignedURLTTLSeconds   int64 // TTL for presigned URLs (default: 7 days)
}

// LoadStorageConfig loads storage configuration from environment variables
func LoadStorageConfig() (*StorageConfig, error) {
	config := &StorageConfig{
		Endpoint:        utils.GetEnv("S3_ENDPOINT", "localhost:9000"),
		PublicEndpoint:  utils.GetEnv("S3_PUBLIC_ENDPOINT", "localhost:9000"),
		AccessKeyID:     utils.GetEnv("S3_ACCESS_KEY", "minioadmin"),
		SecretAccessKey: utils.GetEnv("S3_SECRET_KEY", "minioadmin"),
		BucketName:      utils.GetEnv("S3_BUCKET_NAME", "product-photos"),
		Region:          utils.GetEnv("S3_REGION", "us-east-1"),
		UseSSL:          utils.GetEnvBool("S3_USE_SSL", false),
		ForcePathStyle:  utils.GetEnvBool("S3_FORCE_PATH_STYLE", true),

		MaxPhotoSizeBytes:        utils.GetEnvInt64("MAX_PHOTO_SIZE_BYTES", 10485760),          // 10MB
		MaxPhotosPerProduct:      utils.GetEnvInt("MAX_PHOTOS_PER_PRODUCT", 5),                 // 5 photos
		DefaultStorageQuotaBytes: utils.GetEnvInt64("DEFAULT_STORAGE_QUOTA_BYTES", 5368709120), // 5GB
		PresignedURLTTLSeconds:   utils.GetEnvInt64("PRESIGNED_URL_TTL_SECONDS", 604800),       // 7 days
	}

	// Validate required fields
	if config.Endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT is required")
	}
	if config.PublicEndpoint == "" {
		return nil, fmt.Errorf("S3_PUBLIC_ENDPOINT is required")
	}
	if config.AccessKeyID == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY is required")
	}
	if config.SecretAccessKey == "" {
		return nil, fmt.Errorf("S3_SECRET_KEY is required")
	}
	if config.BucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME is required")
	}

	return config, nil
}
