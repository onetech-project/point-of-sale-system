-- Migration 000050 down: Restore foreign key constraints to consent_records

-- Note: This will fail if there are records with invalid references
-- Data must be cleaned up first before reverting

ALTER TABLE consent_records
ADD CONSTRAINT consent_records_subject_id_fkey FOREIGN KEY (subject_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE consent_records
ADD CONSTRAINT consent_records_guest_order_id_fkey FOREIGN KEY (guest_order_id) REFERENCES guest_orders (id) ON DELETE SET NULL;