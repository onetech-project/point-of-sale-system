-- Create payment_transactions table
CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,

-- Midtrans details
midtrans_transaction_id VARCHAR(255) UNIQUE,
midtrans_order_id VARCHAR(255) NOT NULL,

-- Transaction details
amount INTEGER NOT NULL CHECK (amount > 0),
payment_type VARCHAR(50),
transaction_status VARCHAR(50),
fraud_status VARCHAR(50),

-- Notification
notification_payload JSONB,
signature_key VARCHAR(512),
signature_verified BOOLEAN NOT NULL DEFAULT false,

-- Timestamps
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
notification_received_at TIMESTAMP,
settled_at TIMESTAMP,

-- Idempotency
idempotency_key VARCHAR(255) UNIQUE );

-- Indexes
CREATE INDEX idx_payment_transactions_order_id ON payment_transactions (order_id);

CREATE INDEX idx_payment_transactions_midtrans_transaction_id ON payment_transactions (midtrans_transaction_id);

CREATE INDEX idx_payment_transactions_created_at ON payment_transactions (created_at DESC);

CREATE INDEX idx_payment_transactions_idempotency_key ON payment_transactions (idempotency_key);

-- Comments
COMMENT ON TABLE payment_transactions IS 'Midtrans payment tracking with full webhook audit trail';

COMMENT ON COLUMN payment_transactions.idempotency_key IS 'Prevents duplicate webhook processing: {midtrans_id}:{status}';

COMMENT ON COLUMN payment_transactions.signature_verified IS 'Must be true before processing webhook';

COMMENT ON COLUMN payment_transactions.notification_payload IS 'Full webhook payload for audit';