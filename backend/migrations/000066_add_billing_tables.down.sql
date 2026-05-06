DROP INDEX IF EXISTS idx_billing_payment_attempts_invoice_id;

DROP TABLE IF EXISTS billing_payment_attempts;

DROP INDEX IF EXISTS idx_billing_invoices_status;

DROP INDEX IF EXISTS idx_billing_invoices_tenant_id;

DROP TABLE IF EXISTS billing_invoices;

DROP INDEX IF EXISTS idx_tenants_subscription_status;

ALTER TABLE tenants DROP COLUMN IF EXISTS subscription_status;