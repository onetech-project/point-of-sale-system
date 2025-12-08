-- Drop indexes
DROP INDEX IF EXISTS idx_delivery_addresses_place_id;

DROP INDEX IF EXISTS idx_delivery_addresses_lat_lng;

DROP INDEX IF EXISTS idx_delivery_addresses_order_id;

-- Drop table
DROP TABLE IF EXISTS delivery_addresses CASCADE;