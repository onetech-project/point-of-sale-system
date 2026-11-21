-- Migration: 000076_add_unit_cost_to_order_items

ALTER TABLE order_items
DROP COLUMN IF EXISTS unit_cost;
