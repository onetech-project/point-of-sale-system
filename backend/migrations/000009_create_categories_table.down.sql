-- Rollback categories table creation
DROP TRIGGER IF EXISTS trg_categories_updated_at ON categories;
DROP POLICY IF EXISTS tenant_isolation ON categories;
DROP TABLE IF EXISTS categories CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();
