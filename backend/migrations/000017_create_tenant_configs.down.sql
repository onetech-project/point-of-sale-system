-- Drop indexes
DROP INDEX IF EXISTS idx_tenant_configs_tenant_id;

-- Drop table
DROP TABLE IF EXISTS tenant_configs CASCADE;