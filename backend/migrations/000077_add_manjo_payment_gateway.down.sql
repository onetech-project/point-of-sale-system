DROP TABLE IF EXISTS payment_refunds;

ALTER TABLE payment_transactions
DROP COLUMN IF EXISTS gateway_external_id,
DROP COLUMN IF EXISTS gateway_reference_no,
DROP COLUMN IF EXISTS payment_gateway;

COMMENT ON COLUMN payment_transactions.midtrans_transaction_id IS NULL;
COMMENT ON COLUMN payment_transactions.midtrans_order_id IS NULL;

ALTER TABLE tenant_configs
DROP COLUMN IF EXISTS manjo_environment,
DROP COLUMN IF EXISTS manjo_partner_id,
DROP COLUMN IF EXISTS manjo_merchant_id,
DROP COLUMN IF EXISTS manjo_private_key,
DROP COLUMN IF EXISTS manjo_client_secret,
DROP COLUMN IF EXISTS manjo_client_id,
DROP COLUMN IF EXISTS payment_gateway;
