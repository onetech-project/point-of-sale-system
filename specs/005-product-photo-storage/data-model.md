# Data Model: Product Photo Storage

**Feature**: 005-product-photo-storage  
**Date**: December 12, 2025  
**Status**: Phase 1 Design

## Overview

This document defines the data structures for product photo storage in object storage (S3/MinIO). The model extends the existing product schema with a new `product_photos` table and enhances the `tenants` table for storage quota tracking.

---

## Entity Relationship Diagram

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────┐
│    tenants      │         │    products      │         │ product_photos  │
├─────────────────┤         ├──────────────────┤         ├─────────────────┤
│ id (PK)         │────┐    │ id (PK)          │────┐    │ id (PK)         │
│ name            │    │    │ tenant_id (FK)   │    │    │ product_id (FK) │
│ ...existing...  │    └───→│ sku              │    └───→│ tenant_id (FK)  │
│ storage_used    │         │ name             │         │ storage_key     │
│ storage_quota   │         │ ...existing...   │         │ filename        │
└─────────────────┘         └──────────────────┘         │ file_size       │
                                                          │ mime_type       │
                                                          │ width_px        │
                                                          │ height_px       │
                                                          │ display_order   │
                                                          │ is_primary      │
                                                          │ created_at      │
                                                          │ updated_at      │
                                                          └─────────────────┘
```

**Relationships**:
- One Tenant has many Products (existing)
- One Tenant has many Product Photos (new)
- One Product has many Product Photos (new, 0..5 photos per product)
- One Product Photo belongs to one Product and one Tenant

---

## Database Schema

### New Table: `product_photos`

Stores metadata for product photos stored in object storage.

```sql
CREATE TABLE product_photos (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Storage information
    storage_key TEXT NOT NULL,              -- S3 object key: photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.ext
    original_filename TEXT NOT NULL,        -- User's original filename (sanitized)
    file_size_bytes INTEGER NOT NULL,       -- File size in bytes for quota tracking
    mime_type TEXT NOT NULL,                -- image/jpeg, image/png, image/webp, image/gif
    
    -- Image dimensions
    width_px INTEGER,                       -- Image width in pixels
    height_px INTEGER,                      -- Image height in pixels
    
    -- Display configuration
    display_order INTEGER NOT NULL DEFAULT 0,  -- Order in product photo carousel (0 = first)
    is_primary BOOLEAN NOT NULL DEFAULT false, -- Primary photo shown in listings
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT fk_product FOREIGN KEY (product_id) 
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT chk_file_size CHECK (file_size_bytes > 0 AND file_size_bytes <= 10485760),  -- Max 10MB
    CONSTRAINT chk_dimensions CHECK (
        (width_px IS NULL AND height_px IS NULL) OR 
        (width_px > 0 AND height_px > 0 AND width_px <= 4096 AND height_px <= 4096)
    ),
    CONSTRAINT chk_display_order CHECK (display_order >= 0),
    CONSTRAINT unique_display_order UNIQUE (product_id, display_order),
    CONSTRAINT unique_storage_key UNIQUE (storage_key)
);

-- Indexes for efficient queries
CREATE INDEX idx_product_photos_product_id ON product_photos(product_id);
CREATE INDEX idx_product_photos_tenant_id ON product_photos(tenant_id);
CREATE INDEX idx_product_photos_created_at ON product_photos(created_at);
CREATE INDEX idx_product_photos_primary ON product_photos(product_id, is_primary) WHERE is_primary = true;

-- Trigger to update updated_at timestamp
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

-- Ensure only one primary photo per product
CREATE UNIQUE INDEX idx_one_primary_per_product 
    ON product_photos(product_id) 
    WHERE is_primary = true;
```

**Field Descriptions**:

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | UUID | No | Unique photo identifier |
| `product_id` | UUID | No | Foreign key to products table |
| `tenant_id` | UUID | No | Foreign key to tenants table (for isolation) |
| `storage_key` | TEXT | No | S3 object key (path in bucket) |
| `original_filename` | TEXT | No | Original filename from upload (sanitized) |
| `file_size_bytes` | INTEGER | No | File size for quota tracking (1 to 10MB) |
| `mime_type` | TEXT | No | Image MIME type (validated at upload) |
| `width_px` | INTEGER | Yes | Image width (NULL if not decoded) |
| `height_px` | INTEGER | Yes | Image height (NULL if not decoded) |
| `display_order` | INTEGER | No | Order in carousel (0-based, unique per product) |
| `is_primary` | BOOLEAN | No | Primary photo for product listings (only one per product) |
| `created_at` | TIMESTAMP | No | Upload timestamp |
| `updated_at` | TIMESTAMP | No | Last modification timestamp |

**Business Rules**:
1. Each product can have 0-5 photos (enforced at application level)
2. Only one photo per product can be `is_primary = true` (enforced by unique index)
3. Display order must be sequential within a product (0, 1, 2, 3, 4)
4. Deleting a product cascades to delete all associated photos
5. Deleting a tenant cascades to delete all photos (and products)

---

### Modified Table: `tenants`

Add storage quota tracking fields to existing tenants table.

```sql
-- Migration: Add storage tracking to tenants
ALTER TABLE tenants 
    ADD COLUMN storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN storage_quota_bytes BIGINT NOT NULL DEFAULT 5368709120;  -- 5GB default

