-- Migration: Add notification preference columns to users
-- Path: backend/migrations/000024_add_notification_preferences_to_users.up.sql

ALTER TABLE IF EXISTS users
  ADD COLUMN IF NOT EXISTS notification_email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS notification_in_app_enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_users_tenant_notification_email ON users (notification_email_enabled, tenant_id);
