-- Rollback stock_adjustments table creation
DROP TRIGGER IF EXISTS trg_calculate_delta ON stock_adjustments;
DROP FUNCTION IF EXISTS calculate_quantity_delta();
DROP POLICY IF EXISTS tenant_isolation ON stock_adjustments;
DROP TABLE IF EXISTS stock_adjustments CASCADE;
