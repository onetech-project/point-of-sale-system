-- Migration 000052: Enforce audit_events immutability (T113)
-- Purpose: Prevent UPDATE and DELETE operations on audit_events table for UU PDP compliance
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022 Article 56: 7-year retention)

-- Create trigger function to prevent UPDATE and DELETE
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'UPDATE operations on audit_events are not allowed. Audit logs are immutable.';
    ELSIF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'DELETE operations on audit_events are not allowed. Audit logs are immutable.';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to parent table (inherits to all partitions)
CREATE TRIGGER trg_audit_events_immutability
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

-- Revoke UPDATE and DELETE permissions from all roles
REVOKE UPDATE, DELETE ON audit_events FROM PUBLIC;

-- Add comment explaining immutability
COMMENT ON TRIGGER trg_audit_events_immutability ON audit_events IS 'UU PDP compliance: Prevents modification or deletion of audit logs. Audit trail must be immutable for investigations (Article 56).';