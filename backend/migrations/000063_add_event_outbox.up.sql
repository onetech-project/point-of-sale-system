-- Migration: 000063_add_event_outbox.up.sql
-- Purpose: Create event_outbox table for transactional outbox pattern
-- Features: Reliable event publishing, retry mechanism, ordered event processing

CREATE TABLE IF NOT EXISTS event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_type VARCHAR(100) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT
);

-- Index for efficient polling of unpublished events
CREATE INDEX idx_outbox_pending ON event_outbox (created_at)
WHERE
    published_at IS NULL;

-- Column comments for documentation
COMMENT ON TABLE event_outbox IS 'Transactional outbox for reliable Kafka event publishing';

COMMENT ON COLUMN event_outbox.event_type IS 'Event type identifier (e.g., offline_order.created)';

COMMENT ON COLUMN event_outbox.event_key IS 'Kafka partition key (e.g., order_id for ordering)';

COMMENT ON COLUMN event_outbox.event_payload IS 'Full event payload as JSON';

COMMENT ON COLUMN event_outbox.topic IS 'Target Kafka topic (e.g., offline-orders-audit)';

COMMENT ON COLUMN event_outbox.published_at IS 'Timestamp when successfully published to Kafka (NULL = pending)';