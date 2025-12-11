-- Migration: 000024_allow_in_app_notification_type
-- Description: Expand notifications.type CHECK to include 'in_app'
-- Date: 2025-12-11

BEGIN;

-- Drop existing constraint if present
ALTER TABLE notifications
DROP CONSTRAINT IF EXISTS notifications_type_check;

-- Add updated check constraint including 'in_app'
ALTER TABLE notifications
ADD CONSTRAINT notifications_type_check CHECK (
    type IN (
        'email',
        'sms',
        'push',
        'in_app'
    )
);

COMMIT;