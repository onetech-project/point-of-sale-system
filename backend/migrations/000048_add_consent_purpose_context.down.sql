-- Migration 000048 down: Remove context field from consent_purposes

-- Drop index
DROP INDEX IF EXISTS idx_consent_purposes_context;

-- Remove context column
ALTER TABLE consent_purposes DROP COLUMN IF EXISTS context;