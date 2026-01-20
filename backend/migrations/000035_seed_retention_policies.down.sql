-- Migration 000035: Remove seeded retention policies
DELETE FROM retention_policies
WHERE
    table_name IN (
        'guest_orders',
        'audit_events',
        'users',
        'verification_tokens',
        'password_reset_tokens',
        'invitations',
        'sessions'
    );