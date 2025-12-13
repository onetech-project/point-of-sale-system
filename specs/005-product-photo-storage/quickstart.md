# Quickstart Guide: Product Photo Storage

**Feature**: 005-product-photo-storage  
**For**: Developers implementing this feature  
**Date**: December 12, 2025

## Overview

This guide helps developers implement object storage-based product photo management in the product-service. You'll set up MinIO, implement photo upload/retrieval endpoints, and test the complete workflow.

**Estimated Time**: 4-6 hours for core implementation

---

## Prerequisites

- [ ] Go 1.23+ installed
- [ ] Docker and Docker Compose installed
- [ ] Access to project repository
- [ ] Familiarity with Echo web framework
- [ ] Understanding of PostgreSQL and migrations
- [ ] Basic knowledge of S3-compatible storage APIs

---

## Phase 1: Environment Setup (30 minutes)

### Step 1: Add MinIO to Docker Compose

Edit `docker-compose.yml` and add MinIO service:

```yaml
  minio:
    image: minio/minio:latest
    container_name: pos-minio
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"     # S3 API
      - "9001:9001"     # Web Console
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

volumes:
  minio_data:  # Add to existing volumes section
```

### Step 2: Update Product Service Configuration

Edit `backend/product-service/.env`:

```env
# Object Storage Configuration
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=pos-product-photos
S3_USE_SSL=false
S3_REGION=us-east-1

# Storage Quota
DEFAULT_STORAGE_QUOTA_BYTES=5368709120  # 5GB

# Photo Settings
MAX_PHOTO_SIZE_BYTES=10485760           # 10MB
MAX_PHOTO_DIMENSION_PX=4096
MAX_PHOTOS_PER_PRODUCT=5
PHOTO_RESIZE_THRESHOLD_PX=2048
PHOTO_URL_CACHE_TTL_SECONDS=518400      # 6 days
```

### Step 3: Start Services

```bash
cd /home/asrock/code/POS/point-of-sale-system
docker-compose up -d postgres redis minio
docker-compose logs -f minio  # Wait for "API: http://..."
```

### Step 4: Initialize MinIO Bucket

Access MinIO console at http://localhost:9001 (minioadmin/minioadmin):
1. Click "Buckets" → "Create Bucket"
2. Name: `pos-product-photos`
3. Click "Create"
4. Optional: Set bucket to public read if needed for guest ordering

Or use MinIO CLI:

```bash
docker exec -it pos-minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker exec -it pos-minio mc mb local/pos-product-photos
docker exec -it pos-minio mc ls local
```

---

## Phase 2: Database Schema (30 minutes)

### Step 1: Create Migration File

Create `backend/migrations/007_add_product_photos.up.sql`:

```sql
-- Add storage tracking to tenants table
ALTER TABLE tenants 
    ADD COLUMN IF NOT EXISTS storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS storage_quota_bytes BIGINT NOT NULL DEFAULT 5368709120;

ALTER TABLE tenants 
    ADD CONSTRAINT chk_storage_non_negative CHECK (storage_used_bytes >= 0);

CREATE INDEX IF NOT EXISTS idx_tenants_storage ON tenants(id, storage_used_bytes);

-- Create product_photos table
CREATE TABLE IF NOT EXISTS product_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    storage_key TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    
    width_px INTEGER,
    height_px INTEGER,
    
    display_order INTEGER NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_product FOREIGN KEY (product_id) 
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT chk_file_size CHECK (file_size_bytes > 0 AND file_size_bytes <= 10485760),
    CONSTRAINT chk_dimensions CHECK (
        (width_px IS NULL AND height_px IS NULL) OR 
        (width_px > 0 AND height_px > 0 AND width_px <= 4096 AND height_px <= 4096)
    ),
    CONSTRAINT chk_display_order CHECK (display_order >= 0),
    CONSTRAINT unique_display_order UNIQUE (product_id, display_order),
    CONSTRAINT unique_storage_key UNIQUE (storage_key)
);

CREATE INDEX idx_product_photos_product_id ON product_photos(product_id);
CREATE INDEX idx_product_photos_tenant_id ON product_photos(tenant_id);
CREATE INDEX idx_product_photos_created_at ON product_photos(created_at);

CREATE UNIQUE INDEX idx_one_primary_per_product 
    ON product_photos(product_id) 
    WHERE is_primary = true;

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_product_photos_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_product_photos_updated_at
    BEFORE UPDATE ON product_photos
    FOR EACH ROW
    EXECUTE FUNCTION update_product_photos_updated_at();
```

