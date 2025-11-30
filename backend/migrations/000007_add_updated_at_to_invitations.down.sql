-- Remove index
DROP INDEX IF EXISTS idx_invitations_updated_at;

-- Remove updated_at column from invitations table
ALTER TABLE invitations DROP COLUMN IF EXISTS updated_at;
