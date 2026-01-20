-- Migration: 000037_add_guest_orders_anonymization_flag
-- Purpose: Rollback anonymization tracking columns

DROP INDEX IF EXISTS idx_guest_orders_anonymized;

ALTER TABLE guest_orders
DROP COLUMN IF EXISTS anonymized_at,
DROP COLUMN IF EXISTS is_anonymized;