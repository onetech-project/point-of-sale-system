-- Tenant-level notification configuration
CREATE TABLE IF NOT EXISTS notification_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    tenant_id UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    order_notifications_enabled BOOLEAN NOT NULL DEFAULT true,
    test_mode BOOLEAN NOT NULL DEFAULT false,
    test_email VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id)
);

CREATE INDEX idx_notification_configs_tenant ON notification_configs (tenant_id);

COMMENT ON TABLE notification_configs IS 'Tenant-level notification behavior settings';

COMMENT ON COLUMN notification_configs.test_mode IS 'If true, all emails go to test_email only';