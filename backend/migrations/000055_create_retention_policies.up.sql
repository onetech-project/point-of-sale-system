-- Extend retention_policies table with additional columns for enhanced data retention management
-- UU PDP compliance: data minimization principle (Article 11)
-- This migration adds columns to the existing table created in migration 000032

-- Add new columns if they don't exist
DO $$ 
BEGIN
    -- Add grace_period_days column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'retention_policies' AND column_name = 'grace_period_days') THEN
        ALTER TABLE retention_policies ADD COLUMN grace_period_days INTEGER;
    END IF;

    -- Add cleanup_method column (default to 'hard_delete' to match existing cleanup_enabled behavior)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'retention_policies' AND column_name = 'cleanup_method') THEN
        ALTER TABLE retention_policies ADD COLUMN cleanup_method VARCHAR(20) DEFAULT 'hard_delete' 
            CHECK (cleanup_method IN ('soft_delete', 'hard_delete', 'anonymize'));
    END IF;

    -- Add notification_days_before column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'retention_policies' AND column_name = 'notification_days_before') THEN
        ALTER TABLE retention_policies ADD COLUMN notification_days_before INTEGER;
    END IF;

    -- Add is_active column (maps to existing cleanup_enabled)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'retention_policies' AND column_name = 'is_active') THEN
        ALTER TABLE retention_policies ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
        -- Copy values from cleanup_enabled to is_active
        UPDATE retention_policies SET is_active = cleanup_enabled;
    END IF;
END $$;

-- Indexes for efficient policy queries
CREATE INDEX IF NOT EXISTS idx_retention_policies_active ON retention_policies (table_name, is_active)
WHERE
    is_active = TRUE;

-- Update existing retention policies with new column values
UPDATE retention_policies
SET
    cleanup_method = 'hard_delete',
    is_active = cleanup_enabled
WHERE
    cleanup_method IS NULL;

-- Update specific policies with grace periods and notification settings
UPDATE retention_policies
SET
    grace_period_days = 90,
    notification_days_before = 30
WHERE
    table_name = 'users'
    AND record_type = 'deleted_user';

-- Comments for documentation
COMMENT ON COLUMN retention_policies.grace_period_days IS 'Soft delete grace period before hard delete (e.g., 90 days for deleted users)';

COMMENT ON COLUMN retention_policies.cleanup_method IS 'soft_delete: mark as deleted | hard_delete: permanent removal | anonymize: remove PII only';

COMMENT ON COLUMN retention_policies.notification_days_before IS 'Send notification N days before deletion (e.g., 30 days notice for user data deletion)';

COMMENT ON COLUMN retention_policies.is_active IS 'Whether this retention policy is currently active and should be enforced';