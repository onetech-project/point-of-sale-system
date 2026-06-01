-- Migration: 000074_enforce_positive_billing_invoice_amount
-- Description: Prevent new zero-value subscription invoices while allowing existing rows during rollout.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'billing_invoices_amount_idr_positive'
    ) THEN
        ALTER TABLE billing_invoices
        ADD CONSTRAINT billing_invoices_amount_idr_positive
        CHECK (amount_idr > 0) NOT VALID;
    END IF;
END $$;
