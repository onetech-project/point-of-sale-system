-- GIN index for JSONB queries on transaction_id
CREATE INDEX idx_notifications_order_metadata ON notifications USING GIN (metadata jsonb_path_ops)
WHERE
    event_type LIKE 'order.paid.%';

COMMENT ON INDEX idx_notifications_order_metadata IS 'Fast lookup for order notifications by transaction_id';