-- Add constraint to prevent negative usage
ALTER TABLE tenants 
    ADD CONSTRAINT chk_storage_non_negative CHECK (storage_used_bytes >= 0);

-- Index for quota monitoring queries
CREATE INDEX idx_tenants_storage ON tenants(id, storage_used_bytes);
```

**New Fields**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `storage_used_bytes` | BIGINT | 0 | Current storage usage in bytes |
| `storage_quota_bytes` | BIGINT | 5GB | Maximum allowed storage in bytes |

**Business Rules**:
1. Storage quota can be adjusted per tenant by platform admin
2. Storage usage is incremented atomically on successful photo upload
3. Storage usage is decremented atomically on photo deletion
4. Upload is rejected if `storage_used_bytes + new_file_size > storage_quota_bytes`

---

### Modified Table: `products` (Optional Enhancement)

Add denormalized fields for quick access to primary photo (optional optimization).

```sql
-- Optional: Add primary photo reference to products table
ALTER TABLE products 
    ADD COLUMN primary_photo_id UUID,
    ADD COLUMN photo_count INTEGER NOT NULL DEFAULT 0;

-- Foreign key to primary photo (nullable, as products may have no photos)
ALTER TABLE products 
    ADD CONSTRAINT fk_primary_photo 
    FOREIGN KEY (primary_photo_id) 
    REFERENCES product_photos(id) ON DELETE SET NULL;

-- Index for quick photo lookup
CREATE INDEX idx_products_primary_photo ON products(primary_photo_id) WHERE primary_photo_id IS NOT NULL;
```

**Note**: This is an optimization. Primary photo can also be queried from `product_photos` table with `is_primary = true`. Denormalization trades consistency for performance.

---

## Go Data Structures

### ProductPhoto Model

```go
package models

import (
	"time"
	"github.com/google/uuid"
)

// ProductPhoto represents a photo associated with a product
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
	
	// Computed fields (not in database)
	PhotoURL         string     `json:"photo_url,omitempty" db:"-"`  // Presigned S3 URL
}

// ProductPhotoUploadRequest represents the API request for uploading a photo
type ProductPhotoUploadRequest struct {
	ProductID    uuid.UUID `json:"product_id" validate:"required"`
	DisplayOrder int       `json:"display_order" validate:"gte=0,lte=4"`
	IsPrimary    bool      `json:"is_primary"`
}

// ProductPhotoResponse is the API response for photo operations
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
	PhotoURL         string    `json:"photo_url"`  // Presigned S3 URL for direct access
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ProductWithPhotos extends Product with photo information
type ProductWithPhotos struct {
	Product
	Photos []ProductPhotoResponse `json:"photos"`
}
```

### Tenant Model Extension

```go
// Add to existing Tenant struct in models/tenant.go
type Tenant struct {
	// ...existing fields...
	
	StorageUsedBytes  int64 `json:"storage_used_bytes" db:"storage_used_bytes"`
	StorageQuotaBytes int64 `json:"storage_quota_bytes" db:"storage_quota_bytes"`
}

