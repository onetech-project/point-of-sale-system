-- Migration: 000066_add_billing_tables
-- Description: Add subscription_status to tenants, billing invoices, and payment attempts tables

-- Add subscription_status column to tenants for fast enforcement queries
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20) NOT NULL DEFAULT 'trial' CHECK (
    subscription_status IN (
        'trial',
        'active',
        'grace_period',
        'expired',
        'cancelled'
    )
);

CREATE INDEX IF NOT EXISTS idx_tenants_subscription_status ON tenants (subscription_status);

-- Backfill subscription_status based on existing data
UPDATE tenants
SET
    subscription_status = CASE
        WHEN subscription_plan = 'trial'
        AND (
            trial_ends_at IS NULL
            OR trial_ends_at > NOW()
        ) THEN 'trial'
        WHEN subscription_plan = 'trial'
        AND trial_ends_at < NOW()
        AND trial_ends_at > NOW() - INTERVAL '7 days' THEN 'grace_period'
        WHEN subscription_plan = 'trial'
        AND trial_ends_at <= NOW() - INTERVAL '7 days' THEN 'expired'
        WHEN subscription_plan != 'trial'
        AND (
            subscription_ends_at IS NULL
            OR subscription_ends_at > NOW()
        ) THEN 'active'
        WHEN subscription_plan != 'trial'
        AND subscription_ends_at < NOW()
        AND subscription_ends_at > NOW() - INTERVAL '7 days' THEN 'grace_period'
        WHEN subscription_plan != 'trial'
        AND subscription_ends_at <= NOW() - INTERVAL '7 days' THEN 'expired'
        ELSE 'trial'
    END
WHERE
    subscription_status = 'trial';

-- Billing invoices table
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL,
    invoice_number VARCHAR(30) NOT NULL UNIQUE,
    amount_idr INTEGER NOT NULL,
    billing_interval VARCHAR(10) NOT NULL CHECK (
        billing_interval IN ('monthly', 'annual')
    ),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (
        status IN (
            'pending',
            'paid',
            'expired',
            'cancelled'
        )
    ),
    paid_at TIMESTAMPTZ,
    midtrans_order_id VARCHAR(100),
    midtrans_payment_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_tenant_id ON billing_invoices (tenant_id);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_status ON billing_invoices (status);

-- Billing payment attempts
CREATE TABLE IF NOT EXISTS billing_payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    invoice_id UUID NOT NULL REFERENCES billing_invoices (id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    amount_idr INTEGER NOT NULL,
    midtrans_order_id VARCHAR(100),
    payment_method VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (
        status IN (
            'pending',
            'completed',
            'failed'
        )
    ),
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_payment_attempts_invoice_id ON billing_payment_attempts (invoice_id);