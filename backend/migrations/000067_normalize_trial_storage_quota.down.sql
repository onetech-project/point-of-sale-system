-- Migration: 000067_normalize_trial_storage_quota rollback
-- Restore the earlier product-photo default without rewriting tenant-specific quotas.

ALTER TABLE tenants
  ALTER COLUMN storage_quota_bytes SET DEFAULT 5368709120;

COMMENT ON COLUMN tenants.storage_quota_bytes IS 'Storage quota limit for tenant in bytes (default 5GB)';
