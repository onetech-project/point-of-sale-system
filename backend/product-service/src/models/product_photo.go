package models

import (
	"time"

	"github.com/google/uuid"
)

// ProductPhoto represents metadata for a product photo stored in object storage
type ProductPhoto struct {
	// Identity
	ID        uuid.UUID `json:"id" db:"id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`

	// Storage information
	StorageKey       string `json:"storage_key" db:"storage_key"`             // S3 object key: photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.ext
	OriginalFilename string `json:"original_filename" db:"original_filename"` // User's original filename (sanitized)
	FileSizeBytes    int    `json:"file_size_bytes" db:"file_size_bytes"`     // File size in bytes for quota tracking
	MimeType         string `json:"mime_type" db:"mime_type"`                 // image/jpeg, image/png, image/webp, image/gif

	// Image dimensions
	WidthPx  *int `json:"width_px,omitempty" db:"width_px"`   // Image width in pixels (NULL if not decoded)
	HeightPx *int `json:"height_px,omitempty" db:"height_px"` // Image height in pixels (NULL if not decoded)

	// Display configuration
	DisplayOrder int  `json:"display_order" db:"display_order"` // Order in carousel (0-based, unique per product)
	IsPrimary    bool `json:"is_primary" db:"is_primary"`       // Primary photo shown in listings (only one per product)

	// Audit
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Runtime field (not stored in database)
	PhotoURL string `json:"photo_url,omitempty" db:"-"` // Presigned URL for photo access
}

// Validate performs validation on ProductPhoto fields
func (p *ProductPhoto) Validate() error {
	// Validate required fields
	if p.ProductID == uuid.Nil {
		return ErrInvalidProductID
	}
	if p.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	if p.StorageKey == "" {
		return ErrInvalidStorageKey
	}
	if p.OriginalFilename == "" {
		return ErrInvalidFilename
	}
	if p.FileSizeBytes <= 0 || p.FileSizeBytes > 10485760 { // 10MB max
		return ErrInvalidFileSize
	}

	// Validate MIME type
	validMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	if !validMimeTypes[p.MimeType] {
		return ErrUnsupportedMimeType
	}

	// Validate dimensions if present
	if p.WidthPx != nil && (*p.WidthPx <= 0 || *p.WidthPx > 4096) {
		return ErrInvalidDimensions
	}
	if p.HeightPx != nil && (*p.HeightPx <= 0 || *p.HeightPx > 4096) {
		return ErrInvalidDimensions
	}

	// Validate display order
	if p.DisplayOrder < 0 {
		return ErrInvalidDisplayOrder
	}

	return nil
}

// ProductPhotoCreateRequest represents the request to upload a product photo
type ProductPhotoCreateRequest struct {
	ProductID    uuid.UUID `json:"product_id" form:"product_id"`
	DisplayOrder *int      `json:"display_order,omitempty" form:"display_order"`
	IsPrimary    *bool     `json:"is_primary,omitempty" form:"is_primary"`
}

// ProductPhotoUpdateRequest represents the request to update photo metadata
type ProductPhotoUpdateRequest struct {
	DisplayOrder *int  `json:"display_order,omitempty"`
	IsPrimary    *bool `json:"is_primary,omitempty"`
}

// ProductPhotoReorderRequest represents the request to reorder multiple photos
type ProductPhotoReorderRequest struct {
	PhotoOrders []PhotoOrder `json:"photo_orders"`
}

// PhotoOrder represents a single photo's new order position
type PhotoOrder struct {
	PhotoID      uuid.UUID `json:"photo_id"`
	DisplayOrder int       `json:"display_order"`
}

// StorageQuotaResponse represents storage usage information
type StorageQuotaResponse struct {
	TenantID          uuid.UUID `json:"tenant_id"`
	StorageUsedBytes  int64     `json:"storage_used_bytes"`
	StorageQuotaBytes int64     `json:"storage_quota_bytes"`
	AvailableBytes    int64     `json:"available_bytes"`
	UsagePercentage   float64   `json:"usage_percentage"`
	PhotoCount        int       `json:"photo_count"`
	ApproachingLimit  bool      `json:"approaching_limit"` // true if usage > 80%
	QuotaExceeded     bool      `json:"quota_exceeded"`    // true if usage >= quota
}

// Custom errors for ProductPhoto
var (
	ErrInvalidProductID    = &ValidationError{Field: "product_id", Message: "invalid product ID"}
	ErrInvalidTenantID     = &ValidationError{Field: "tenant_id", Message: "invalid tenant ID"}
	ErrInvalidStorageKey   = &ValidationError{Field: "storage_key", Message: "storage key cannot be empty"}
	ErrInvalidFilename     = &ValidationError{Field: "original_filename", Message: "filename cannot be empty"}
	ErrInvalidFileSize     = &ValidationError{Field: "file_size_bytes", Message: "file size must be between 1 byte and 10MB"}
	ErrUnsupportedMimeType = &ValidationError{Field: "mime_type", Message: "unsupported MIME type (allowed: jpeg, png, webp, gif)"}
	ErrInvalidDimensions   = &ValidationError{Field: "dimensions", Message: "dimensions must be between 1 and 4096 pixels"}
	ErrInvalidDisplayOrder = &ValidationError{Field: "display_order", Message: "display order must be non-negative"}
	ErrMaxPhotosReached    = &ValidationError{Field: "photos", Message: "product already has maximum number of photos (5)"}
	ErrQuotaExceeded       = &ValidationError{Field: "storage_quota", Message: "storage quota exceeded"}
	ErrPhotoNotFound       = &ValidationError{Field: "photo_id", Message: "photo not found"}
	ErrUnauthorizedAccess  = &ValidationError{Field: "tenant_id", Message: "unauthorized access to photo"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
