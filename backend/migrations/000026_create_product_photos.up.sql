-- Migration: 000026_create_product_photos.up.sql
-- Feature: 005-product-photo-storage
-- Description: Add product_photos table and storage tracking to tenants

-- Add storage tracking columns to tenants table
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS storage_used_bytes BIGINT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS storage_quota_bytes BIGINT NOT NULL DEFAULT 5368709120;
-- 5GB default

COMMENT ON COLUMN tenants.storage_used_bytes IS 'Total storage used by tenant in bytes (sum of all product photo file sizes)';

COMMENT ON COLUMN tenants.storage_quota_bytes IS 'Storage quota limit for tenant in bytes (default 5GB)';

-- Create product_photos table
CREATE TABLE IF NOT EXISTS product_photos (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    tenant_id UUID NOT NULL,

-- Storage information
storage_key TEXT NOT NULL, -- S3 object key: photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.ext
original_filename TEXT NOT NULL, -- User's original filename (sanitized)
file_size_bytes INTEGER NOT NULL, -- File size in bytes for quota tracking
mime_type TEXT NOT NULL, -- image/jpeg, image/png, image/webp, image/gif

-- Image dimensions
width_px INTEGER, -- Image width in pixels
height_px INTEGER, -- Image height in pixels

-- Display configuration
display_order INTEGER NOT NULL DEFAULT 0, -- Order in product photo carousel (0 = first)
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
CREATE INDEX IF NOT EXISTS idx_product_photos_product_id ON product_photos (product_id);

CREATE INDEX IF NOT EXISTS idx_product_photos_tenant_id ON product_photos (tenant_id);

CREATE INDEX IF NOT EXISTS idx_product_photos_created_at ON product_photos (created_at);

CREATE INDEX IF NOT EXISTS idx_product_photos_primary ON product_photos (product_id, is_primary)
WHERE
    is_primary = true;

-- Ensure only one primary photo per product
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_primary_per_product ON product_photos (product_id)
WHERE
    is_primary = true;

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

-- Add comments for documentation
COMMENT ON TABLE product_photos IS 'Stores metadata for product photos in object storage (S3/MinIO)';

COMMENT ON COLUMN product_photos.storage_key IS 'S3 object key following pattern: photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.ext';

COMMENT ON COLUMN product_photos.display_order IS 'Zero-based ordering for photo carousel display (0 = first photo)';

COMMENT ON COLUMN product_photos.is_primary IS 'Primary photo displayed in product listings (only one per product)';