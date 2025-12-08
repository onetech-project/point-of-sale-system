-- Add Midtrans payment configuration to tenant_configs
-- Each tenant will have their own Midtrans credentials

ALTER TABLE tenant_configs
ADD COLUMN midtrans_server_key TEXT,
ADD COLUMN midtrans_client_key TEXT,
ADD COLUMN midtrans_merchant_id TEXT,
ADD COLUMN midtrans_environment VARCHAR(20) DEFAULT 'sandbox' CHECK (
    midtrans_environment IN ('sandbox', 'production')
);

-- Comments
COMMENT ON COLUMN tenant_configs.midtrans_server_key IS 'Tenant-specific Midtrans server key (encrypted in production)';

COMMENT ON COLUMN tenant_configs.midtrans_client_key IS 'Tenant-specific Midtrans client key for frontend';

COMMENT ON COLUMN tenant_configs.midtrans_merchant_id IS 'Tenant-specific Midtrans merchant ID';

COMMENT ON COLUMN tenant_configs.midtrans_environment IS 'Midtrans environment: sandbox or production';