-- Drop indexes
DROP INDEX IF EXISTS idx_guest_orders_session_id;

DROP INDEX IF EXISTS idx_guest_orders_created_at;

DROP INDEX IF EXISTS idx_guest_orders_order_reference;

DROP INDEX IF EXISTS idx_guest_orders_tenant_status;

-- Drop table
DROP TABLE IF EXISTS guest_orders CASCADE;