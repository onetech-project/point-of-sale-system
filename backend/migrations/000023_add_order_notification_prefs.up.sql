-- Add notification preference to users table
ALTER TABLE users
ADD COLUMN receive_order_notifications BOOLEAN DEFAULT false;

-- Index for efficient queries
CREATE INDEX idx_users_order_notifications ON users (
    tenant_id,
    receive_order_notifications
)
WHERE
    receive_order_notifications = true;

COMMENT ON COLUMN users.receive_order_notifications IS 'Whether user receives email notifications for paid orders';