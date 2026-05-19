DROP INDEX IF EXISTS idx_tenant_terms_acceptances_tenant;
DROP TABLE IF EXISTS tenant_terms_acceptances;

ALTER TABLE order_items
DROP CONSTRAINT IF EXISTS order_items_product_id_fkey;

ALTER TABLE order_items
ADD CONSTRAINT order_items_product_id_fkey
FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM order_items WHERE product_id IS NULL) THEN
        ALTER TABLE order_items ALTER COLUMN product_id SET NOT NULL;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_tenants_subscription_retention_due;

ALTER TABLE tenants
DROP COLUMN IF EXISTS subscription_data_anonymized_at,
DROP COLUMN IF EXISTS subscription_retention_started_at;
