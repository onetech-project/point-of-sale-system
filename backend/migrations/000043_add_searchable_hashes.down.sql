-- Rollback searchable hash columns for encrypted fields

-- Remove comments
COMMENT ON COLUMN notifications.recipient_hash IS NULL;

COMMENT ON COLUMN guest_orders.customer_email_hash IS NULL;

COMMENT ON COLUMN invitations.token_hash IS NULL;

COMMENT ON COLUMN invitations.email_hash IS NULL;

COMMENT ON COLUMN users.email_hash IS NULL;

-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_recipient_hash;

DROP INDEX IF EXISTS idx_guest_orders_email_hash;

DROP INDEX IF EXISTS idx_invitations_token_hash;

DROP INDEX IF EXISTS idx_invitations_email_hash;

DROP INDEX IF EXISTS idx_users_tenant_email_hash;

DROP INDEX IF EXISTS idx_users_email_hash;

-- Drop columns
ALTER TABLE notifications DROP COLUMN IF EXISTS recipient_hash;

ALTER TABLE guest_orders DROP COLUMN IF EXISTS customer_email_hash;

ALTER TABLE invitations DROP COLUMN IF EXISTS token_hash;

ALTER TABLE invitations DROP COLUMN IF EXISTS email_hash;

ALTER TABLE users DROP COLUMN IF EXISTS email_hash;