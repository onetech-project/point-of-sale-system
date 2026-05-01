-- Migration: 000060_add_offline_orders.up.sql
-- Purpose: Extend guest_orders table to support offline order recording
-- Features: order_type distinction, data consent fields, user tracking

ALTER TABLE guest_orders
ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'online',
ADD COLUMN IF NOT EXISTS data_consent_given BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS consent_method VARCHAR(20),
ADD COLUMN IF NOT EXISTS recorded_by_user_id UUID REFERENCES users (id),
ADD COLUMN IF NOT EXISTS last_modified_by_user_id UUID REFERENCES users (id),
ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMP;

-- Add CHECK constraint for order_type
ALTER TABLE guest_orders
ADD CONSTRAINT check_order_type CHECK (
    order_type IN ('online', 'offline')
);

-- Add CHECK constraint for consent_method
ALTER TABLE guest_orders
ADD CONSTRAINT check_consent_method CHECK (
    consent_method IS NULL
    OR consent_method IN (
        'verbal',
        'written',
        'digital'
    )
);

-- Create index for offline order queries
CREATE INDEX IF NOT EXISTS idx_guest_orders_type_status ON guest_orders (order_type, status, tenant_id);

-- Create index for user tracking
CREATE INDEX IF NOT EXISTS idx_guest_orders_recorded_by ON guest_orders (recorded_by_user_id)
WHERE
    order_type = 'offline';

-- Create partial index for pending payment offline orders
CREATE INDEX IF NOT EXISTS idx_offline_orders_pending_payment ON guest_orders (tenant_id, created_at DESC)
WHERE
    order_type = 'offline'
    AND status = 'PENDING';

-- Add column comments for documentation
COMMENT ON COLUMN guest_orders.order_type IS 'Distinguishes online (public self-service) vs offline (staff-recorded) orders';

COMMENT ON COLUMN guest_orders.data_consent_given IS 'UU PDP/GDPR: Whether customer explicitly consented to data collection';

COMMENT ON COLUMN guest_orders.consent_method IS 'How consent was obtained: verbal, written form, digital signature';

COMMENT ON COLUMN guest_orders.recorded_by_user_id IS 'Staff user who created the offline order';

COMMENT ON COLUMN guest_orders.last_modified_by_user_id IS 'Staff user who last edited the order';