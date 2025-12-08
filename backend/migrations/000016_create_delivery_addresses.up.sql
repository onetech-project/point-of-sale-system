-- Create delivery_addresses table
CREATE TABLE delivery_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,

-- Address details
address_text TEXT NOT NULL,

-- Geocoding results
latitude DECIMAL(10, 8),
longitude DECIMAL(11, 8),
geocoded_address TEXT,
place_id VARCHAR(255),

-- Service area validation
is_serviceable BOOLEAN NOT NULL DEFAULT false,
service_area_zone VARCHAR(100),

-- Delivery fee
calculated_delivery_fee INTEGER CHECK (calculated_delivery_fee >= 0),
distance_km DECIMAL(6, 2),

-- Metadata
geocoded_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_delivery_addresses_order_id ON delivery_addresses (order_id);

CREATE INDEX idx_delivery_addresses_lat_lng ON delivery_addresses (latitude, longitude);

CREATE INDEX idx_delivery_addresses_place_id ON delivery_addresses (place_id);

-- Comments
COMMENT ON TABLE delivery_addresses IS 'Geocoded delivery addresses with service area validation';

COMMENT ON COLUMN delivery_addresses.is_serviceable IS 'False if address outside service area';

COMMENT ON COLUMN delivery_addresses.distance_km IS 'Distance from tenant location (Haversine)';