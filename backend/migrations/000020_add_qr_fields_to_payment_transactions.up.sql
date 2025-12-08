-- Add QR code fields to payment_transactions table
ALTER TABLE payment_transactions
ADD COLUMN qr_code_url TEXT,
ADD COLUMN qr_string TEXT,
ADD COLUMN expiry_time TIMESTAMP;

COMMENT ON COLUMN payment_transactions.qr_code_url IS 'URL to QRIS QR code image from Midtrans actions array';

COMMENT ON COLUMN payment_transactions.qr_string IS 'Raw QRIS string data for generating QR code';

COMMENT ON COLUMN payment_transactions.expiry_time IS 'When the QRIS payment expires (default 15 minutes)';