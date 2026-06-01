CREATE TABLE user_onboarding_progress (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tour_key VARCHAR(64) NOT NULL,
  tour_version INTEGER NOT NULL,
  role VARCHAR(32) NOT NULL,
  completed_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT user_onboarding_progress_role_check CHECK (role IN ('owner', 'manager', 'cashier')),
  CONSTRAINT user_onboarding_progress_tour_version_check CHECK (tour_version > 0),
  CONSTRAINT user_onboarding_progress_tour_key_check CHECK (
    tour_key IN ('owner-onboarding', 'manager-onboarding', 'cashier-onboarding')
  ),
  CONSTRAINT user_onboarding_progress_user_tour_version_unique UNIQUE (user_id, tour_key, tour_version)
);

CREATE INDEX idx_user_onboarding_progress_tenant_user ON user_onboarding_progress(tenant_id, user_id);
CREATE INDEX idx_user_onboarding_progress_tour ON user_onboarding_progress(tour_key, tour_version);
CREATE INDEX idx_user_onboarding_progress_completed_at ON user_onboarding_progress(completed_at DESC);
