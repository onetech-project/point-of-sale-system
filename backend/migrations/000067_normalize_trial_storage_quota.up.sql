-- Migration: 000067_normalize_trial_storage_quota
-- Description: Make the subscription trial storage quota default authoritative.

ALTER TABLE tenants
  ALTER COLUMN storage_quota_bytes SET DEFAULT 2147483648;

UPDATE tenants
SET storage_quota_bytes = 2147483648
WHERE storage_quota_bytes = 5368709120;

COMMENT ON COLUMN tenants.storage_quota_bytes IS 'Storage quota limit for tenant in bytes (default 2 GB = 2147483648 bytes)';
