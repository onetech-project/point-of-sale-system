-- Migration: Increase password reset token column size for encrypted values
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022 compliance)
-- Task: Support deterministic encryption of password reset tokens
-- Reason: Vault ciphertext format "vault:v1:..." is much larger than plaintext (64 chars)
-- Date: 2026-01-06

-- ============================================================================
-- Password Reset Tokens Table - Increase token column size
-- ============================================================================
-- Current: token VARCHAR(255)
-- Required: VARCHAR(512) to store encrypted token values
-- Original plaintext: 64 characters (hex-encoded 32 bytes)
-- Encrypted format: ~200-300 characters (vault:v1:base64_encoded_ciphertext)

ALTER TABLE password_reset_tokens
ALTER COLUMN token TYPE VARCHAR(512);

COMMENT ON COLUMN password_reset_tokens.token IS 'Password reset token (encrypted with Vault Transit Engine using deterministic encryption)';

-- ============================================================================
-- Performance Notes
-- ============================================================================
-- This ALTER TABLE operation is safe and does not require table rewrite in PostgreSQL
-- when increasing VARCHAR size (only metadata change).
-- Expected downtime: <1 second
-- Index rebuild: Not required (existing index idx_password_reset_tokens_token remains valid)