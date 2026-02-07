-- Migration 000059: Rollback - Remove LOGIN action from audit_events constraint

-- Drop existing constraint
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS chk_action;

-- Recreate constraint without LOGIN
ALTER TABLE audit_events
ADD CONSTRAINT chk_action CHECK (
    action IN (
        'CREATE',
        'READ',
        'UPDATE',
        'DELETE',
        'ACCESS',
        'EXPORT',
        'ANONYMIZE'
    )
);