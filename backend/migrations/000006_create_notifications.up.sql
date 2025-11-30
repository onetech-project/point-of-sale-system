-- Migration: 000006_create_notifications
-- Description: Create notifications table for audit log
-- Author: CTO Hero Mode
-- Date: 2025-11-23

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email', 'sms', 'push')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    event_type VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    body TEXT NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_msg TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_notifications_tenant_id ON notifications(tenant_id);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_event_type ON notifications(event_type);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX idx_notifications_recipient ON notifications(recipient);

-- Comments for documentation
COMMENT ON TABLE notifications IS 'Notification audit log for all emails/SMS sent';
COMMENT ON COLUMN notifications.event_type IS 'Event that triggered notification (e.g., user.registered, user.login)';
COMMENT ON COLUMN notifications.metadata IS 'Additional event data (JSON)';
COMMENT ON COLUMN notifications.retry_count IS 'Number of send attempts (max 3)';
