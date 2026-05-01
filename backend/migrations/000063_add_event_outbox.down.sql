-- Migration: 000063_add_event_outbox.down.sql
-- Purpose: Rollback event_outbox table

DROP TABLE IF EXISTS event_outbox CASCADE;