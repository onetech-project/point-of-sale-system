-- Migration: 000060_add_offline_orders.down.sql
-- Purpose: Rollback offline order support from guest_orders table

-- Drop indexes first
DROP INDEX IF EXISTS idx_offline_orders_pending_payment;

DROP INDEX IF EXISTS idx_guest_orders_recorded_by;

DROP INDEX IF EXISTS idx_guest_orders_type_status;

-- Drop constraints
ALTER TABLE guest_orders
DROP CONSTRAINT IF EXISTS check_consent_method,
DROP CONSTRAINT IF EXISTS check_order_type;

-- Drop columns
ALTER TABLE guest_orders
DROP COLUMN IF EXISTS last_modified_at,
DROP COLUMN IF EXISTS last_modified_by_user_id,
DROP COLUMN IF EXISTS recorded_by_user_id,
DROP COLUMN IF EXISTS consent_method,
DROP COLUMN IF EXISTS data_consent_given,
DROP COLUMN IF EXISTS order_type;