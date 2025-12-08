-- Remove charge_delivery_fee column from order_settings
ALTER TABLE order_settings DROP COLUMN IF EXISTS charge_delivery_fee;