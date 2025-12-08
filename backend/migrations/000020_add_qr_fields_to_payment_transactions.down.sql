-- Remove QR code fields from payment_transactions table
ALTER TABLE payment_transactions
DROP COLUMN IF EXISTS qr_code_url,
DROP COLUMN IF EXISTS qr_string,
DROP COLUMN IF EXISTS expiry_time;