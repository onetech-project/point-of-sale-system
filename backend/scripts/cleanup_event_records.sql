-- Cleanup script for event_records retention
-- Deletes processed event records older than the configured retention window (30 days default)

DELETE FROM event_records
WHERE
    processed_at < now() - interval '30 days';