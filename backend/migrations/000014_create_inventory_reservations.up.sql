-- Create inventory_reservations table
CREATE TABLE inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,

-- Reservation details
quantity INTEGER NOT NULL CHECK (quantity > 0),
status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (
    status IN (
        'active',
        'expired',
        'converted',
        'released'
    )
),

-- Timing
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    released_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_inventory_reservations_order_product ON inventory_reservations (order_id, product_id);

CREATE INDEX idx_inventory_reservations_expires_at ON inventory_reservations (expires_at);

CREATE INDEX idx_inventory_reservations_status_expires ON inventory_reservations (status, expires_at);

CREATE INDEX idx_inventory_reservations_product_status ON inventory_reservations (product_id, status);

-- Comments
COMMENT ON TABLE inventory_reservations IS 'Temporary holds on inventory during checkout (15min TTL)';

COMMENT ON COLUMN inventory_reservations.status IS 'active: held, expired: TTL passed, converted: order paid, released: cancelled';

COMMENT ON COLUMN inventory_reservations.expires_at IS 'Background job marks expired reservations';