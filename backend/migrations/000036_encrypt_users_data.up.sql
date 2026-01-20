-- Migration: 000036_encrypt_users_data
-- Purpose: Document encryption approach for users table
--
-- NO SCHEMA CHANGES REQUIRED
--
-- This migration serves as documentation that encryption is implemented at the application layer.
-- The existing columns (email, first_name, last_name, verification_token) will store encrypted values
-- in the format "vault:v1:..." using HashiCorp Vault Transit Engine (AES-256-GCM96).
--
-- Application-layer encryption provides:
-- 1. Transparent encryption/decryption in repository layer
-- 2. No schema complexity (no duplicate *_encrypted columns)
-- 3. Backup files automatically contain encrypted data
-- 4. Database administrators cannot read PII without Vault access
--
-- Implementation: See backend/user-service/src/repository/user_repo.go for encryption logic
-- Encryption utility: See backend/user-service/src/utils/encryption.go for VaultClient

-- This is a placeholder migration - no SQL execution needed
SELECT 1 WHERE FALSE;