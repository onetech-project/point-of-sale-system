-- Create order_notes table for audit trail
-- Each note records who added it and when
CREATE TABLE IF NOT EXISTS order_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    order_id UUID NOT NULL REFERENCES guest_orders (id) ON DELETE CASCADE,
    note TEXT NOT NULL,
    created_by_user_id UUID, -- NULL for system-generated notes
    created_by_name VARCHAR(255), -- Store name for display
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for efficient querying by order_id
CREATE INDEX idx_order_notes_order_id ON order_notes (order_id);

-- Index for querying by creation time
CREATE INDEX idx_order_notes_created_at ON order_notes (created_at DESC);

-- Migrate existing notes from guest_orders.notes to order_notes
INSERT INTO
    order_notes (
        order_id,
        note,
        created_by_name,
        created_at
    )
SELECT id, notes, 'System', created_at
FROM guest_orders
WHERE
    notes IS NOT NULL
    AND notes != '';