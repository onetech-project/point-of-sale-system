-- Migration: Increase column sizes to accommodate Vault Transit Engine ciphertext
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022 compliance)
-- Task: T069a - Prerequisite for T069 data migration
-- Reason: Vault ciphertext format "vault:v1:..." is ~8-10x larger than plaintext
-- Date: 2026-01-05

-- ============================================================================
-- Users Table - Increase PII column sizes
-- ============================================================================
-- Current: first_name VARCHAR(50), last_name VARCHAR(50)
-- Required: VARCHAR(512) to store encrypted values
-- email already VARCHAR(255), which is sufficient but increase for consistency

ALTER TABLE users
ALTER COLUMN email TYPE VARCHAR(512),
ALTER COLUMN first_name TYPE VARCHAR(512),
ALTER COLUMN last_name TYPE VARCHAR(512);

COMMENT ON COLUMN users.email IS 'User email (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN users.first_name IS 'User first name (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN users.last_name IS 'User last name (encrypted with Vault Transit Engine)';

-- ============================================================================
-- Guest Orders Table - Increase customer PII column sizes
-- ============================================================================
-- Current: customer_name, customer_phone, customer_email, ip_address have small sizes
-- Required: Larger sizes for encrypted values

ALTER TABLE guest_orders
ALTER COLUMN customer_name TYPE VARCHAR(512),
ALTER COLUMN customer_phone TYPE VARCHAR(100),
ALTER COLUMN customer_email TYPE VARCHAR(512),
ALTER COLUMN ip_address TYPE VARCHAR(100);

COMMENT ON COLUMN guest_orders.customer_name IS 'Guest customer name (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN guest_orders.customer_phone IS 'Guest customer phone (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN guest_orders.customer_email IS 'Guest customer email (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN guest_orders.ip_address IS 'Guest customer IP address (encrypted with Vault Transit Engine)';

-- ============================================================================
-- Tenant Configs Table - Increase payment credential column sizes
-- ============================================================================
-- Current: midtrans_server_key, midtrans_client_key have small sizes
-- Required: VARCHAR(512) for encrypted payment credentials

ALTER TABLE tenant_configs
ALTER COLUMN midtrans_server_key TYPE VARCHAR(512),
ALTER COLUMN midtrans_client_key TYPE VARCHAR(512);

COMMENT ON COLUMN tenant_configs.midtrans_server_key IS 'Midtrans server key (encrypted with Vault Transit Engine)';

COMMENT ON COLUMN tenant_configs.midtrans_client_key IS 'Midtrans client key (encrypted with Vault Transit Engine)';

-- ============================================================================
-- Performance Notes
-- ============================================================================
-- These ALTER TABLE operations are safe and do not require table rewrite in PostgreSQL
-- when increasing VARCHAR size (only metadata change).
-- Expected downtime: <1 second per table
-- Index rebuild: Not required (VARCHAR size increase doesn't affect existing indexes)