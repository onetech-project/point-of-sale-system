-- Add updated_at column to invitations table
ALTER TABLE invitations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Create an index for better query performance
CREATE INDEX IF NOT EXISTS idx_invitations_updated_at ON invitations(updated_at);
