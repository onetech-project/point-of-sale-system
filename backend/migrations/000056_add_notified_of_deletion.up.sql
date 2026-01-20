-- Migration: Add notified_of_deletion flag to users table (T187)
-- Feature: 006-uu-pdp-compliance - User Story 8 (Data Retention and Cleanup)
-- Purpose: Track whether users have been notified about pending deletion

BEGIN;

-- Add column for deletion notification tracking
ALTER TABLE users
ADD COLUMN notified_of_deletion BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for deletion notification queries
-- Query pattern: WHERE deleted_at IS NOT NULL AND notified_of_deletion = false AND deleted_at < NOW() - INTERVAL '60 days'
CREATE INDEX idx_users_deletion_notification ON users (
    deleted_at,
    notified_of_deletion
)
WHERE
    deleted_at IS NOT NULL;

COMMIT;