// StorageQuotaInfo provides storage usage information
type StorageQuotaInfo struct {
	UsedBytes      int64   `json:"used_bytes"`
	QuotaBytes     int64   `json:"quota_bytes"`
	AvailableBytes int64   `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

func (t *Tenant) GetStorageQuotaInfo() StorageQuotaInfo {
	available := t.StorageQuotaBytes - t.StorageUsedBytes
	if available < 0 {
		available = 0
	}
	
	percent := 0.0
	if t.StorageQuotaBytes > 0 {
		percent = float64(t.StorageUsedBytes) / float64(t.StorageQuotaBytes) * 100
	}
	
	return StorageQuotaInfo{
		UsedBytes:      t.StorageUsedBytes,
		QuotaBytes:     t.StorageQuotaBytes,
		AvailableBytes: available,
		UsagePercent:   percent,
	}
}
```

---

## Storage Key Format

Object storage keys follow a structured pattern for organization and access control:

```
photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.{extension}
```

**Example**:
```
photos/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/a1b2c3d4-5678-90ef-ghij-klmnopqrstuv_1734048000.jpg
```

**Components**:
- `photos/`: Root prefix for all product photos
- `{tenant_id}`: UUID of the tenant (for isolation)
- `{product_id}`: UUID of the product
- `{photo_id}`: UUID of the photo record
- `{timestamp}`: Unix timestamp of upload (for uniqueness)
- `{extension}`: File extension (jpg, png, webp, gif)

**Benefits**:
1. **Tenant Isolation**: All photos for a tenant grouped under `photos/{tenant_id}/`
2. **Product Grouping**: All photos for a product grouped under `photos/{tenant_id}/{product_id}/`
3. **Uniqueness**: Combination of photo_id + timestamp prevents collisions
4. **Queryability**: Can list all photos for a tenant or product using prefix queries
5. **Security**: IAM policies can restrict access by tenant_id prefix

---

## Data Validation Rules

### Upload Validation

| Field | Rule | Validation |
|-------|------|------------|
| File Size | Max 10MB | `file_size_bytes <= 10485760` |
| MIME Type | Allowed types | `image/jpeg, image/png, image/webp, image/gif` |
| Dimensions | Max 4096x4096 | `width_px <= 4096 AND height_px <= 4096` |
| Filename | Sanitized | Remove special chars, max 255 chars |
| Display Order | Range | `0 <= display_order <= 4` |
| Photos per Product | Max 5 | Count existing photos before upload |
| Storage Quota | Tenant limit | `storage_used + file_size <= storage_quota` |

### Integrity Rules

1. **Primary Photo**: Only one `is_primary = true` per product (database constraint)
2. **Display Order**: Unique per product (database constraint)
3. **Cascade Delete**: Deleting product removes all photos (foreign key cascade)
4. **Tenant Ownership**: `product.tenant_id = photo.tenant_id` (validated at application level)
5. **Storage Consistency**: Storage key must exist in S3 when record exists in database

---

## Migration Strategy

### Phase 1: Add New Tables (This Feature)

```sql
-- Run migration script
-- 1. Create product_photos table
-- 2. Add storage tracking to tenants
-- 3. Optionally add primary_photo_id to products
```

### Phase 2: Dual-Mode Support

Application code supports both old and new photo storage:
- Photos with `photo_path` = local path: serve from filesystem (legacy)
- Photos with `storage_key` in `product_photos` table: serve from S3 (new)

### Phase 3: Data Migration (Manual, Post-Feature)

Separate migration script to move existing photos:
1. Read photos from `products.photo_path` and local filesystem
2. Upload to S3 with correct tenant prefix
3. Create `product_photos` record
4. Update `products.photo_path` to NULL
5. Validate and report migration status

---

## Data Access Patterns

### Common Queries

1. **Get all photos for a product**:
```sql
SELECT * FROM product_photos 
WHERE product_id = $1 AND tenant_id = $2 
ORDER BY display_order ASC;
```

2. **Get primary photo for a product**:
```sql
SELECT * FROM product_photos 
WHERE product_id = $1 AND tenant_id = $2 AND is_primary = true 
LIMIT 1;
```

3. **Get tenant storage usage**:
```sql
SELECT storage_used_bytes, storage_quota_bytes 
FROM tenants 
WHERE id = $1;
```

4. **Count photos for a product**:
```sql
SELECT COUNT(*) FROM product_photos 
WHERE product_id = $1 AND tenant_id = $2;
```

5. **List all photos for a tenant** (admin/reporting):
```sql
SELECT * FROM product_photos 
WHERE tenant_id = $1 
ORDER BY created_at DESC 
LIMIT 100;
```

### Performance Considerations

- **Indexes**: All foreign keys (product_id, tenant_id) are indexed
- **Pagination**: Use LIMIT/OFFSET for large result sets
- **Caching**: Photo URLs cached in Redis (6-day TTL for public products)
- **Denormalization**: Consider caching photo count on products table for quick access

---

## Data Lifecycle

### Photo Upload Lifecycle

1. **Validation**: Check file size, type, dimensions, quota
2. **Image Processing**: Resize if needed, optimize quality
3. **S3 Upload**: Upload to object storage with tenant prefix
4. **Database Insert**: Create product_photos record with metadata
5. **Quota Update**: Increment tenant storage_used_bytes
6. **URL Generation**: Generate presigned URL for immediate access
7. **Cache**: Store URL in Redis if public product

### Photo Deletion Lifecycle

1. **Authorization**: Verify tenant owns the photo
2. **Database Lookup**: Get storage_key and file_size
3. **Database Delete**: Remove product_photos record (transaction start)
4. **S3 Delete**: Remove object from storage
5. **Quota Update**: Decrement tenant storage_used_bytes (transaction commit)
6. **Cache Invalidation**: Remove cached URL from Redis
7. **Cleanup**: Handle failures with retry queue

### Product Deletion Lifecycle

1. **Foreign Key Cascade**: Database automatically deletes product_photos records
2. **Async Cleanup**: Background job finds orphaned S3 objects and deletes them
3. **Quota Reconciliation**: Recalculate tenant storage usage

---

## Summary

This data model provides:
- ✅ **Multi-tenant isolation**: Tenant ID in every record
- ✅ **Scalability**: Supports unlimited photos via object storage
- ✅ **Integrity**: Foreign key constraints and unique indexes
- ✅ **Quota management**: Track storage usage per tenant
- ✅ **Flexibility**: Supports multiple photos per product with ordering
- ✅ **Performance**: Indexed queries, URL caching
- ✅ **Migration path**: Dual-mode support for gradual migration

Next: Define API contracts in `contracts/` directory.
