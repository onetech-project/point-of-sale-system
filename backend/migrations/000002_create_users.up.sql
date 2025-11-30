-- Migration: 000002_create_users
-- Description: Create users table with email verification support
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    locale VARCHAR(10) NOT NULL DEFAULT 'en',
    role VARCHAR(20) NOT NULL DEFAULT 'cashier' CHECK (role IN ('owner', 'manager', 'cashier')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    
    -- Email verification fields (T334)
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    verification_token VARCHAR(255),
    verification_token_expires_at TIMESTAMPTZ,
    
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure email is unique per tenant
    CONSTRAINT unique_tenant_email UNIQUE(tenant_id, email)
);

-- Indexes for performance and tenant isolation
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_verification_token ON users(verification_token) 
WHERE verification_token IS NOT NULL;
CREATE INDEX idx_users_email_verified ON users(email_verified, created_at);

-- Comments for documentation
COMMENT ON TABLE users IS 'User accounts with role-based access control';
COMMENT ON COLUMN users.tenant_id IS 'Foreign key to tenant - ensures data isolation';
COMMENT ON COLUMN users.role IS 'User role: owner (full access), manager (limited), cashier (POS only)';
COMMENT ON COLUMN users.email_verified IS 'Whether user has verified their email address';
COMMENT ON COLUMN users.verification_token IS 'Token for email verification (expires in 24h)';
