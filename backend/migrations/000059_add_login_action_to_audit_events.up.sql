-- Migration 000059: Add LOGIN action to audit_events constraint
-- Purpose: Support LOGIN action for authentication audit events

-- Drop existing constraint
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_action;

-- Recreate constraint with LOGIN added
ALTER TABLE audit_events
ADD CONSTRAINT chk_action CHECK (
    action IN (
        'CREATE',
        'READ',
        'UPDATE',
        'DELETE',
        'ACCESS',
        'EXPORT',
        'ANONYMIZE',
        'LOGIN'
    )
);

COMMENT ON CONSTRAINT chk_action ON audit_events IS 'Valid audit actions including authentication events';