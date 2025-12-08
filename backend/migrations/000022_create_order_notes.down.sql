-- Drop order_notes table
DROP INDEX IF EXISTS idx_order_notes_created_at;

DROP INDEX IF EXISTS idx_order_notes_order_id;

DROP TABLE IF EXISTS order_notes;