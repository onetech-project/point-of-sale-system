-- Create guest_orders table
CREATE TABLE guest_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_reference VARCHAR(20) UNIQUE NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

-- Order details
status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (
    status IN (
        'PENDING',
        'PAID',
        'COMPLETE',
        'CANCELLED'
    )
),
subtotal_amount INTEGER NOT NULL CHECK (subtotal_amount >= 0),
delivery_fee INTEGER NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
total_amount INTEGER NOT NULL CHECK (total_amount >= 0),

-- Customer contact
customer_name VARCHAR(255) NOT NULL,
customer_phone VARCHAR(20) NOT NULL,

-- Delivery type
delivery_type VARCHAR(20) NOT NULL CHECK (
    delivery_type IN (
        'pickup',
        'delivery',
        'dine_in'
    )
),
table_number VARCHAR(50),
notes TEXT,

-- Timestamps
created_at TIMESTAMP NOT NULL DEFAULT NOW(),
paid_at TIMESTAMP,
completed_at TIMESTAMP,
cancelled_at TIMESTAMP,

-- Metadata
session_id VARCHAR(255), ip_address INET, user_agent TEXT );

-- Indexes
CREATE INDEX idx_guest_orders_tenant_status ON guest_orders (tenant_id, status);

CREATE INDEX idx_guest_orders_order_reference ON guest_orders (order_reference);

CREATE INDEX idx_guest_orders_created_at ON guest_orders (created_at DESC);

CREATE INDEX idx_guest_orders_session_id ON guest_orders (session_id);

-- Comments
COMMENT ON TABLE guest_orders IS 'Orders placed by unauthenticated guests via public menu';

COMMENT ON COLUMN guest_orders.order_reference IS 'Human-readable order reference: GO-XXXXXX';

COMMENT ON COLUMN guest_orders.total_amount IS 'All amounts stored in smallest currency unit (IDR cents)';

COMMENT ON COLUMN guest_orders.status IS 'Order lifecycle: PENDING → PAID → COMPLETE or CANCELLED';