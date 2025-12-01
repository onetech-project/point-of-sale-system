-- Rollback products table creation
DROP TRIGGER IF EXISTS trg_products_updated_at ON products;
DROP POLICY IF EXISTS tenant_isolation ON products;
DROP TABLE IF EXISTS products CASCADE;
