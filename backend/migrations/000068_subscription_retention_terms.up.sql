-- Migration: 000068_subscription_retention_terms
-- Purpose: Track subscription-retention windows and audited Terms of Service acceptance.

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS subscription_retention_started_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS subscription_data_anonymized_at TIMESTAMPTZ;

COMMENT ON COLUMN tenants.subscription_retention_started_at IS 'When the 30-day operational data retention timer started after entering grace period';
COMMENT ON COLUMN tenants.subscription_data_anonymized_at IS 'When expired tenant operational data was anonymized/deleted after subscription retention elapsed';

UPDATE tenants
SET subscription_retention_started_at = NOW()
WHERE subscription_status IN ('grace_period', 'expired')
  AND subscription_retention_started_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenants_subscription_retention_due
ON tenants (subscription_retention_started_at)
WHERE subscription_status IN ('grace_period', 'expired')
  AND subscription_data_anonymized_at IS NULL;

-- Preserve order financial history while allowing product catalog cleanup.
ALTER TABLE order_items
DROP CONSTRAINT IF EXISTS order_items_product_id_fkey;

ALTER TABLE order_items
ALTER COLUMN product_id DROP NOT NULL;

ALTER TABLE order_items
ADD CONSTRAINT order_items_product_id_fkey
FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS tenant_terms_acceptances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    terms_version VARCHAR(20) NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, terms_version)
);

CREATE INDEX IF NOT EXISTS idx_tenant_terms_acceptances_tenant
ON tenant_terms_acceptances (tenant_id, accepted_at DESC);

COMMENT ON TABLE tenant_terms_acceptances IS 'Audited Terms of Service acceptance records for tenant registration';
