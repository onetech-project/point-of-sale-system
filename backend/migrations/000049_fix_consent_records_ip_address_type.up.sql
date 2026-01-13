-- Migration 000049: Fix ip_address column type to store encrypted data
-- Purpose: Change ip_address from INET to TEXT to support encrypted storage
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

-- Change ip_address column from INET to TEXT to store Vault ciphertext
ALTER TABLE consent_records
ALTER COLUMN ip_address TYPE TEXT USING ip_address::text;

-- Update comment
COMMENT ON COLUMN consent_records.ip_address IS 'Encrypted IP address using Vault Transit Engine - stores vault:v1: ciphertext';