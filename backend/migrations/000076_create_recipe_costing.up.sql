-- Phase 2: product recipes, versioning, overheads, and costing foundation.

CREATE TABLE recipes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  version INTEGER NOT NULL CHECK (version > 0),
  yield_quantity NUMERIC(18,6) NOT NULL DEFAULT 1 CHECK (yield_quantity > 0),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  effective_from TIMESTAMP NOT NULL DEFAULT NOW(),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_recipes_product_version UNIQUE (tenant_id, product_id, version)
);

CREATE UNIQUE INDEX idx_recipes_active_product
  ON recipes(tenant_id, product_id)
  WHERE is_active = TRUE;
CREATE INDEX idx_recipes_product_versions
  ON recipes(tenant_id, product_id, version DESC);

CREATE TABLE recipe_items (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
  ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE RESTRICT,
  quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
  uom_id UUID NOT NULL REFERENCES uoms(id) ON DELETE RESTRICT,
  normalized_quantity_base NUMERIC(18,6) NOT NULL CHECK (normalized_quantity_base > 0),
  waste_percentage NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (waste_percentage >= 0 AND waste_percentage <= 100),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recipe_items_recipe
  ON recipe_items(tenant_id, recipe_id);
CREATE INDEX idx_recipe_items_ingredient
  ON recipe_items(tenant_id, ingredient_id);

CREATE TABLE recipe_overheads (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
  name VARCHAR(120) NOT NULL,
  calculation_type VARCHAR(20) NOT NULL CHECK (calculation_type IN ('FIXED', 'PERCENTAGE')),
  amount NUMERIC(18,6) NOT NULL CHECK (amount >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recipe_overheads_recipe
  ON recipe_overheads(tenant_id, recipe_id);

ALTER TABLE recipes ENABLE ROW LEVEL SECURITY;
CREATE POLICY recipes_tenant_isolation ON recipes
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE recipe_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY recipe_items_tenant_isolation ON recipe_items
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE recipe_overheads ENABLE ROW LEVEL SECURITY;
CREATE POLICY recipe_overheads_tenant_isolation ON recipe_overheads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE TRIGGER trg_recipes_updated_at
  BEFORE UPDATE ON recipes
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_recipe_items_updated_at
  BEFORE UPDATE ON recipe_items
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_recipe_overheads_updated_at
  BEFORE UPDATE ON recipe_overheads
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
