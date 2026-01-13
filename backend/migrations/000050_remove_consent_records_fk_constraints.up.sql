-- Migration 000050: Remove foreign key constraints from consent_records
-- Purpose: Allow storing tenant_id or user_id in subject_id without FK constraint
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)
-- Rationale: Application-level validation via CHECK constraint is sufficient

-- Drop foreign key constraints
ALTER TABLE consent_records
DROP CONSTRAINT IF EXISTS consent_records_subject_id_fkey,
DROP CONSTRAINT IF EXISTS consent_records_guest_order_id_fkey;

-- Add comments explaining the change
COMMENT ON COLUMN consent_records.subject_id IS 'User ID or Tenant ID for tenant-type consents (no FK constraint - application enforced)';

COMMENT ON COLUMN consent_records.guest_order_id IS 'Guest order reference for guest-type consents (no FK constraint - application enforced)';