Create `backend/migrations/007_add_product_photos.down.sql`:

```sql
DROP TRIGGER IF EXISTS trigger_update_product_photos_updated_at ON product_photos;
DROP FUNCTION IF EXISTS update_product_photos_updated_at();
DROP TABLE IF EXISTS product_photos;
DROP INDEX IF EXISTS idx_tenants_storage;
ALTER TABLE tenants DROP COLUMN IF EXISTS storage_quota_bytes;
ALTER TABLE tenants DROP COLUMN IF EXISTS storage_used_bytes;
```

### Step 2: Run Migration

```bash
cd backend/product-service
go run cmd/migrate/main.go up  # Or your migration command
```

Verify:

```bash
psql -U pos_user -d pos_db -h localhost
\dt product_photos
\d product_photos
\q
```

---

## Phase 3: Dependencies (15 minutes)

### Step 1: Add Go Dependencies

```bash
cd backend/product-service
go get github.com/minio/minio-go/v7@latest
go mod tidy
```

### Step 2: Verify go.mod

Check that `go.mod` includes:

```go
require (
    github.com/disintegration/imaging v1.6.2
    github.com/minio/minio-go/v7 v7.0.63
    // ...existing dependencies
)
```

---

## Phase 4: Core Implementation (3-4 hours)

### Step 1: Create Models

Create `backend/product-service/src/models/product_photo.go`:

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

type ProductPhoto struct {
    ID               uuid.UUID  `json:"id" db:"id"`
    ProductID        uuid.UUID  `json:"product_id" db:"product_id"`
    TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
    StorageKey       string     `json:"storage_key" db:"storage_key"`
    OriginalFilename string     `json:"original_filename" db:"original_filename"`
    FileSizeBytes    int        `json:"file_size_bytes" db:"file_size_bytes"`
    MimeType         string     `json:"mime_type" db:"mime_type"`
    WidthPx          *int       `json:"width_px,omitempty" db:"width_px"`
    HeightPx         *int       `json:"height_px,omitempty" db:"height_px"`
    DisplayOrder     int        `json:"display_order" db:"display_order"`
    IsPrimary        bool       `json:"is_primary" db:"is_primary"`
    CreatedAt        time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
    
    PhotoURL         string     `json:"photo_url,omitempty" db:"-"`
}

type ProductPhotoResponse struct {
    ID               string    `json:"id"`
    ProductID        string    `json:"product_id"`
    OriginalFilename string    `json:"original_filename"`
    FileSizeBytes    int       `json:"file_size_bytes"`
    MimeType         string    `json:"mime_type"`
    WidthPx          *int      `json:"width_px,omitempty"`
    HeightPx         *int      `json:"height_px,omitempty"`
    DisplayOrder     int       `json:"display_order"`
    IsPrimary        bool      `json:"is_primary"`
    PhotoURL         string    `json:"photo_url"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### Step 2: Create Storage Service

Create `backend/product-service/src/services/storage_service.go`:

```go
package services

import (
    "context"
    "fmt"
    "io"
    "time"
    
    "github.com/google/uuid"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
    "github.com/pos/backend/product-service/src/config"
)

type StorageService struct {
    client *minio.Client
    bucket string
}

func NewStorageService() (*StorageService, error) {
    endpoint := config.GetEnv("S3_ENDPOINT", "minio:9000")
    accessKey := config.GetEnv("S3_ACCESS_KEY", "minioadmin")
    secretKey := config.GetEnv("S3_SECRET_KEY", "minioadmin")
    useSSL := config.GetEnv("S3_USE_SSL", "false") == "true"
    bucket := config.GetEnv("S3_BUCKET", "pos-product-photos")
    
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create minio client: %w", err)
    }
    
    return &StorageService{
        client: client,
        bucket: bucket,
    }, nil
}

