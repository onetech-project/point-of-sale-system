-- Create tenant_configs table for guest ordering configuration
CREATE TABLE tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,

-- Delivery types
enabled_delivery_types TEXT [] NOT NULL DEFAULT '{pickup}' CHECK (
    array_length(enabled_delivery_types, 1) > 0
),

-- Service area (for delivery)
service_area_type VARCHAR(20) CHECK (
    service_area_type IN ('radius', 'polygon', NULL)
),
service_area_data JSONB,

-- Delivery fee pricing
enable_delivery_fee_calculation BOOLEAN DEFAULT true,
delivery_fee_type VARCHAR(20) CHECK (
    delivery_fee_type IN (
        'distance',
        'zone',
        'flat',
        NULL
    )
),
delivery_fee_config JSONB,

-- Operational settings
inventory_reservation_ttl_minutes INTEGER DEFAULT 15 CHECK (
    inventory_reservation_ttl_minutes >= 5
),
min_order_amount INTEGER DEFAULT 0 CHECK (min_order_amount >= 0),

-- Tenant location (for distance calculation)
location_lat DECIMAL(10, 8),
    location_lng DECIMAL(11, 8),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_tenant_configs_tenant_id ON tenant_configs (tenant_id);

-- Comments
COMMENT ON TABLE tenant_configs IS 'Per-tenant configuration for guest ordering feature';

COMMENT ON COLUMN tenant_configs.enabled_delivery_types IS 'Array of: pickup, delivery, dine_in';

COMMENT ON COLUMN tenant_configs.service_area_data IS 'Flexible JSONB: radius={center:{lat,lng},radius_km:5} OR polygon={coordinates:[[lat,lng],...]}';

COMMENT ON COLUMN tenant_configs.delivery_fee_config IS 'Flexible JSONB pricing rules (see research.md)';

COMMENT ON COLUMN tenant_configs.enable_delivery_fee_calculation IS 'Tenant can disable automatic delivery fee calculation';

-- Default config for existing tenants
INSERT INTO
    tenant_configs (
        tenant_id,
        enabled_delivery_types,
        enable_delivery_fee_calculation,
        inventory_reservation_ttl_minutes
    )
SELECT id, '{pickup}', false, 15
FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;