-- Migration: 000001_create_tenants
-- Description: Create tenants table for multi-tenancy support
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE tenants IS 'Multi-tenant organizations (businesses using the POS system)';
COMMENT ON COLUMN tenants.slug IS 'URL-friendly unique identifier for tenant';
COMMENT ON COLUMN tenants.status IS 'Tenant account status: active, suspended, deleted, or inactive';
