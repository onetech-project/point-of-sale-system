-- Migration: 000070_platform_command_center
-- Description: Add platform-owner command center audit support and query indexes

ALTER TABLE tenant_lifecycle_events
DROP CONSTRAINT IF EXISTS tenant_lifecycle_events_action_check;

ALTER TABLE tenant_lifecycle_events
ADD CONSTRAINT tenant_lifecycle_events_action_check CHECK (
    action IN (
        'suspend',
        'reactivate',
        'deactivate',
        'schedule_delete',
        'cancel_delete',
        'delete',
        'change_plan',
        'change_billing_cycle',
        'extend_trial',
        'extend_grace',
        'regenerate_payment_link',
        'add_note',
        'assign_support_owner'
    )
);

CREATE TABLE IF NOT EXISTS platform_tenant_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    platform_admin_id UUID REFERENCES platform_admins (id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_tenant_notes_tenant_id
    ON platform_tenant_notes (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tenants_status_subscription_status
    ON tenants (status, subscription_status);

CREATE INDEX IF NOT EXISTS idx_tenants_subscription_plan_cycle
    ON tenants (subscription_plan, billing_cycle);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_status_paid_at
    ON billing_invoices (status, paid_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_status_due_at
    ON billing_invoices (status, due_at ASC);

CREATE INDEX IF NOT EXISTS idx_users_tenant_last_login
    ON users (tenant_id, last_login_at DESC);
