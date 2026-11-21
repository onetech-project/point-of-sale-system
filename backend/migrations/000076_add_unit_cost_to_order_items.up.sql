-- Migration: 000076_add_unit_cost_to_order_items
-- Description: Snapshot each order item's cost price at transaction time so profit reports are
-- computed from the transaction itself rather than from the product's current cost.
--
-- Items written before this migration had no cost recorded. They are backfilled once from the
-- product's current cost_price, which is the closest value still available. Items whose product
-- has since been deleted stay NULL and are left out of profit reports.

ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS unit_cost INTEGER CHECK (unit_cost >= 0);

UPDATE order_items oi
SET unit_cost = ROUND(p.cost_price)::integer
FROM products p
WHERE p.id = oi.product_id
  AND oi.unit_cost IS NULL;

COMMENT ON COLUMN order_items.unit_cost IS 'Snapshot of product cost_price at time of order (NULL when unknown); profit = total_price - quantity * unit_cost';
