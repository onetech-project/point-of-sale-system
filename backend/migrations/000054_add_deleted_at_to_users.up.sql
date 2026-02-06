-- Migration: 000054_add_deleted_at_to_users
-- Description: Add deleted_at column for soft delete with 90-day retention per UU PDP Article 5
-- Author: AI Assistant
-- Date: 2026-01-14

ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- Create index for cleanup job queries
CREATE INDEX idx_users_deleted_at ON users (deleted_at)
WHERE
    deleted_at IS NOT NULL;

-- Comments for documentation
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp - user data retained for 90 days per UU PDP compliance';

-- Create table for tracking deletion notifications (to avoid duplicate emails)
CREATE TABLE IF NOT EXISTS user_deletion_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL,
    notified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_type VARCHAR(50) NOT NULL DEFAULT 'upcoming_deletion',
    CONSTRAINT fk_user_deletion_notifications_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_deletion_notifications_user_id ON user_deletion_notifications (user_id);

COMMENT ON TABLE user_deletion_notifications IS 'Tracks deletion warning notifications sent to users (30 days before permanent deletion)';