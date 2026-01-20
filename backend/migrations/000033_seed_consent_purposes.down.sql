-- Migration 000033: Remove seeded consent purposes
DELETE FROM consent_purposes
WHERE
    purpose_code IN (
        'operational',
        'analytics',
        'advertising',
        'third_party_midtrans'
    );