-- Add analytics-optimized indexes for dashboard queries
-- These indexes improve performance for sales overview, customer insights, and inventory tasks

-- Sales Analytics Indexes
-- Composite index for orders analytics queries (tenant_id, status, created_at)
-- Optimizes: SELECT COUNT(*), SUM(total_amount) FROM guest_orders WHERE tenant_id = X AND status = 'COMPLETE' AND created_at BETWEEN...
CREATE INDEX IF NOT EXISTS idx_guest_orders_analytics ON guest_orders (
    tenant_id,
    status,
    created_at DESC
);

-- Index for daily sales aggregation (tenant_id, created_at)
-- Optimizes: GROUP BY DATE(created_at) queries
CREATE INDEX IF NOT EXISTS idx_guest_orders_daily_sales ON guest_orders (tenant_id, created_at DESC)
WHERE
    status = 'COMPLETE';

-- Index for order items analytics (product_id, total_price)
-- Optimizes: SELECT product_id, SUM(quantity), SUM(total_price) FROM order_items...
CREATE INDEX IF NOT EXISTS idx_order_items_analytics ON order_items (product_id, total_price);

-- Customer Analytics Indexes
-- Index for customer order count and total spent
-- Optimizes: SELECT customer_phone, COUNT(*), SUM(total_amount) FROM guest_orders WHERE tenant_id = X GROUP BY customer_phone
CREATE INDEX IF NOT EXISTS idx_guest_orders_customer_analytics ON guest_orders (
    customer_phone,
    tenant_id,
    total_amount
)
WHERE
    status = 'COMPLETE';

-- Index for new customer tracking (tenant_id, customer_phone, created_at)
-- Optimizes: SELECT DISTINCT customer_phone FROM guest_orders WHERE tenant_id = X AND created_at BETWEEN...
CREATE INDEX IF NOT EXISTS idx_guest_orders_new_customers ON guest_orders (
    tenant_id,
    customer_phone,
    created_at DESC
)
WHERE
    status = 'COMPLETE';

-- Inventory Analytics Indexes
-- Partial index for low stock products
-- Optimizes: SELECT * FROM products WHERE tenant_id = X AND stock_quantity <= 10
CREATE INDEX IF NOT EXISTS idx_products_low_stock ON products (tenant_id, stock_quantity)
WHERE
    stock_quantity <= 10
    AND archived_at IS NULL;

-- Index for product inventory value calculation
-- Optimizes: SELECT SUM(stock_quantity * cost_price) FROM products WHERE tenant_id = X
CREATE INDEX IF NOT EXISTS idx_products_inventory_value ON products (
    tenant_id,
    stock_quantity,
    cost_price
)
WHERE
    archived_at IS NULL;

-- Index for out of stock products
-- Optimizes: SELECT COUNT(*) FROM products WHERE tenant_id = X AND quantity = 0
CREATE INDEX IF NOT EXISTS idx_products_out_of_stock ON products (tenant_id)
WHERE
    stock_quantity = 0
    AND archived_at IS NULL;

-- Category sales breakdown index
-- Optimizes: SELECT category_id, SUM(stock_quantity * cost_price) FROM order_items JOIN products...
CREATE INDEX IF NOT EXISTS idx_products_category_analytics ON products (
    tenant_id,
    category_id,
    cost_price
)
WHERE
    archived_at IS NULL;

-- Comment explaining the indexes
COMMENT ON INDEX idx_guest_orders_analytics IS 'Composite index for sales analytics queries filtering by tenant, status, and date range';

COMMENT ON INDEX idx_guest_orders_daily_sales IS 'Partial index for daily sales aggregation queries (COMPLETE orders only)';

COMMENT ON INDEX idx_order_items_analytics IS 'Index for product-level sales analytics (top products, revenue by product)';

COMMENT ON INDEX idx_guest_orders_customer_analytics IS 'Index for customer analytics (order count, total spent, customer segments)';

COMMENT ON INDEX idx_guest_orders_new_customers IS 'Index for new customer tracking and growth analysis';

COMMENT ON INDEX idx_products_low_stock IS 'Partial index for inventory alerts (low stock products)';

COMMENT ON INDEX idx_products_inventory_value IS 'Index for total inventory value calculation';

COMMENT ON INDEX idx_products_out_of_stock IS 'Partial index for out of stock product count';

COMMENT ON INDEX idx_products_category_analytics IS 'Index for category-level sales breakdown';