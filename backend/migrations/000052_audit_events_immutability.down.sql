-- Migration 000052: Rollback audit_events immutability enforcement
-- Purpose: Remove trigger and restore UPDATE/DELETE permissions (for testing/development only)

-- Drop trigger
DROP TRIGGER IF EXISTS trg_audit_events_immutability ON audit_events;

-- Drop trigger function
DROP FUNCTION IF EXISTS prevent_audit_modification ();

-- Restore permissions (if needed for testing)
-- GRANT UPDATE, DELETE ON audit_events TO <role_name>;