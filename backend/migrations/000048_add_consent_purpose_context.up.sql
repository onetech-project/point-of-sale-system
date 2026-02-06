-- Migration 000048: Add context field to consent_purposes
-- Purpose: Filter consent purposes by context (tenant registration vs guest checkout)
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

-- Add context field to consent_purposes table
ALTER TABLE consent_purposes
ADD COLUMN context VARCHAR(20) NOT NULL DEFAULT 'tenant' CHECK (
    context IN ('tenant', 'guest')
);

-- Update existing tenant-specific purposes
UPDATE consent_purposes
SET
    context = 'tenant'
WHERE
    purpose_code IN (
        'operational',
        'analytics',
        'advertising',
        'third_party_midtrans'
    );

-- Update existing guest-specific purposes
UPDATE consent_purposes
SET
    context = 'guest'
WHERE
    purpose_code IN (
        'order_processing',
        'order_communications',
        'promotional_communications',
        'payment_processing_midtrans'
    );

-- Create index for efficient filtering
CREATE INDEX idx_consent_purposes_context ON consent_purposes (context, display_order);

-- Comments
COMMENT ON COLUMN consent_purposes.context IS 'Context where consent is collected: tenant (registration) or guest (checkout)';