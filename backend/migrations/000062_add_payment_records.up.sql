-- Migration: 000062_add_payment_records.up.sql
-- Purpose: Create payment_records table for tracking payment transactions
-- Features: Payment history log, multiple payment methods, balance tracking

CREATE TABLE IF NOT EXISTS payment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL,

-- Payment details
payment_number INTEGER NOT NULL, -- 0 for down payment, 1+ for installments
amount_paid INTEGER NOT NULL CHECK (amount_paid > 0),
payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
payment_method VARCHAR(50) NOT NULL,

-- Balance tracking
remaining_balance_after INTEGER NOT NULL CHECK (remaining_balance_after >= 0),

-- Metadata
recorded_by_user_id UUID NOT NULL REFERENCES users (id),
notes TEXT,
receipt_number VARCHAR(100),

-- Timestamps
created_at TIMESTAMP NOT NULL DEFAULT NOW(),

-- Constraints
CONSTRAINT check_payment_method
    CHECK (payment_method IN ('cash', 'card', 'bank_transfer', 'check', 'other'))
);

-- Indexes for query performance
CREATE INDEX idx_payment_records_order_id ON payment_records (order_id, payment_date DESC);

CREATE INDEX idx_payment_records_date ON payment_records (payment_date DESC);

CREATE INDEX idx_payment_records_recorded_by ON payment_records (recorded_by_user_id);

-- Column comments for documentation
COMMENT ON TABLE payment_records IS 'Transaction log of payments received for offline orders';

COMMENT ON COLUMN payment_records.payment_number IS '0 = down payment, 1+ = installment number';

COMMENT ON COLUMN payment_records.remaining_balance_after IS 'Outstanding balance after this payment applied';

COMMENT ON COLUMN payment_records.payment_method IS 'Payment type: cash, card, bank_transfer, check, or other';