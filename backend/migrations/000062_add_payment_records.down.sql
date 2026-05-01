-- Migration: 000062_add_payment_records.down.sql
-- Purpose: Rollback payment_records table

DROP TABLE IF EXISTS payment_records CASCADE;