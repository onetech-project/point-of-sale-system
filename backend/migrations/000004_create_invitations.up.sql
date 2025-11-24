-- Migration: 000004_create_invitations
-- Description: Create invitations table for team member invites
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'manager', 'cashier')),
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'cancelled')),
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure one pending invitation per email per tenant
    CONSTRAINT unique_tenant_email_invitation UNIQUE(tenant_id, email, status)
);

-- Indexes for performance
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_email ON invitations(tenant_id, email);
CREATE INDEX idx_invitations_status ON invitations(status);
CREATE INDEX idx_invitations_expires_at ON invitations(expires_at);

-- Comments for documentation
COMMENT ON TABLE invitations IS 'Team member invitation tokens';
COMMENT ON COLUMN invitations.token IS 'Unique invitation token sent via email';
COMMENT ON COLUMN invitations.expires_at IS 'Invitation expiration (typically 7 days)';
COMMENT ON COLUMN invitations.invited_by IS 'User who sent the invitation';
