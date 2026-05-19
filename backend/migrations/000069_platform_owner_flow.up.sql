-- Migration: 000069_platform_owner_flow
-- Description: Extend billing invoices and add platform-owner administration tables

ALTER TABLE billing_invoices
ADD COLUMN IF NOT EXISTS due_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS payment_link_expires_at TIMESTAMPTZ;

UPDATE billing_invoices
SET due_at = COALESCE(due_at, period_start),
    payment_link_expires_at = COALESCE(payment_link_expires_at, created_at + INTERVAL '24 hours')
WHERE due_at IS NULL
   OR payment_link_expires_at IS NULL;

ALTER TABLE billing_invoices
ALTER COLUMN due_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_invoices_due_at ON billing_invoices (due_at);

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS deletion_requested_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS scheduled_delete_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS status_reason TEXT;

CREATE TABLE IF NOT EXISTS platform_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(120) NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'platform_owner' CHECK (
        role IN ('platform_owner', 'platform_operator')
    ),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (
        status IN ('active', 'suspended')
    ),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_admins_status ON platform_admins (status);

CREATE TABLE IF NOT EXISTS platform_admin_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(500) NOT NULL UNIQUE,
    platform_admin_id UUID NOT NULL REFERENCES platform_admins (id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    terminated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_admin_sessions_session_id ON platform_admin_sessions (session_id);
CREATE INDEX IF NOT EXISTS idx_platform_admin_sessions_admin_id ON platform_admin_sessions (platform_admin_id);
CREATE INDEX IF NOT EXISTS idx_platform_admin_sessions_expires_at ON platform_admin_sessions (expires_at);

CREATE TABLE IF NOT EXISTS support_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    subject VARCHAR(180) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(30) NOT NULL DEFAULT 'billing' CHECK (
        category IN ('billing', 'technical', 'account', 'other')
    ),
    priority VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (
        priority IN ('low', 'normal', 'high', 'urgent')
    ),
    status VARCHAR(30) NOT NULL DEFAULT 'open' CHECK (
        status IN ('open', 'in_progress', 'waiting_on_tenant', 'resolved', 'closed')
    ),
    created_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    created_by_email VARCHAR(255),
    assigned_to_platform_admin_id UUID REFERENCES platform_admins (id) ON DELETE SET NULL,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_tenant_id ON support_tickets (tenant_id);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets (status, priority, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_support_tickets_assignee ON support_tickets (assigned_to_platform_admin_id);

CREATE TABLE IF NOT EXISTS support_ticket_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES support_tickets (id) ON DELETE CASCADE,
    author_type VARCHAR(20) NOT NULL CHECK (
        author_type IN ('tenant_user', 'platform_admin', 'system')
    ),
    author_id UUID,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_notes_ticket_id ON support_ticket_notes (ticket_id, created_at ASC);

CREATE TABLE IF NOT EXISTS tenant_lifecycle_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    platform_admin_id UUID REFERENCES platform_admins (id) ON DELETE SET NULL,
    action VARCHAR(30) NOT NULL CHECK (
        action IN ('suspend', 'reactivate', 'deactivate', 'schedule_delete', 'cancel_delete', 'delete')
    ),
    reason TEXT,
    before_status VARCHAR(20),
    after_status VARCHAR(20),
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delete_after TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_lifecycle_events_tenant_id ON tenant_lifecycle_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_lifecycle_events_action ON tenant_lifecycle_events (action, created_at DESC);
