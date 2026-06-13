-- Phase 3: immutable order item COGS snapshots captured at fulfillment time.

CREATE TABLE order_item_cost_snapshots (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
  order_item_id UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
  item_type VARCHAR(20) NOT NULL DEFAULT 'product' CHECK (item_type IN ('product', 'bundle')),
  product_id UUID REFERENCES products(id) ON DELETE RESTRICT,
  bundle_id UUID,
  recipe_id UUID REFERENCES recipes(id) ON DELETE SET NULL,
  recipe_version INTEGER NOT NULL DEFAULT 0 CHECK (recipe_version >= 0),
  quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
  material_cost NUMERIC(18,2) NOT NULL CHECK (material_cost >= 0),
  fixed_overhead_cost NUMERIC(18,2) NOT NULL CHECK (fixed_overhead_cost >= 0),
  percentage_overhead_cost NUMERIC(18,2) NOT NULL CHECK (percentage_overhead_cost >= 0),
  total_cogs NUMERIC(18,2) NOT NULL CHECK (total_cogs >= 0),
  selling_price NUMERIC(18,2) NOT NULL CHECK (selling_price >= 0),
  gross_profit NUMERIC(18,2) NOT NULL,
  gross_margin_percentage NUMERIC(9,4) NOT NULL,
  discount_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
  pricing_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  component_costs JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_order_item_cost_snapshots_order_item UNIQUE (tenant_id, order_item_id),
  CONSTRAINT chk_order_item_cost_snapshots_item_target CHECK (
    (item_type = 'product' AND product_id IS NOT NULL)
    OR
    (item_type = 'bundle' AND bundle_id IS NOT NULL)
  )
);

CREATE INDEX idx_order_item_cost_snapshots_order
  ON order_item_cost_snapshots(tenant_id, order_id);
CREATE INDEX idx_order_item_cost_snapshots_product
  ON order_item_cost_snapshots(tenant_id, product_id);
CREATE INDEX idx_order_item_cost_snapshots_bundle
  ON order_item_cost_snapshots(tenant_id, bundle_id)
  WHERE bundle_id IS NOT NULL;
CREATE INDEX idx_order_item_cost_snapshots_created
  ON order_item_cost_snapshots(tenant_id, created_at DESC);

ALTER TABLE order_item_cost_snapshots ENABLE ROW LEVEL SECURITY;
CREATE POLICY order_item_cost_snapshots_tenant_isolation ON order_item_cost_snapshots
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
