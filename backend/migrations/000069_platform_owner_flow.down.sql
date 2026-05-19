DROP INDEX IF EXISTS idx_tenant_lifecycle_events_action;
DROP INDEX IF EXISTS idx_tenant_lifecycle_events_tenant_id;
DROP TABLE IF EXISTS tenant_lifecycle_events;

DROP INDEX IF EXISTS idx_support_ticket_notes_ticket_id;
DROP TABLE IF EXISTS support_ticket_notes;

DROP INDEX IF EXISTS idx_support_tickets_assignee;
DROP INDEX IF EXISTS idx_support_tickets_status;
DROP INDEX IF EXISTS idx_support_tickets_tenant_id;
DROP TABLE IF EXISTS support_tickets;

DROP INDEX IF EXISTS idx_platform_admin_sessions_expires_at;
DROP INDEX IF EXISTS idx_platform_admin_sessions_admin_id;
DROP INDEX IF EXISTS idx_platform_admin_sessions_session_id;
DROP TABLE IF EXISTS platform_admin_sessions;

DROP INDEX IF EXISTS idx_platform_admins_status;
DROP TABLE IF EXISTS platform_admins;

ALTER TABLE tenants
DROP COLUMN IF EXISTS status_reason,
DROP COLUMN IF EXISTS scheduled_delete_at,
DROP COLUMN IF EXISTS deletion_requested_at,
DROP COLUMN IF EXISTS deactivated_at,
DROP COLUMN IF EXISTS suspended_at;

DROP INDEX IF EXISTS idx_billing_invoices_due_at;

ALTER TABLE billing_invoices
DROP COLUMN IF EXISTS payment_link_expires_at,
DROP COLUMN IF EXISTS due_at;
