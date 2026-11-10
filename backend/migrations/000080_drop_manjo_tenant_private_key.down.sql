ALTER TABLE tenant_configs
ADD COLUMN IF NOT EXISTS manjo_private_key TEXT;

COMMENT ON COLUMN tenant_configs.manjo_private_key IS 'PEM RSA private key for the SHA256withRSA access token signature (Vault-encrypted)';
