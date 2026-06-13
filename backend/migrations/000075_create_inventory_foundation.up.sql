-- Inventory foundation: UoMs, ingredients, conversions, and stock ledger.

CREATE TABLE uoms (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  code VARCHAR(50) NOT NULL,
  name VARCHAR(120) NOT NULL,
  category VARCHAR(30) NOT NULL CHECK (category IN ('weight', 'volume', 'count', 'packaging')),
  is_base_unit BOOLEAN NOT NULL DEFAULT FALSE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_uoms_tenant_code UNIQUE (tenant_id, code),
  CONSTRAINT uq_uoms_global_code UNIQUE (code, tenant_id)
);

CREATE UNIQUE INDEX idx_uoms_global_code ON uoms(code) WHERE tenant_id IS NULL;
CREATE INDEX idx_uoms_tenant_category ON uoms(tenant_id, category, is_active);

CREATE TABLE ingredients (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  outlet_id UUID,
  sku VARCHAR(80) NOT NULL,
  name VARCHAR(255) NOT NULL,
  base_uom_id UUID NOT NULL REFERENCES uoms(id) ON DELETE RESTRICT,
  current_stock_base NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (current_stock_base >= 0),
  average_cost_per_base_unit NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (average_cost_per_base_unit >= 0),
  minimum_stock_base NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (minimum_stock_base >= 0),
  track_stock BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_ingredients_tenant_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX idx_ingredients_tenant_name ON ingredients(tenant_id, name);
CREATE INDEX idx_ingredients_tenant_active ON ingredients(tenant_id, is_active);
CREATE INDEX idx_ingredients_low_stock ON ingredients(tenant_id, current_stock_base, minimum_stock_base)
  WHERE is_active = TRUE AND track_stock = TRUE;

CREATE TABLE ingredient_uom_conversions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  ingredient_id UUID REFERENCES ingredients(id) ON DELETE CASCADE,
  from_uom_id UUID NOT NULL REFERENCES uoms(id) ON DELETE RESTRICT,
  to_uom_id UUID NOT NULL REFERENCES uoms(id) ON DELETE RESTRICT,
  multiplier NUMERIC(18,6) NOT NULL CHECK (multiplier > 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_ingredient_conversions UNIQUE (tenant_id, ingredient_id, from_uom_id, to_uom_id),
  CONSTRAINT chk_packaging_requires_ingredient CHECK (
    ingredient_id IS NOT NULL OR tenant_id IS NULL
  )
);

CREATE INDEX idx_ingredient_conversions_lookup
  ON ingredient_uom_conversions(tenant_id, ingredient_id, from_uom_id, to_uom_id);
CREATE UNIQUE INDEX idx_ingredient_conversions_global_unique
  ON ingredient_uom_conversions(from_uom_id, to_uom_id)
  WHERE tenant_id IS NULL AND ingredient_id IS NULL;

CREATE TABLE stock_movements (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  outlet_id UUID,
  ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE RESTRICT,
  movement_type VARCHAR(40) NOT NULL CHECK (movement_type IN (
    'INITIAL_STOCK',
    'PURCHASE_IN',
    'SALE_RESERVATION',
    'SALE_RESERVATION_RELEASED',
    'SALE_CONSUMPTION',
    'WASTE',
    'MANUAL_ADJUSTMENT',
    'RETURN_IN',
    'RETURN_OUT'
  )),
  quantity_base NUMERIC(18,6) NOT NULL,
  unit_cost NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
  total_cost NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (total_cost >= 0),
  reference_type VARCHAR(60),
  reference_id VARCHAR(120),
  idempotency_key VARCHAR(160),
  reason VARCHAR(120),
  occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_stock_movement_quantity_nonzero CHECK (quantity_base <> 0)
);

CREATE UNIQUE INDEX idx_stock_movements_idempotency
  ON stock_movements(tenant_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_stock_movements_ingredient_date
  ON stock_movements(tenant_id, ingredient_id, occurred_at DESC);
CREATE INDEX idx_stock_movements_reference
  ON stock_movements(tenant_id, reference_type, reference_id);

ALTER TABLE uoms ENABLE ROW LEVEL SECURITY;
CREATE POLICY uoms_tenant_isolation ON uoms
  USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ingredients ENABLE ROW LEVEL SECURITY;
CREATE POLICY ingredients_tenant_isolation ON ingredients
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ingredient_uom_conversions ENABLE ROW LEVEL SECURITY;
CREATE POLICY ingredient_uom_conversions_tenant_isolation ON ingredient_uom_conversions
  USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
CREATE POLICY stock_movements_tenant_isolation ON stock_movements
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE TRIGGER trg_uoms_updated_at
  BEFORE UPDATE ON uoms
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_ingredients_updated_at
  BEFORE UPDATE ON ingredients
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_ingredient_uom_conversions_updated_at
  BEFORE UPDATE ON ingredient_uom_conversions
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

INSERT INTO uoms (code, name, category, is_base_unit) VALUES
  ('kilogram', 'Kilogram', 'weight', TRUE),
  ('gram', 'Gram', 'weight', TRUE),
  ('milligram', 'Milligram', 'weight', TRUE),
  ('liter', 'Liter', 'volume', TRUE),
  ('milliliter', 'Milliliter', 'volume', TRUE),
  ('piece', 'Piece', 'count', TRUE),
  ('portion', 'Portion', 'count', FALSE),
  ('slice', 'Slice', 'count', FALSE),
  ('egg', 'Egg', 'count', FALSE),
  ('item', 'Item', 'count', TRUE),
  ('tray', 'Tray', 'packaging', FALSE),
  ('box', 'Box', 'packaging', FALSE),
  ('sachet', 'Sachet', 'packaging', FALSE),
  ('bottle', 'Bottle', 'packaging', FALSE),
  ('pack', 'Pack', 'packaging', FALSE)
ON CONFLICT DO NOTHING;

INSERT INTO ingredient_uom_conversions (from_uom_id, to_uom_id, multiplier)
SELECT f.id, t.id, v.multiplier
FROM (VALUES
  ('kilogram', 'gram', 1000::NUMERIC),
  ('gram', 'kilogram', 0.001::NUMERIC),
  ('gram', 'milligram', 1000::NUMERIC),
  ('milligram', 'gram', 0.001::NUMERIC),
  ('kilogram', 'milligram', 1000000::NUMERIC),
  ('milligram', 'kilogram', 0.000001::NUMERIC),
  ('liter', 'milliliter', 1000::NUMERIC),
  ('milliliter', 'liter', 0.001::NUMERIC)
) AS v(from_code, to_code, multiplier)
JOIN uoms f ON f.code = v.from_code AND f.tenant_id IS NULL
JOIN uoms t ON t.code = v.to_code AND t.tenant_id IS NULL
ON CONFLICT DO NOTHING;
