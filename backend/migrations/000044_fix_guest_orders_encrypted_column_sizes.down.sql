-- Rollback: Revert guest_orders column sizes
-- Warning: This will fail if any encrypted data exceeds the old limits

ALTER TABLE guest_orders
ALTER COLUMN customer_phone TYPE VARCHAR(100);

ALTER TABLE guest_orders ALTER COLUMN ip_address TYPE VARCHAR(100);

ALTER TABLE guest_orders ALTER COLUMN user_agent TYPE VARCHAR(512);