func (s *StorageService) UploadPhoto(ctx context.Context, tenantID, productID, photoID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (string, error) {
    storageKey := fmt.Sprintf("photos/%s/%s/%s_%d.%s",
        tenantID.String(),
        productID.String(),
        photoID.String(),
        time.Now().Unix(),
        getExtension(filename))
    
    _, err := s.client.PutObject(ctx, s.bucket, storageKey, reader, size,
        minio.PutObjectOptions{
            ContentType: contentType,
            UserMetadata: map[string]string{
                "tenant-id":   tenantID.String(),
                "product-id":  productID.String(),
                "photo-id":    photoID.String(),
            },
        })
    
    if err != nil {
        return "", fmt.Errorf("failed to upload to S3: %w", err)
    }
    
    return storageKey, nil
}

func (s *StorageService) GetPhotoURL(ctx context.Context, storageKey string, expiryHours int) (string, error) {
    url, err := s.client.PresignedGetObject(ctx, s.bucket, storageKey,
        time.Duration(expiryHours)*time.Hour, nil)
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned URL: %w", err)
    }
    return url.String(), nil
}

func (s *StorageService) DeletePhoto(ctx context.Context, storageKey string) error {
    err := s.client.RemoveObject(ctx, s.bucket, storageKey, minio.RemoveObjectOptions{})
    if err != nil {
        return fmt.Errorf("failed to delete from S3: %w", err)
    }
    return nil
}

func getExtension(filename string) string {
    // Extract extension from filename
    // Implementation details...
    return "jpg"
}
```

### Step 3: Create Repository

Create `backend/product-service/src/repository/photo_repository.go`:

```go
package repository

import (
    "context"
    "database/sql"
    
    "github.com/google/uuid"
    "github.com/pos/backend/product-service/src/models"
)

type PhotoRepository struct {
    db *sql.DB
}

func NewPhotoRepository(db *sql.DB) *PhotoRepository {
    return &PhotoRepository{db: db}
}

func (r *PhotoRepository) Create(ctx context.Context, photo *models.ProductPhoto) error {
    query := `
        INSERT INTO product_photos (
            id, product_id, tenant_id, storage_key, original_filename,
            file_size_bytes, mime_type, width_px, height_px,
            display_order, is_primary
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING created_at, updated_at`
    
    return r.db.QueryRowContext(ctx, query,
        photo.ID, photo.ProductID, photo.TenantID, photo.StorageKey,
        photo.OriginalFilename, photo.FileSizeBytes, photo.MimeType,
        photo.WidthPx, photo.HeightPx, photo.DisplayOrder, photo.IsPrimary,
    ).Scan(&photo.CreatedAt, &photo.UpdatedAt)
}

func (r *PhotoRepository) GetByProduct(ctx context.Context, productID, tenantID uuid.UUID) ([]models.ProductPhoto, error) {
    query := `
        SELECT * FROM product_photos
        WHERE product_id = $1 AND tenant_id = $2
        ORDER BY display_order ASC`
    
    // Implementation...
    return nil, nil
}

func (r *PhotoRepository) Delete(ctx context.Context, photoID, tenantID uuid.UUID) error {
    query := `DELETE FROM product_photos WHERE id = $1 AND tenant_id = $2`
    _, err := r.db.ExecContext(ctx, query, photoID, tenantID)
    return err
}

// Additional methods...
```

### Step 4: Create API Handlers

Create `backend/product-service/api/photo_handler.go`:

```go
package api

import (
    "net/http"
    
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "github.com/pos/backend/product-service/src/services"
)

type PhotoHandler struct {
    photoService *services.PhotoService
}

func NewPhotoHandler(photoService *services.PhotoService) *PhotoHandler {
    return &PhotoHandler{photoService: photoService}
}

func (h *PhotoHandler) UploadPhoto(c echo.Context) error {
    tenantID := c.Get("tenant_id").(uuid.UUID)
    productID, _ := uuid.Parse(c.Param("product_id"))
    
    file, err := c.FormFile("photo")
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "No file uploaded")
    }
    
    src, err := file.Open()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open file")
    }
    defer src.Close()
    
    photo, err := h.photoService.UploadPhoto(c.Request().Context(), tenantID, productID, file)
    if err != nil {
        // Handle different error types
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    
    return c.JSON(http.StatusCreated, map[string]interface{}{
        "status": "success",
        "data":   photo,
    })
}

func (h *PhotoHandler) ListPhotos(c echo.Context) error {
    tenantID := c.Get("tenant_id").(uuid.UUID)
    productID, _ := uuid.Parse(c.Param("product_id"))
    
    photos, err := h.photoService.ListPhotos(c.Request().Context(), productID, tenantID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status": "success",
        "data":   photos,
    })
}

// Additional handlers...
```

### Step 5: Register Routes

Edit `backend/product-service/main.go`:

```go
// Initialize services
storageService, err := services.NewStorageService()
if err != nil {
    log.Fatal("Failed to initialize storage service:", err)
}

