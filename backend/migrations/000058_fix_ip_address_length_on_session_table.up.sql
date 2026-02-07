ALTER TABLE sessions
ALTER COLUMN ip_address TYPE TEXT USING ip_address::text;

COMMENT ON COLUMN sessions.ip_address IS 'Encrypted IP address using Vault Transit Engine - stores vault:v1: ciphertext';