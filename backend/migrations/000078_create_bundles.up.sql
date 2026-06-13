-- Phase 4: bundle definitions and bundle product membership.

CREATE TABLE bundles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  sku VARCHAR(80) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  selling_price NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (selling_price >= 0),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_bundles_tenant_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX idx_bundles_tenant_name
  ON bundles(tenant_id, name);
CREATE INDEX idx_bundles_tenant_active
  ON bundles(tenant_id, is_active);

CREATE TABLE bundle_items (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  bundle_id UUID NOT NULL REFERENCES bundles(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_bundle_items_product UNIQUE (tenant_id, bundle_id, product_id)
);

CREATE INDEX idx_bundle_items_bundle
  ON bundle_items(tenant_id, bundle_id);
CREATE INDEX idx_bundle_items_product
  ON bundle_items(tenant_id, product_id);

ALTER TABLE bundles ENABLE ROW LEVEL SECURITY;
CREATE POLICY bundles_tenant_isolation ON bundles
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE bundle_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY bundle_items_tenant_isolation ON bundle_items
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE TRIGGER trg_bundles_updated_at
  BEFORE UPDATE ON bundles
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_bundle_items_updated_at
  BEFORE UPDATE ON bundle_items
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
