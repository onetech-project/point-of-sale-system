-- Migration: 000003_create_sessions
-- Description: Create sessions table for authentication
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(500) NOT NULL UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    terminated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Comments for documentation
COMMENT ON TABLE sessions IS 'User authentication sessions';
COMMENT ON COLUMN sessions.session_id IS 'JWT session identifier';
COMMENT ON COLUMN sessions.expires_at IS 'Session expiration time (typically 15 minutes)';
COMMENT ON COLUMN sessions.terminated_at IS 'Timestamp when session was explicitly terminated';
COMMENT ON COLUMN sessions.ip_address IS 'IP address where session was created (security audit)';
