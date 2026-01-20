-- Rollback: Remove notified_of_deletion flag from users table

BEGIN;

-- Drop index
DROP INDEX IF EXISTS idx_users_deletion_notification;

-- Remove column
ALTER TABLE users DROP COLUMN IF EXISTS notified_of_deletion;

COMMIT;