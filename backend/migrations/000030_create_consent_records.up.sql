-- Migration 000030: Create consent_records table
-- Purpose: Track individual consent grants and revocations with full metadata
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    subject_type VARCHAR(10) NOT NULL,
    subject_id UUID REFERENCES users (id) ON DELETE CASCADE,
    guest_order_id UUID REFERENCES guest_orders (id) ON DELETE SET NULL,
    purpose_id UUID NOT NULL REFERENCES consent_purposes (id),
    granted BOOLEAN NOT NULL,
    policy_version VARCHAR(20) NOT NULL REFERENCES privacy_policies (version),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    ip_address INET NOT NULL,
    user_agent TEXT NOT NULL,
    session_id UUID,
    consent_method VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_subject_type CHECK (
        subject_type IN ('tenant', 'guest')
    ),
    CONSTRAINT chk_consent_method CHECK (
        consent_method IN (
            'registration',
            'checkout',
            'settings_update',
            'api'
        )
    ),
    CONSTRAINT chk_revoked_after_granted CHECK (
        revoked_at IS NULL
        OR revoked_at >= granted_at
    ),
    CONSTRAINT chk_subject_identity CHECK (
        (
            subject_type = 'tenant'
            AND subject_id IS NOT NULL
            AND guest_order_id IS NULL
        )
        OR (
            subject_type = 'guest'
            AND subject_id IS NULL
            AND guest_order_id IS NOT NULL
        )
    )
);

-- Indexes for efficient consent lookups
CREATE INDEX idx_consent_records_active_tenant ON consent_records (
    subject_type,
    subject_id,
    purpose_id,
    revoked_at
)
WHERE
    revoked_at IS NULL;

CREATE INDEX idx_consent_records_active_guest ON consent_records (
    guest_order_id,
    purpose_id,
    revoked_at
)
WHERE
    revoked_at IS NULL;

CREATE INDEX idx_consent_records_tenant_history ON consent_records (tenant_id, granted_at DESC);

CREATE INDEX idx_consent_records_purpose_analytics ON consent_records (purpose_id, granted_at DESC);

-- Comments
COMMENT ON TABLE consent_records IS 'UU PDP No.27 Tahun 2022: Individual consent tracking with full legal proof metadata';

COMMENT ON COLUMN consent_records.ip_address IS 'Legal proof of consent action per UU PDP Article 6';

COMMENT ON COLUMN consent_records.revoked_at IS 'NULL = active consent, timestamp = revoked';