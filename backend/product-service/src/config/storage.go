package config

import (
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
func LoadStorageConfig() *StorageConfig {
	config := &StorageConfig{
		Endpoint:        utils.GetEnv("S3_ENDPOINT"),
		PublicEndpoint:  utils.GetEnv("S3_PUBLIC_ENDPOINT"),
		AccessKeyID:     utils.GetEnv("S3_ACCESS_KEY"),
		SecretAccessKey: utils.GetEnv("S3_SECRET_KEY"),
		BucketName:      utils.GetEnv("S3_BUCKET_NAME"),
		Region:          utils.GetEnv("S3_REGION"),
		UseSSL:          utils.GetEnvBool("S3_USE_SSL"),
		ForcePathStyle:  utils.GetEnvBool("S3_FORCE_PATH_STYLE"),

		MaxPhotoSizeBytes:        utils.GetEnvInt64("MAX_PHOTO_SIZE_BYTES"),        // 10MB
		MaxPhotosPerProduct:      utils.GetEnvInt("MAX_PHOTOS_PER_PRODUCT"),        // 5 photos
		DefaultStorageQuotaBytes: utils.GetEnvInt64("DEFAULT_STORAGE_QUOTA_BYTES"), // 5GB
		PresignedURLTTLSeconds:   utils.GetEnvInt64("PRESIGNED_URL_TTL_SECONDS"),   // 7 days
	}

	return config
}
