-- Create stock_adjustments table for audit trail
CREATE TABLE stock_adjustments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  previous_quantity INTEGER NOT NULL,
  new_quantity INTEGER NOT NULL,
  quantity_delta INTEGER NOT NULL,
  reason VARCHAR(50) NOT NULL,
  notes TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_stock_adjustments_product ON stock_adjustments(product_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_tenant_date ON stock_adjustments(tenant_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_user ON stock_adjustments(user_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_reason ON stock_adjustments(tenant_id, reason, created_at DESC);

-- Row-Level Security for multi-tenant isolation
ALTER TABLE stock_adjustments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stock_adjustments
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger to calculate quantity delta
CREATE OR REPLACE FUNCTION calculate_quantity_delta()
RETURNS TRIGGER AS $$
BEGIN
  NEW.quantity_delta := NEW.new_quantity - NEW.previous_quantity;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_calculate_delta
  BEFORE INSERT ON stock_adjustments
  FOR EACH ROW
  EXECUTE FUNCTION calculate_quantity_delta();
