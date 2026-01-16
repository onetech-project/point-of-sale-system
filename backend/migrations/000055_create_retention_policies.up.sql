-- Create retention_policies table for automated data retention management
-- UU PDP compliance: data minimization principle (Article 11)

CREATE TABLE IF NOT EXISTS retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(50) NOT NULL,
    record_type VARCHAR(50), -- Optional subtype (e.g., 'verification_token', 'completed_order')
    retention_period_days INTEGER NOT NULL CHECK (retention_period_days > 0),
    retention_field VARCHAR(50) NOT NULL, -- Timestamp field to check ('created_at', 'expired_at', 'deleted_at')
    grace_period_days INTEGER, -- Soft delete grace period before hard delete
    legal_minimum_days INTEGER, -- Minimum retention by law (5 years = 1825, 7 years = 2555)
    cleanup_method VARCHAR(20) NOT NULL CHECK (cleanup_method IN ('soft_delete', 'hard_delete', 'anonymize')),
    notification_days_before INTEGER, -- Send notification N days before deletion
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

-- Ensure retention period meets legal minimum
CONSTRAINT check_legal_minimum CHECK (
    legal_minimum_days IS NULL
    OR retention_period_days >= legal_minimum_days
),

-- One policy per table/type combination
CONSTRAINT unique_table_record_type UNIQUE (table_name, record_type)
);

-- Indexes for efficient policy queries
CREATE INDEX idx_retention_policies_active ON retention_policies (table_name, is_active)
WHERE
    is_active = TRUE;

CREATE INDEX idx_retention_policies_period ON retention_policies (retention_period_days);

-- Insert default retention policies
INSERT INTO
    retention_policies (
        table_name,
        record_type,
        retention_period_days,
        retention_field,
        grace_period_days,
        legal_minimum_days,
        cleanup_method,
        notification_days_before
    )
VALUES
    -- Verification tokens: 48 hours after expiration
    (
        'email_verification_tokens',
        NULL,
        2,
        'expired_at',
        NULL,
        NULL,
        'hard_delete',
        NULL
    ),

-- Password reset tokens: 24 hours after consumption
(
    'password_reset_tokens',
    'consumed',
    1,
    'consumed_at',
    NULL,
    NULL,
    'hard_delete',
    NULL
),

-- Expired invitations: 30 days after expiration
(
    'user_invitations',
    'expired',
    30,
    'expired_at',
    NULL,
    NULL,
    'hard_delete',
    NULL
),

-- Expired sessions: 7 days after expiration
(
    'user_sessions',
    NULL,
    7,
    'expired_at',
    NULL,
    NULL,
    'hard_delete',
    NULL
),

-- Deleted users: 90 days grace period after soft delete
( 'users', 'deleted', 90, 'deleted_at', 90, NULL, 'hard_delete', 30 ),

-- Guest orders: 5 years (Indonesian tax law) after completion
(
    'guest_orders',
    'completed',
    1825,
    'created_at',
    NULL,
    1825,
    'hard_delete',
    NULL
),

-- Audit events: 7 years (Indonesian record retention law)
(
    'audit_events',
    NULL,
    2555,
    'timestamp',
    NULL,
    2555,
    'hard_delete',
    NULL
);

-- Comments for documentation
COMMENT ON TABLE retention_policies IS 'Automated data retention rules per UU PDP Article 11 (data minimization)';

COMMENT ON COLUMN retention_policies.legal_minimum_days IS 'Minimum retention by law: 1825 days (5 years) for tax, 2555 days (7 years) for audit';

COMMENT ON COLUMN retention_policies.cleanup_method IS 'soft_delete: mark as deleted | hard_delete: permanent removal | anonymize: remove PII only';