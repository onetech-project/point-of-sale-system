-- Create product import jobs for asynchronous create-only bulk product imports
CREATE TABLE product_import_jobs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  failure_code VARCHAR(64),
  source_file_name VARCHAR(255) NOT NULL,
  source_format VARCHAR(16) NOT NULL,
  source_file_size BIGINT NOT NULL,
  source_file_bytes BYTEA NOT NULL,
  row_count INTEGER NOT NULL DEFAULT 0,
  created_count INTEGER NOT NULL DEFAULT 0,
  error_count INTEGER NOT NULL DEFAULT 0,
  warning_count INTEGER NOT NULL DEFAULT 0,
  errors JSONB NOT NULL DEFAULT '[]'::jsonb,
  warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT product_import_jobs_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
  CONSTRAINT product_import_jobs_format_check CHECK (source_format IN ('csv', 'xlsx')),
  CONSTRAINT product_import_jobs_counts_check CHECK (
    row_count >= 0 AND created_count >= 0 AND error_count >= 0 AND warning_count >= 0
  ),
  CONSTRAINT product_import_jobs_source_size_check CHECK (source_file_size > 0)
);

CREATE INDEX idx_product_import_jobs_tenant_created ON product_import_jobs(tenant_id, created_at DESC);
CREATE INDEX idx_product_import_jobs_tenant_status ON product_import_jobs(tenant_id, status);
CREATE INDEX idx_product_import_jobs_user_created ON product_import_jobs(user_id, created_at DESC);

ALTER TABLE product_import_jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product_import_jobs
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE TRIGGER trg_product_import_jobs_updated_at
  BEFORE UPDATE ON product_import_jobs
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
