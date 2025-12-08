-- Add charge_delivery_fee column to order_settings
-- This allows tenants to disable system delivery fee collection
-- when they handle delivery fees externally (e.g., third-party delivery services)
ALTER TABLE order_settings
ADD COLUMN IF NOT EXISTS charge_delivery_fee BOOLEAN DEFAULT true;

COMMENT ON COLUMN order_settings.charge_delivery_fee IS 'When false, the system will not charge delivery fees in orders. Useful when tenant uses external delivery services.';