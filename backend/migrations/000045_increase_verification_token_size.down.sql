-- Rollback migration: Restore original verification_token column size
-- WARNING: This rollback will FAIL if any encrypted tokens exist in the table
-- that are longer than 255 characters. You must decrypt tokens before rollback.

ALTER TABLE users ALTER COLUMN verification_token TYPE VARCHAR(255);

COMMENT ON COLUMN users.verification_token IS 'Token for email verification (expires in 24h)';