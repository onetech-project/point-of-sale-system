-- Drop indexes
DROP INDEX IF EXISTS idx_payment_transactions_idempotency_key;

DROP INDEX IF EXISTS idx_payment_transactions_created_at;

DROP INDEX IF EXISTS idx_payment_transactions_midtrans_transaction_id;

DROP INDEX IF EXISTS idx_payment_transactions_order_id;

-- Drop table
DROP TABLE IF EXISTS payment_transactions CASCADE;