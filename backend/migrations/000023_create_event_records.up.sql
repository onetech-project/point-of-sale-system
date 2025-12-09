-- Migration: Create event_records table for deduplication and audit
-- Path: backend/migrations/000023_create_event_records.up.sql

CREATE TABLE IF NOT EXISTS event_records (
    event_id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_event_records_tenant_processed_at ON event_records (tenant_id, processed_at);

-- Note: Retention policy (e.g., DELETE FROM event_records WHERE processed_at < now() - interval '30 days')
-- should be implemented as a scheduled job / partition drop depending on DB hosting.