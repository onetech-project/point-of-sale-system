-- Drop order_settings table
DROP TRIGGER IF EXISTS update_order_settings_updated_at ON order_settings;
DROP INDEX IF EXISTS idx_order_settings_tenant_id;
DROP TABLE IF EXISTS order_settings;
