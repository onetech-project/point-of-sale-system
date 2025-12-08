-- Add customer_email column to guest_orders table
ALTER TABLE guest_orders ADD COLUMN customer_email VARCHAR(255);

-- Add index for email lookups
CREATE INDEX idx_guest_orders_customer_email ON guest_orders (customer_email)
WHERE
    customer_email IS NOT NULL;