-- Migration 000029: Create privacy_policies table
-- Purpose: Track versioned privacy policy text with effective dates
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

CREATE TABLE IF NOT EXISTS privacy_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    version VARCHAR(20) UNIQUE NOT NULL,
    policy_text_id TEXT NOT NULL,
    policy_text_en TEXT,
    effective_date TIMESTAMPTZ NOT NULL,
    change_summary_id TEXT,
    change_summary_en TEXT,
    is_major_update BOOLEAN NOT NULL DEFAULT FALSE,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_version_format CHECK (version ~ '^\d+\.\d+\.\d+$'),
    CONSTRAINT chk_effective_date_not_future CHECK (
        (is_current = FALSE)
        OR (effective_date <= NOW())
    )
);

-- Indexes
CREATE UNIQUE INDEX idx_privacy_policies_version ON privacy_policies (version);

CREATE UNIQUE INDEX idx_privacy_policies_current ON privacy_policies (is_current)
WHERE
    is_current = TRUE;

CREATE INDEX idx_privacy_policies_effective_date ON privacy_policies (effective_date DESC);

-- Comments
COMMENT ON TABLE privacy_policies IS 'UU PDP No.27 Tahun 2022: Versioned privacy policy for consent management';

COMMENT ON COLUMN privacy_policies.policy_text_id IS 'Indonesian policy text (legally binding)';

COMMENT ON COLUMN privacy_policies.is_major_update IS 'Material changes require re-consent';

COMMENT ON CONSTRAINT chk_effective_date_not_future ON privacy_policies IS 'Current policy cannot have future effective date';