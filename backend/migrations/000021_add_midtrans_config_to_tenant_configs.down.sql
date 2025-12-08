-- Rollback: Remove Midtrans payment configuration from tenant_configs

ALTER TABLE tenant_configs
DROP COLUMN IF EXISTS midtrans_server_key,
DROP COLUMN IF EXISTS midtrans_client_key,
DROP COLUMN IF EXISTS midtrans_merchant_id,
DROP COLUMN IF EXISTS midtrans_environment;