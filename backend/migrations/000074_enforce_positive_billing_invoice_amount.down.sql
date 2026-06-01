-- Migration: 000074_enforce_positive_billing_invoice_amount

ALTER TABLE billing_invoices
DROP CONSTRAINT IF EXISTS billing_invoices_amount_idr_positive;
