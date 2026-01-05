-- Migration 000032: Create retention_policies table
-- Purpose: Define automated data retention rules per table/record type
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

CREATE TABLE IF NOT EXISTS retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    table_name VARCHAR(50) NOT NULL,
    record_type VARCHAR(50),
    retention_period_days INT NOT NULL,
    retention_field VARCHAR(50) NOT NULL,
    legal_minimum_days INT NOT NULL DEFAULT 0,
    cleanup_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_retention_period_positive CHECK (retention_period_days > 0),
    CONSTRAINT chk_retention_meets_legal CHECK (
        retention_period_days >= legal_minimum_days
    ),
    CONSTRAINT uq_retention_policy UNIQUE (table_name, record_type)
);

-- Index for cleanup job queries
CREATE INDEX idx_retention_policies_cleanup ON retention_policies (cleanup_enabled)
WHERE
    cleanup_enabled = TRUE;

-- Comments
COMMENT ON TABLE retention_policies IS 'UU PDP No.27 Tahun 2022: Data retention configuration with legal minimums';

COMMENT ON COLUMN retention_policies.retention_field IS 'Timestamp field to check (created_at, expired_at, deleted_at)';

COMMENT ON COLUMN retention_policies.legal_minimum_days IS 'Minimum retention required by law (e.g., 1825 for tax)';