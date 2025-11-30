-- Migration: 000005_create_password_reset_tokens
-- Description: Create password reset tokens table
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- Comments for documentation
COMMENT ON TABLE password_reset_tokens IS 'Password reset tokens for forgot password flow';
COMMENT ON COLUMN password_reset_tokens.token IS 'Unique reset token sent via email';
COMMENT ON COLUMN password_reset_tokens.expires_at IS 'Token expiration (typically 1 hour)';
COMMENT ON COLUMN password_reset_tokens.used_at IS 'When token was used (prevents reuse)';
