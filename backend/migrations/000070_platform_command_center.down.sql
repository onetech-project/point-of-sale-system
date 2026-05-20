DROP INDEX IF EXISTS idx_users_tenant_last_login;
DROP INDEX IF EXISTS idx_billing_invoices_status_due_at;
DROP INDEX IF EXISTS idx_billing_invoices_status_paid_at;
DROP INDEX IF EXISTS idx_tenants_subscription_plan_cycle;
DROP INDEX IF EXISTS idx_tenants_status_subscription_status;

DROP INDEX IF EXISTS idx_platform_tenant_notes_tenant_id;
DROP TABLE IF EXISTS platform_tenant_notes;

ALTER TABLE tenant_lifecycle_events
DROP CONSTRAINT IF EXISTS tenant_lifecycle_events_action_check;

ALTER TABLE tenant_lifecycle_events
ADD CONSTRAINT tenant_lifecycle_events_action_check CHECK (
    action IN (
        'suspend',
        'reactivate',
        'deactivate',
        'schedule_delete',
        'cancel_delete',
        'delete'
    )
);
