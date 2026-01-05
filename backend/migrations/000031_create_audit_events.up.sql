-- Migration 000031: Create audit_events table (partitioned)
-- Purpose: Immutable audit trail for UU PDP compliance investigations
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID DEFAULT gen_random_uuid (),
    event_id VARCHAR(100) NOT NULL,
    tenant_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_type VARCHAR(20) NOT NULL,
    actor_id UUID,
    actor_email VARCHAR(255),
    session_id UUID,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(50),
    before_value JSONB,
    after_value JSONB,
    metadata JSONB,
    purpose VARCHAR(100),
    consent_id UUID,
    CONSTRAINT chk_actor_type CHECK (
        actor_type IN (
            'user',
            'system',
            'guest',
            'admin'
        )
    ),
    CONSTRAINT chk_action CHECK (
        action IN (
            'CREATE',
            'READ',
            'UPDATE',
            'DELETE',
            'ACCESS',
            'EXPORT',
            'ANONYMIZE'
        )
    )
)
PARTITION BY
    RANGE (timestamp);

-- Create initial partitions (current month + next month)
CREATE TABLE audit_events_2026_01 PARTITION OF audit_events FOR
VALUES
FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE audit_events_2026_02 PARTITION OF audit_events FOR
VALUES
FROM ('2026-02-01') TO ('2026-03-01');

-- Indexes on parent table (inherited by all partitions)
CREATE UNIQUE INDEX idx_audit_events_event_id ON audit_events (event_id, timestamp);

CREATE INDEX idx_audit_events_tenant_time ON audit_events (tenant_id, timestamp DESC);

CREATE INDEX idx_audit_events_actor_time ON audit_events (actor_id, timestamp DESC)
WHERE
    actor_id IS NOT NULL;

CREATE INDEX idx_audit_events_resource ON audit_events (
    resource_type,
    resource_id,
    timestamp DESC
);

CREATE INDEX idx_audit_events_action ON audit_events (
    tenant_id,
    action,
    timestamp DESC
);

CREATE INDEX idx_audit_events_metadata ON audit_events USING GIN (metadata jsonb_path_ops);

-- Immutability enforcement: Revoke UPDATE and DELETE
-- Note: This will be enforced in application code and via database triggers
-- REVOKE UPDATE, DELETE ON audit_events FROM PUBLIC;

-- Comments
COMMENT ON TABLE audit_events IS 'UU PDP No.27 Tahun 2022: Immutable audit trail (7-year retention), partitioned monthly';

COMMENT ON COLUMN audit_events.event_id IS 'Idempotency key for Kafka deduplication';

COMMENT ON COLUMN audit_events.before_value IS 'State before change (sensitive fields encrypted)';

COMMENT ON COLUMN audit_events.after_value IS 'State after change (sensitive fields encrypted)';