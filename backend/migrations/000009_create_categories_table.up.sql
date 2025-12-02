-- Create categories table for organizing products
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE UNIQUE INDEX idx_categories_tenant_name ON categories(tenant_id, name);
CREATE INDEX idx_categories_tenant_order ON categories(tenant_id, display_order);

-- Row-Level Security for multi-tenant isolation
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON categories
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger for updated_at
CREATE TRIGGER trg_categories_updated_at
  BEFORE UPDATE ON categories
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
