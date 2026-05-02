-- Migration: 000065_add_subscription_fields
-- Description: Add subscription management fields to tenants table
-- Supports: 7-day free trial, monthly/annual billing, storage quota

ALTER TABLE tenants
  ADD COLUMN IF NOT EXISTS subscription_plan VARCHAR(20) NOT NULL DEFAULT 'trial' 
    CHECK (subscription_plan IN ('trial', 'starter', 'professional', 'enterprise')),
  ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly'
    CHECK (billing_cycle IN ('monthly', 'annual')),
  ADD COLUMN IF NOT EXISTS trial_started_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS subscribed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS subscription_ends_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS storage_quota_bytes BIGINT NOT NULL DEFAULT 2147483648,
  ADD COLUMN IF NOT EXISTS storage_used_bytes BIGINT NOT NULL DEFAULT 0;

-- Backfill trial dates for existing tenants (trial started at account creation)
UPDATE tenants
SET
  trial_started_at = created_at,
  trial_ends_at = created_at + INTERVAL '7 days'
WHERE trial_started_at IS NULL;

-- Index for subscription queries
CREATE INDEX IF NOT EXISTS idx_tenants_subscription_plan ON tenants(subscription_plan);
CREATE INDEX IF NOT EXISTS idx_tenants_trial_ends_at ON tenants(trial_ends_at);
CREATE INDEX IF NOT EXISTS idx_tenants_subscription_ends_at ON tenants(subscription_ends_at);

COMMENT ON COLUMN tenants.subscription_plan IS 'Subscription tier: trial, starter, professional, enterprise';
COMMENT ON COLUMN tenants.billing_cycle IS 'Billing frequency: monthly or annual';
COMMENT ON COLUMN tenants.trial_started_at IS 'When the 7-day free trial started';
COMMENT ON COLUMN tenants.trial_ends_at IS 'When the 7-day free trial expires';
COMMENT ON COLUMN tenants.storage_quota_bytes IS 'Maximum storage allowed (default 2 GB = 2147483648 bytes)';
COMMENT ON COLUMN tenants.storage_used_bytes IS 'Currently used storage in bytes';
