-- Create products table for inventory management
CREATE TABLE products (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  sku VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
  selling_price DECIMAL(10,2) NOT NULL CHECK (selling_price >= 0),
  cost_price DECIMAL(10,2) NOT NULL CHECK (cost_price >= 0),
  tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
  stock_quantity INTEGER NOT NULL DEFAULT 0,
  photo_path VARCHAR(500),
  photo_size INTEGER,
  archived_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE UNIQUE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_products_tenant_name ON products(tenant_id, name);
CREATE INDEX idx_products_tenant_category ON products(tenant_id, category_id);
CREATE INDEX idx_products_tenant_archived ON products(tenant_id, archived_at);
CREATE INDEX idx_products_stock_quantity ON products(tenant_id, stock_quantity) WHERE archived_at IS NULL;

-- Row-Level Security for multi-tenant isolation
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON products
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger for updated_at
CREATE TRIGGER trg_products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
