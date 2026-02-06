-- Migration 000028: Create consent_purposes table
-- Purpose: Define reusable consent types with required/optional flags
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

CREATE TABLE IF NOT EXISTS consent_purposes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    purpose_code VARCHAR(50) UNIQUE NOT NULL,
    purpose_name_en VARCHAR(100) NOT NULL,
    purpose_name_id VARCHAR(100) NOT NULL,
    description_en TEXT NOT NULL,
    description_id TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_purpose_code_format CHECK (purpose_code ~ '^[a-z_]+$'),
    CONSTRAINT chk_display_order_positive CHECK (display_order > 0)
);

-- Indexes for efficient querying
CREATE UNIQUE INDEX idx_consent_purposes_code ON consent_purposes (purpose_code);

CREATE INDEX idx_consent_purposes_display ON consent_purposes (is_required, display_order);

-- Comments for documentation
COMMENT ON TABLE consent_purposes IS 'UU PDP No.27 Tahun 2022: Consent purpose definitions for data processing transparency';

COMMENT ON COLUMN consent_purposes.purpose_name_id IS 'Indonesian display name (primary, legally binding)';

COMMENT ON COLUMN consent_purposes.is_required IS 'Mandatory consents cannot be revoked while account/order is active';