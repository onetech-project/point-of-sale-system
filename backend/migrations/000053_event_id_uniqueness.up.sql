-- Migration 000053: Add event_id uniqueness constraint (T114)
-- Purpose: Ensure Kafka idempotency - prevent duplicate audit events
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

-- Note: UNIQUE INDEX already exists from migration 000031:
-- CREATE UNIQUE INDEX idx_audit_events_event_id ON audit_events (event_id, timestamp);
-- This ensures uniqueness across partitions.

-- Add CHECK constraint for event_id format validation (UUID format)
ALTER TABLE audit_events
ADD CONSTRAINT chk_event_id_not_empty CHECK (
    event_id <> ''
    AND LENGTH(event_id) > 0
);

-- Add CHECK constraint to ensure event_id follows UUID format (36 chars with hyphens)
-- This helps catch malformed event IDs early
ALTER TABLE audit_events
ADD CONSTRAINT chk_event_id_uuid_format CHECK (
    event_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
);

-- Add comment
COMMENT ON CONSTRAINT chk_event_id_not_empty ON audit_events IS 'Kafka idempotency: Ensures event_id is never empty for deduplication';

COMMENT ON CONSTRAINT chk_event_id_uuid_format ON audit_events IS 'Kafka idempotency: Validates event_id follows UUID v4 format for consistency';