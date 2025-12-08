-- Drop indexes
DROP INDEX IF EXISTS idx_inventory_reservations_product_status;

DROP INDEX IF EXISTS idx_inventory_reservations_status_expires;

DROP INDEX IF EXISTS idx_inventory_reservations_expires_at;

DROP INDEX IF EXISTS idx_inventory_reservations_order_product;

-- Drop table
DROP TABLE IF EXISTS inventory_reservations CASCADE;