-- Migration: Add performance indexes for offline orders
-- Purpose: Optimize query performance for offline order operations
-- Tables: guest_orders, payment_terms, payment_records, installment_schedules

-- Index for listing offline orders by tenant and status (most common query)
CREATE INDEX IF NOT EXISTS idx_guest_orders_offline_tenant_status ON guest_orders (tenant_id, status)
WHERE
    order_type = 'offline'
    AND deleted_at IS NULL;

-- Index for finding offline orders by recorded user (staff tracking)
CREATE INDEX IF NOT EXISTS idx_guest_orders_offline_recorded_by ON guest_orders (
    tenant_id,
    recorded_by_user_id
)
WHERE
    order_type = 'offline'
    AND deleted_at IS NULL;

-- Index for soft-deleted offline orders (audit queries)
CREATE INDEX IF NOT EXISTS idx_guest_orders_offline_deleted ON guest_orders (tenant_id, deleted_at)
WHERE
    order_type = 'offline'
    AND deleted_at IS NOT NULL;

-- Index for modified offline orders (audit trail queries)
CREATE INDEX IF NOT EXISTS idx_guest_orders_offline_modified ON guest_orders (
    tenant_id,
    last_modified_at DESC
)
WHERE
    order_type = 'offline';

-- Index for payment terms lookups by order
CREATE INDEX IF NOT EXISTS idx_payment_terms_order ON payment_terms (order_id, tenant_id);

-- Index for payment records by order (payment history)
CREATE INDEX IF NOT EXISTS idx_payment_records_order ON payment_records (
    order_id,
    tenant_id,
    recorded_at DESC
);

-- Index for pending installments (analytics and alerts)
CREATE INDEX IF NOT EXISTS idx_installment_schedules_pending ON installment_schedules (tenant_id, status, due_date)
WHERE
    status = 'pending';

-- Index for analytics queries (order type breakdown by date)
CREATE INDEX IF NOT EXISTS idx_guest_orders_analytics ON guest_orders (
    tenant_id,
    order_type,
    status,
    created_at DESC
)
WHERE
    deleted_at IS NULL;

-- Composite index for event outbox processing (transactional outbox pattern)
CREATE INDEX IF NOT EXISTS idx_event_outbox_pending ON event_outbox (
    published_at,
    retry_count,
    created_at
)
WHERE
    published_at IS NULL;

-- Comment indexes for documentation
COMMENT ON INDEX idx_guest_orders_offline_tenant_status IS 'Optimizes listing offline orders by tenant and status filter';

COMMENT ON INDEX idx_guest_orders_offline_recorded_by IS 'Optimizes staff performance tracking queries';

COMMENT ON INDEX idx_guest_orders_offline_deleted IS 'Optimizes soft-deleted order audit queries';

COMMENT ON INDEX idx_guest_orders_offline_modified IS 'Optimizes edit audit trail queries';

COMMENT ON INDEX idx_payment_terms_order IS 'Optimizes payment terms lookups';

COMMENT ON INDEX idx_payment_records_order IS 'Optimizes payment history retrieval';

COMMENT ON INDEX idx_installment_schedules_pending IS 'Optimizes pending installment analytics';

COMMENT ON INDEX idx_guest_orders_analytics IS 'Optimizes analytics queries for offline vs online breakdown';

COMMENT ON INDEX idx_event_outbox_pending IS 'Optimizes outbox worker batch processing';