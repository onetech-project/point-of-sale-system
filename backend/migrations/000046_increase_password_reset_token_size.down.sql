-- Rollback migration: Restore original password reset token column size
-- WARNING: This rollback will FAIL if any encrypted tokens exist in the table
-- that are longer than 255 characters. You must decrypt tokens before rollback.

ALTER TABLE password_reset_tokens
ALTER COLUMN token TYPE VARCHAR(255);

COMMENT ON COLUMN password_reset_tokens.token IS 'Unique reset token sent via email';