DROP INDEX IF EXISTS idx_tenants_subscription_ends_at;
DROP INDEX IF EXISTS idx_tenants_trial_ends_at;
DROP INDEX IF EXISTS idx_tenants_subscription_plan;

ALTER TABLE tenants
  DROP COLUMN IF EXISTS subscription_plan,
  DROP COLUMN IF EXISTS billing_cycle,
  DROP COLUMN IF EXISTS trial_started_at,
  DROP COLUMN IF EXISTS trial_ends_at,
  DROP COLUMN IF EXISTS subscribed_at,
  DROP COLUMN IF EXISTS subscription_ends_at;
