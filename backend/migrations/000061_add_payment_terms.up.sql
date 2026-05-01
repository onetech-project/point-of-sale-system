-- Migration: 000061_add_payment_terms.up.sql
-- Purpose: Create payment_terms table for offline order installment plans
-- Features: Down payment support, flexible installment schedules, balance tracking

CREATE TABLE IF NOT EXISTS payment_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES guest_orders(id) ON DELETE CASCADE,

-- Payment structure
total_amount INTEGER NOT NULL CHECK (total_amount > 0),
down_payment_amount INTEGER CHECK (
    down_payment_amount >= 0
    AND down_payment_amount < total_amount
),
installment_count INTEGER CHECK (installment_count >= 0),
installment_amount INTEGER CHECK (installment_amount >= 0),

-- Schedule
payment_schedule JSONB, -- Array of {installment_number, due_date, amount}

-- Status tracking
total_paid INTEGER NOT NULL DEFAULT 0 CHECK (
    total_paid >= 0
    AND total_paid <= total_amount
),
remaining_balance INTEGER NOT NULL CHECK (remaining_balance >= 0),

-- Metadata
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
created_by_user_id UUID NOT NULL REFERENCES users (id),

-- Constraints
CONSTRAINT check_payment_structure
    CHECK (
        (down_payment_amount IS NULL AND installment_count = 0) OR
        (down_payment_amount >= 0 AND installment_count > 0)
    ),

    CONSTRAINT check_remaining_balance
    CHECK (remaining_balance = total_amount - total_paid)
);

-- Indexes for query performance
CREATE INDEX idx_payment_terms_order_id ON payment_terms (order_id);

CREATE INDEX idx_payment_terms_balance ON payment_terms (remaining_balance, order_id)
WHERE
    remaining_balance > 0;

-- Column comments for documentation
COMMENT ON TABLE payment_terms IS 'Payment schedule definition for offline orders with installments';

COMMENT ON COLUMN payment_terms.payment_schedule IS 'JSONB array: [{"installment_number": 1, "due_date": "2026-03-07", "amount": 50000}, ...]';

COMMENT ON COLUMN payment_terms.total_amount IS 'Total order amount (must match guest_orders.total_amount)';

COMMENT ON COLUMN payment_terms.down_payment_amount IS 'Initial payment amount (if any)';

COMMENT ON COLUMN payment_terms.remaining_balance IS 'Computed: total_amount - total_paid';