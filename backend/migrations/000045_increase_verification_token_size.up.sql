-- Migration: Increase verification_token column size for encrypted values
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022 compliance)
-- Task: Support deterministic encryption of verification tokens
-- Reason: Vault ciphertext format "vault:v1:..." is much larger than plaintext (32 chars)
-- Date: 2026-01-06

-- ============================================================================
-- Users Table - Increase verification_token column size
-- ============================================================================
-- Current: verification_token VARCHAR(255)
-- Required: VARCHAR(512) to store encrypted token values
-- Original plaintext: 32 characters
-- Encrypted format: ~200-300 characters (vault:v1:base64_encoded_ciphertext)

ALTER TABLE users ALTER COLUMN verification_token TYPE VARCHAR(512);

COMMENT ON COLUMN users.verification_token IS 'Email verification token (encrypted with Vault Transit Engine using deterministic encryption)';

-- ============================================================================
-- Performance Notes
-- ============================================================================
-- This ALTER TABLE operation is safe and does not require table rewrite in PostgreSQL
-- when increasing VARCHAR size (only metadata change).
-- Expected downtime: <1 second
-- Index rebuild: Not required (existing index idx_users_verification_token remains valid)