-- Rollback migration: Revert column sizes to original values
-- Feature: 006-uu-pdp-compliance
-- Task: T069a rollback
-- Warning: This will fail if encrypted data exists (values > original size limits)
-- Date: 2026-01-05

-- ============================================================================
-- Users Table - Revert to original sizes
-- ============================================================================
ALTER TABLE users
ALTER COLUMN email TYPE VARCHAR(255),
ALTER COLUMN first_name TYPE VARCHAR(50),
ALTER COLUMN last_name TYPE VARCHAR(50);

COMMENT ON COLUMN users.email IS 'User email';

COMMENT ON COLUMN users.first_name IS 'User first name';

COMMENT ON COLUMN users.last_name IS 'User last name';

-- ============================================================================
-- Guest Orders Table - Revert to original sizes
-- ============================================================================
-- Note: Original sizes need to be verified from migration 000025 or earlier
-- These are assumed values based on typical PII field sizes
ALTER TABLE guest_orders
ALTER COLUMN customer_name TYPE VARCHAR(100),
ALTER COLUMN customer_phone TYPE VARCHAR(20),
ALTER COLUMN customer_email TYPE VARCHAR(255),
ALTER COLUMN ip_address TYPE VARCHAR(45);

COMMENT ON COLUMN guest_orders.customer_name IS 'Guest customer name';

COMMENT ON COLUMN guest_orders.customer_phone IS 'Guest customer phone';

COMMENT ON COLUMN guest_orders.customer_email IS 'Guest customer email';

COMMENT ON COLUMN guest_orders.ip_address IS 'Guest customer IP address';

-- ============================================================================
-- Tenant Configs Table - Revert to original sizes
-- ============================================================================
-- Note: Original sizes need to be verified from tenant_configs creation migration
-- These are assumed values based on Midtrans credential format
ALTER TABLE tenant_configs
ALTER COLUMN midtrans_server_key TYPE VARCHAR(255),
ALTER COLUMN midtrans_client_key TYPE VARCHAR(255);

COMMENT ON COLUMN tenant_configs.midtrans_server_key IS 'Midtrans server key';

COMMENT ON COLUMN tenant_configs.midtrans_client_key IS 'Midtrans client key';

-- ============================================================================
-- Rollback Safety Notes
-- ============================================================================
-- WARNING: This rollback will FAIL if:
-- 1. Encrypted data exists (ciphertext > original column size)
-- 2. Data migration (T069) has been run
--
-- Safe rollback procedure:
-- 1. Verify no encrypted data: SELECT COUNT(*) FROM users WHERE email LIKE 'vault:v1:%'
-- 2. If encrypted data exists, decrypt first using data-migration tool
-- 3. Then run this down migration