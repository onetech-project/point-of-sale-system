-- Migration 000049 down: Revert ip_address column type back to INET

-- Note: This will fail if there's encrypted data in the column
-- Data must be decrypted first before reverting
ALTER TABLE consent_records
ALTER COLUMN ip_address TYPE INET USING ip_address::inet;