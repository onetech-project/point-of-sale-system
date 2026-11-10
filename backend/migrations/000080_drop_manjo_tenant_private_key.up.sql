-- Migration: 000080_drop_manjo_tenant_private_key
-- Description: The RSA key pair Manjo uses is the platform's, not each tenant's.
-- The private key that signs our access token requests to Manjo now comes from
-- order-service's MANJO_PRIVATE_KEY, so the per-tenant column is dropped.

ALTER TABLE tenant_configs
DROP COLUMN IF EXISTS manjo_private_key;
