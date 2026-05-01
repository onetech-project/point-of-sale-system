-- Rollback: Remove performance indexes for offline orders

DROP INDEX IF EXISTS idx_event_outbox_pending;

DROP INDEX IF EXISTS idx_guest_orders_analytics;

DROP INDEX IF EXISTS idx_installment_schedules_pending;

DROP INDEX IF EXISTS idx_payment_records_order;

DROP INDEX IF EXISTS idx_payment_terms_order;

DROP INDEX IF EXISTS idx_guest_orders_offline_modified;

DROP INDEX IF EXISTS idx_guest_orders_offline_deleted;

DROP INDEX IF EXISTS idx_guest_orders_offline_recorded_by;

DROP INDEX IF EXISTS idx_guest_orders_offline_tenant_status;