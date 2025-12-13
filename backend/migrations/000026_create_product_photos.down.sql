-- Migration: 000026_create_product_photos.down.sql
-- Feature: 005-product-photo-storage
-- Description: Rollback product_photos table and storage tracking

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_product_photos_updated_at ON product_photos;

-- Drop function
DROP FUNCTION IF EXISTS update_product_photos_updated_at ();

-- Drop indexes
DROP INDEX IF EXISTS idx_one_primary_per_product;

DROP INDEX IF EXISTS idx_product_photos_primary;

DROP INDEX IF EXISTS idx_product_photos_created_at;

DROP INDEX IF EXISTS idx_product_photos_tenant_id;

DROP INDEX IF EXISTS idx_product_photos_product_id;

-- Drop product_photos table
DROP TABLE IF EXISTS product_photos;

-- Remove storage tracking columns from tenants table
ALTER TABLE tenants
DROP COLUMN IF EXISTS storage_quota_bytes,
DROP COLUMN IF EXISTS storage_used_bytes;