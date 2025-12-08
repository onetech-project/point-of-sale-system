-- Create order_items table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,

-- Item details (snapshot at order time)
product_name VARCHAR(255) NOT NULL,
product_sku VARCHAR(100),
quantity INTEGER NOT NULL CHECK (quantity > 0),
unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
total_price INTEGER NOT NULL CHECK (total_price >= 0),

-- Metadata
created_at TIMESTAMP NOT NULL DEFAULT NOW() );

-- Indexes
CREATE INDEX idx_order_items_order_id ON order_items (order_id);

CREATE INDEX idx_order_items_product_id ON order_items (product_id);

-- Comments
COMMENT ON TABLE order_items IS 'Line items for guest orders with price snapshots';

COMMENT ON COLUMN order_items.product_name IS 'Snapshot of product name at time of order (preserves history)';

COMMENT ON COLUMN order_items.total_price IS 'Must equal quantity * unit_price';