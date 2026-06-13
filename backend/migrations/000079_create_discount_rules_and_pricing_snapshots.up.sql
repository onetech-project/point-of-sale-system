-- Phase 4: discount rules and order item pricing snapshots.

CREATE TABLE discount_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    discount_type VARCHAR(20) NOT NULL,
    discount_value NUMERIC(10,2) NOT NULL,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    exclusive BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT discount_rules_target_type_check
        CHECK (target_type IN ('product', 'bundle')),
    CONSTRAINT discount_rules_discount_type_check
        CHECK (discount_type IN ('percentage', 'fixed_amount')),
    CONSTRAINT discount_rules_discount_value_check
        CHECK (
            (discount_type = 'percentage' AND discount_value > 0 AND discount_value <= 100)
            OR
            (discount_type = 'fixed_amount' AND discount_value > 0)
        ),
    CONSTRAINT discount_rules_validity_window_check
        CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at)
);

CREATE INDEX idx_discount_rules_tenant_target
    ON discount_rules (tenant_id, target_type, target_id);

CREATE INDEX idx_discount_rules_tenant_active
    ON discount_rules (tenant_id, is_active, starts_at, ends_at);

CREATE INDEX idx_discount_rules_priority
    ON discount_rules (tenant_id, priority, created_at);

ALTER TABLE order_items
    ADD COLUMN item_type VARCHAR(20) NOT NULL DEFAULT 'product',
    ADD COLUMN bundle_id UUID REFERENCES bundles(id) ON DELETE SET NULL,
    ADD COLUMN list_unit_price INTEGER NOT NULL DEFAULT 0 CHECK (list_unit_price >= 0),
    ADD COLUMN discount_rule_id UUID REFERENCES discount_rules(id) ON DELETE SET NULL,
    ADD COLUMN discount_type VARCHAR(20),
    ADD COLUMN discount_value NUMERIC(10,2),
    ADD COLUMN discount_amount INTEGER NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    ADD COLUMN pricing_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE order_items
SET list_unit_price = unit_price
WHERE list_unit_price = 0
  AND unit_price > 0;

ALTER TABLE order_items
    ADD CONSTRAINT order_items_item_type_check
        CHECK (item_type IN ('product', 'bundle')),
    ADD CONSTRAINT order_items_discount_type_check
        CHECK (discount_type IS NULL OR discount_type IN ('percentage', 'fixed_amount')),
    ADD CONSTRAINT order_items_discount_value_check
        CHECK (discount_value IS NULL OR discount_value >= 0),
    ADD CONSTRAINT order_items_product_or_bundle_check
        CHECK (
            (item_type = 'product' AND product_id IS NOT NULL)
            OR
            (item_type = 'bundle' AND bundle_id IS NOT NULL)
        );

CREATE INDEX idx_order_items_item_type
    ON order_items (item_type);

CREATE INDEX idx_order_items_bundle_id
    ON order_items (bundle_id)
    WHERE bundle_id IS NOT NULL;

CREATE INDEX idx_order_items_discount_rule_id
    ON order_items (discount_rule_id)
    WHERE discount_rule_id IS NOT NULL;

COMMENT ON TABLE discount_rules IS 'Tenant-scoped product and bundle discount rules used for admin pricing previews and order item price snapshots';
COMMENT ON COLUMN discount_rules.exclusive IS 'When true, this discount is applied alone; MVP pricing applies at most one rule and defaults to exclusive behavior';
COMMENT ON COLUMN order_items.unit_price IS 'Discounted final unit price charged at order time';
COMMENT ON COLUMN order_items.list_unit_price IS 'Original unit price before discount at order time';
COMMENT ON COLUMN order_items.pricing_snapshot IS 'Immutable pricing context for discounts and future profitability reporting';
