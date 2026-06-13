DROP INDEX IF EXISTS idx_order_items_discount_rule_id;
DROP INDEX IF EXISTS idx_order_items_bundle_id;
DROP INDEX IF EXISTS idx_order_items_item_type;

ALTER TABLE order_items
    DROP CONSTRAINT IF EXISTS order_items_product_or_bundle_check,
    DROP CONSTRAINT IF EXISTS order_items_discount_value_check,
    DROP CONSTRAINT IF EXISTS order_items_discount_type_check,
    DROP CONSTRAINT IF EXISTS order_items_item_type_check,
    DROP COLUMN IF EXISTS pricing_snapshot,
    DROP COLUMN IF EXISTS discount_amount,
    DROP COLUMN IF EXISTS discount_value,
    DROP COLUMN IF EXISTS discount_type,
    DROP COLUMN IF EXISTS discount_rule_id,
    DROP COLUMN IF EXISTS list_unit_price,
    DROP COLUMN IF EXISTS bundle_id,
    DROP COLUMN IF EXISTS item_type;

DROP INDEX IF EXISTS idx_discount_rules_priority;
DROP INDEX IF EXISTS idx_discount_rules_tenant_active;
DROP INDEX IF EXISTS idx_discount_rules_tenant_target;

DROP TABLE IF EXISTS discount_rules;
