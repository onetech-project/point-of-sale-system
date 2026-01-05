-- Migration 000031: Drop audit_events table and partitions
DROP TABLE IF EXISTS audit_events CASCADE;