photoRepo := repository.NewPhotoRepository(config.DB)
photoService := services.NewPhotoService(photoRepo, storageService)
photoHandler := api.NewPhotoHandler(photoService)

// Register routes
productGroup := e.Group("/api/v1/products")
productGroup.POST("/:product_id/photos", photoHandler.UploadPhoto, authMiddleware, tenantMiddleware)
productGroup.GET("/:product_id/photos", photoHandler.ListPhotos, authMiddleware, tenantMiddleware)
productGroup.GET("/:product_id/photos/:photo_id", photoHandler.GetPhoto, authMiddleware, tenantMiddleware)
productGroup.DELETE("/:product_id/photos/:photo_id", photoHandler.DeletePhoto, authMiddleware, tenantMiddleware)
productGroup.PUT("/:product_id/photos/reorder", photoHandler.ReorderPhotos, authMiddleware, tenantMiddleware)
```

---

## Phase 5: Testing (1-2 hours)

### Step 1: Unit Tests

Create `backend/product-service/tests/unit/storage_service_test.go`:

```go
package unit

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestStorageService_UploadPhoto(t *testing.T) {
    // Mock MinIO client
    // Test upload logic
}
```

### Step 2: Integration Tests

Create `backend/product-service/tests/integration/photo_api_test.go`:

```go
package integration

import (
    "testing"
    "bytes"
    "mime/multipart"
)

func TestPhotoUpload_Success(t *testing.T) {
    // Create test image
    // Upload via API
    // Verify database record
    // Verify S3 object exists
}
```

### Step 3: Manual Testing

```bash
# Upload photo
curl -X POST "http://localhost:8086/api/v1/products/<product-id>/photos" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <tenant-id>" \
  -F "photo=@test-image.jpg" \
  -F "is_primary=true"

# List photos
curl -X GET "http://localhost:8086/api/v1/products/<product-id>/photos" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <tenant-id>"

# Delete photo
curl -X DELETE "http://localhost:8086/api/v1/products/<product-id>/photos/<photo-id>" \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <tenant-id>"
```

---

## Troubleshooting

### MinIO Connection Issues

**Problem**: Cannot connect to MinIO  
**Solution**: Check Docker network, ensure service is running:

```bash
docker ps | grep minio
docker logs pos-minio
ping minio  # From within product-service container
```

### Upload Fails

**Problem**: "Unable to upload to S3"  
**Check**:
1. Bucket exists: `docker exec -it pos-minio mc ls local`
2. Credentials correct in .env
3. File size under 10MB
4. MIME type supported

### Photo URLs Don't Work

**Problem**: Presigned URLs return 403  
**Solution**: Check S3_USE_SSL matches MinIO configuration, verify bucket permissions

### Database Constraint Violation

**Problem**: "duplicate key value violates unique constraint"  
**Check**:
1. Display order conflicts
2. Multiple primary photos
3. Duplicate storage keys

---

## Checklist

- [ ] MinIO running in Docker
- [ ] Database migrations applied
- [ ] Go dependencies installed
- [ ] StorageService implemented
- [ ] PhotoRepository implemented
- [ ] API handlers created
- [ ] Routes registered
- [ ] Unit tests written
- [ ] Integration tests passing
- [ ] Manual API test successful
- [ ] Error handling complete
- [ ] Logging added
- [ ] Documentation updated

---

## Next Steps

1. **Frontend Integration**: Update product forms to support photo upload
2. **Photo Gallery**: Implement image carousel on product detail page
3. **Optimization**: Add image optimization pipeline (resize, compress)
4. **Monitoring**: Set up storage usage alerts
5. **Migration Script**: Create tool to migrate existing photos from filesystem

---

## Resources

- [MinIO Go SDK Documentation](https://min.io/docs/minio/linux/developers/go/API.html)
- [Echo Framework Guide](https://echo.labstack.com/guide/)
- [Data Model](../data-model.md)
- [API Contracts](../contracts/api.md)
- [Research Decisions](../research.md)

---

## Support

If you encounter issues:
1. Check logs: `docker-compose logs product-service minio`
2. Review error responses for specific error codes
3. Verify environment variables are set correctly
4. Consult the research.md for design decisions
5. Open an issue with reproduction steps

**Estimated Total Time**: 4-6 hours for MVP implementation
