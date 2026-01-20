-- Migration 000035: Seed retention_policies with default policies
-- Purpose: Tax records (5y), audit logs (7y), temporary data (48h)
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

INSERT INTO
    retention_policies (
        table_name,
        record_type,
        retention_period_days,
        retention_field,
        legal_minimum_days,
        description
    )
VALUES (
        'guest_orders',
        'completed_order',
        1825, -- 5 years
        'created_at',
        1825, -- Indonesian tax law requirement
        'Completed guest orders retained for 5 years per Indonesian tax regulations'
    ),
    (
        'audit_events',
        NULL,
        2555, -- 7 years
        'timestamp',
        2555, -- Indonesian compliance standard
        'Audit trail retained for 7 years per UU PDP compliance requirements'
    ),
    (
        'users',
        'deleted_user',
        90, -- 90 days
        'deleted_at',
        0, -- No legal minimum for deleted users
        'Soft-deleted users retained for 90 days before permanent deletion'
    ),
    (
        'verification_tokens',
        NULL,
        2, -- 48 hours
        'created_at',
        0,
        'Email verification tokens expire after 48 hours'
    ),
    (
        'password_reset_tokens',
        'consumed',
        1, -- 24 hours
        'consumed_at',
        0,
        'Used password reset tokens deleted after 24 hours'
    ),
    (
        'invitations',
        'expired',
        30, -- 30 days
        'expired_at',
        0,
        'Expired invitations deleted after 30 days'
    ),
    (
        'sessions',
        'expired',
        7, -- 7 days
        'expired_at',
        0,
        'Expired sessions deleted after 7 days'
    )
ON CONFLICT (table_name, record_type) DO NOTHING;

-- Comments
COMMENT ON TABLE retention_policies IS 'Seeded with 7 default retention policies: guest orders (5y), audit events (7y), deleted users (90d), temp tokens (48h-30d)';