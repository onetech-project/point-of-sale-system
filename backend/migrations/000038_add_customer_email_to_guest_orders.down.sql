-- Remove customer_email column from guest_orders table
DROP INDEX IF EXISTS idx_guest_orders_customer_email;

ALTER TABLE guest_orders DROP COLUMN IF EXISTS customer_email;