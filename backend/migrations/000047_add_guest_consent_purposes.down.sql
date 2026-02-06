-- Migration 000047 down: Remove guest checkout consent purposes

DELETE FROM consent_purposes
WHERE
    purpose_code IN (
        'order_processing',
        'order_communications',
        'promotional_communications',
        'payment_processing_midtrans'
    );