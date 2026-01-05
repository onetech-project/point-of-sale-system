-- Migration: 000037_add_guest_orders_anonymization_flag
-- Purpose: Add anonymization tracking for guest order data deletion (UU PDP compliance)
--
-- Adds columns to track when guest customer PII has been anonymized per data deletion requests.
-- The existing customer_name, customer_phone, customer_email, ip_address columns will store
-- encrypted values (vault:v1:...) for active orders, and will be set to NULL/generic values
-- when anonymized.

ALTER TABLE guest_orders
ADD COLUMN is_anonymized BOOLEAN DEFAULT FALSE NOT NULL,
ADD COLUMN anonymized_at TIMESTAMP NULL;

COMMENT ON COLUMN guest_orders.is_anonymized IS 'TRUE when customer PII has been anonymized per deletion request';

COMMENT ON COLUMN guest_orders.anonymized_at IS 'Timestamp when guest data was anonymized (NULL if not anonymized)';

-- Create index for querying anonymized orders
CREATE INDEX idx_guest_orders_anonymized ON guest_orders (is_anonymized)
WHERE
    is_anonymized = TRUE;