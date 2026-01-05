-- Add searchable hash columns for encrypted fields
-- These HMAC hashes allow efficient lookups without decrypting all records

-- Users table: add email_hash for login lookups
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_users_email_hash ON users (email_hash);

CREATE INDEX IF NOT EXISTS idx_users_tenant_email_hash ON users (tenant_id, email_hash);

-- Invitations table: add email_hash and token_hash for lookups
ALTER TABLE invitations
ADD COLUMN IF NOT EXISTS email_hash VARCHAR(64);

ALTER TABLE invitations
ADD COLUMN IF NOT EXISTS token_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_invitations_email_hash ON invitations (tenant_id, email_hash, status);

CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations (token_hash, status);

-- Guest orders table: add email_hash for duplicate detection
ALTER TABLE guest_orders
ADD COLUMN IF NOT EXISTS customer_email_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_guest_orders_email_hash ON guest_orders (customer_email_hash);

-- Notifications table: add recipient_hash for lookups
ALTER TABLE notifications
ADD COLUMN IF NOT EXISTS recipient_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient_hash ON notifications (tenant_id, recipient_hash);

-- Comments
COMMENT ON COLUMN users.email_hash IS 'HMAC-SHA256 hash of email for efficient lookups';

COMMENT ON COLUMN invitations.email_hash IS 'HMAC-SHA256 hash of email for efficient lookups';

COMMENT ON COLUMN invitations.token_hash IS 'HMAC-SHA256 hash of token for efficient lookups';

COMMENT ON COLUMN guest_orders.customer_email_hash IS 'HMAC-SHA256 hash of customer_email for efficient lookups';

COMMENT ON COLUMN notifications.recipient_hash IS 'HMAC-SHA256 hash of recipient for efficient lookups';