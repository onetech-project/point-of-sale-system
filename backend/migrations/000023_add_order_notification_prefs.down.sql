DROP INDEX IF EXISTS idx_users_order_notifications;

ALTER TABLE users DROP COLUMN IF EXISTS receive_order_notifications;