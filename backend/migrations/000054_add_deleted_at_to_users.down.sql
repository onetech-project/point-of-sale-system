-- Rollback migration: 000054_add_deleted_at_to_users
-- Description: Remove deleted_at column and user_deletion_notifications table

DROP TABLE IF EXISTS user_deletion_notifications;

DROP INDEX IF EXISTS idx_users_deleted_at;

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;