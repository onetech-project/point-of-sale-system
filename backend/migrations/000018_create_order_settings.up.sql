-- Create order_settings table
CREATE TABLE IF NOT EXISTS order_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Delivery type toggles
    delivery_enabled BOOLEAN DEFAULT true,
    pickup_enabled BOOLEAN DEFAULT true,
    dine_in_enabled BOOLEAN DEFAULT false,
    
    -- Pricing settings (in cents/smallest currency unit)
    default_delivery_fee INTEGER DEFAULT 10000,
    min_order_amount INTEGER DEFAULT 20000,
    max_delivery_distance DECIMAL(10, 2) DEFAULT 10.0,
    
    -- Processing settings
    estimated_prep_time INTEGER DEFAULT 30, -- minutes
    auto_accept_orders BOOLEAN DEFAULT false,
    require_phone_verification BOOLEAN DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure one settings record per tenant
    UNIQUE(tenant_id)
);

-- Create index for faster tenant lookups
CREATE INDEX idx_order_settings_tenant_id ON order_settings(tenant_id);

-- Create trigger to update updated_at
CREATE TRIGGER update_order_settings_updated_at
    BEFORE UPDATE ON order_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
