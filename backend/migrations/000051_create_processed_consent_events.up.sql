-- Create processed_consent_events table for idempotency tracking
-- Prevents duplicate consent record creation from replayed Kafka events

CREATE TABLE processed_consent_events (
    event_id VARCHAR(100) PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    subject_type VARCHAR(10) NOT NULL CHECK (
        subject_type IN ('tenant', 'guest')
    ),
    subject_id UUID NOT NULL
);

-- Index for querying processed events by tenant and time
CREATE INDEX idx_processed_events_tenant ON processed_consent_events (tenant_id, processed_at DESC);

-- Index for querying by subject
CREATE INDEX idx_processed_events_subject ON processed_consent_events (subject_type, subject_id);

-- Comment
COMMENT ON TABLE processed_consent_events IS 'Tracks processed consent events for idempotency - prevents duplicate consent records from Kafka event replays';

COMMENT ON COLUMN processed_consent_events.event_id IS 'UUID from ConsentGrantedEvent - primary key for idempotency';

COMMENT ON COLUMN processed_consent_events.processed_at IS 'Timestamp when event was successfully processed';

COMMENT ON COLUMN processed_consent_events.tenant_id IS 'Tenant ID from event for partitioning/cleanup';

COMMENT ON COLUMN processed_consent_events.subject_type IS 'tenant or guest - matches consent_records.subject_type';

COMMENT ON COLUMN processed_consent_events.subject_id IS 'user_id (tenant) or order_id (guest) - matches consent subject';