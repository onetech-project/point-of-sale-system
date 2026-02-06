-- Migration: Fix guest_orders column sizes for encrypted data with HMAC
-- Feature: 006-uu-pdp-compliance - Encryption Performance Fix
-- Reason: customer_phone and ip_address VARCHAR(100) too small for encrypted+HMAC values
-- Format: vault:v1:base64_ciphertext:hmac_64_hex_chars requires ~200+ chars for phone
-- Date: 2026-01-06

-- Increase customer_phone from VARCHAR(100) to VARCHAR(512)
-- Phone format with HMAC: vault:v1:encrypted_phone:64_char_hmac
ALTER TABLE guest_orders
ALTER COLUMN customer_phone TYPE VARCHAR(512);

-- Increase ip_address from VARCHAR(100) to VARCHAR(512)
-- IP format with HMAC: vault:v1:encrypted_ip:64_char_hmac
ALTER TABLE guest_orders ALTER COLUMN ip_address TYPE VARCHAR(512);

-- user_agent may also need increase if it gets encrypted with HMAC
ALTER TABLE guest_orders ALTER COLUMN user_agent TYPE VARCHAR(1024);

COMMENT ON COLUMN guest_orders.customer_phone IS 'Guest customer phone (encrypted with Vault Transit Engine + HMAC)';

COMMENT ON COLUMN guest_orders.ip_address IS 'Guest customer IP address (encrypted with Vault Transit Engine + HMAC)';

COMMENT ON COLUMN guest_orders.user_agent IS 'Guest customer user agent (encrypted with Vault Transit Engine + HMAC)';