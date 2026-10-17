-- Migration: 000075_guest_orders_timestamptz (rollback)
-- Restores TIMESTAMP columns holding UTC wall-clock time for every row. Re-applying the up migration
-- afterwards requires app.legacy_order_timezone = 'UTC', since no row holds local time anymore.

DROP INDEX IF EXISTS idx_guest_orders_revenue_date;

DO $$
BEGIN
    IF (
        SELECT data_type
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'guest_orders'
          AND column_name = 'created_at'
    ) <> 'timestamp with time zone' THEN
        RETURN;
    END IF;

    ALTER TABLE guest_orders
        ALTER COLUMN created_at DROP DEFAULT,
        ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC',
        ALTER COLUMN created_at SET DEFAULT NOW(),
        ALTER COLUMN paid_at TYPE TIMESTAMP USING paid_at AT TIME ZONE 'UTC',
        ALTER COLUMN completed_at TYPE TIMESTAMP USING completed_at AT TIME ZONE 'UTC',
        ALTER COLUMN cancelled_at TYPE TIMESTAMP USING cancelled_at AT TIME ZONE 'UTC',
        ALTER COLUMN deleted_at TYPE TIMESTAMP USING deleted_at AT TIME ZONE 'UTC',
        ALTER COLUMN last_modified_at TYPE TIMESTAMP USING last_modified_at AT TIME ZONE 'UTC',
        ALTER COLUMN anonymized_at TYPE TIMESTAMP USING anonymized_at AT TIME ZONE 'UTC';
END $$;
