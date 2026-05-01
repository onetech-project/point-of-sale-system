-- Migration: 000061_add_payment_terms.down.sql
-- Purpose: Rollback payment_terms table

DROP TABLE IF EXISTS payment_terms CASCADE;