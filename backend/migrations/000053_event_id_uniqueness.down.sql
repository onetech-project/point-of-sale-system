-- Migration 000053: Rollback event_id uniqueness constraints

ALTER TABLE audit_events
DROP CONSTRAINT IF EXISTS chk_event_id_not_empty;

ALTER TABLE audit_events
DROP CONSTRAINT IF EXISTS chk_event_id_uuid_format;