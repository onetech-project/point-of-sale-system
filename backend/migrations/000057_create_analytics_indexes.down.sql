-- Remove analytics-optimized indexes

DROP INDEX IF EXISTS idx_guest_orders_analytics;

DROP INDEX IF EXISTS idx_guest_orders_daily_sales;

DROP INDEX IF EXISTS idx_order_items_analytics;

DROP INDEX IF EXISTS idx_guest_orders_customer_analytics;

DROP INDEX IF EXISTS idx_guest_orders_new_customers;

DROP INDEX IF EXISTS idx_products_low_stock;

DROP INDEX IF EXISTS idx_products_inventory_value;

DROP INDEX IF EXISTS idx_products_out_of_stock;

DROP INDEX IF EXISTS idx_products_category_analytics;