-- Migration: 000075_guest_orders_timestamptz
-- Description: Store guest_orders timestamps as TIMESTAMPTZ so every reader and writer agrees on the instant.
--
-- The columns were TIMESTAMP (without time zone) and were written from two sources:
--   * Postgres NOW()/CURRENT_TIMESTAMP, which stores UTC wall-clock time (session timezone is UTC):
--     created_at of online orders (column default) and anonymized_at of tenant data cleanup
--     (billing-service, recognisable by customer_name = 'Anonymized customer')
--   * Go time.Now() from order-service, whose offset is dropped, leaving the service's local
--     wall-clock time (TZ=Asia/Jakarta): created_at of offline orders, paid_at, completed_at,
--     cancelled_at, deleted_at, last_modified_at and anonymized_at of guest data deletion
--
-- The zone used for the Go-written values defaults to Asia/Jakarta. If order-service ran with a
-- different TZ, set it before migrating, e.g.:
--   ALTER DATABASE pos_db SET app.legacy_order_timezone = 'UTC';
-- The migration aborts when that zone disagrees with the offset observed between offline orders
-- and their order_items (whose created_at always comes from NOW()).

DO $$
DECLARE
    legacy_tz TEXT := COALESCE(NULLIF(current_setting('app.legacy_order_timezone', true), ''), 'Asia/Jakarta');
    expected_offset NUMERIC;
    detected_offset NUMERIC;
BEGIN
    IF (
        SELECT data_type
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'guest_orders'
          AND column_name = 'created_at'
    ) = 'timestamp with time zone' THEN
        RAISE NOTICE 'guest_orders timestamps are already TIMESTAMPTZ, skipping';
        RETURN;
    END IF;

    -- Fails fast on an invalid zone name
    expected_offset := EXTRACT(EPOCH FROM ((NOW() AT TIME ZONE legacy_tz) - (NOW() AT TIME ZONE 'UTC'))) / 3600;

    SELECT percentile_disc(0.5) WITHIN GROUP (
        ORDER BY ROUND(EXTRACT(EPOCH FROM (go.created_at - oi.first_item_at)) / 3600)
    )
    INTO detected_offset
    FROM guest_orders go
    JOIN (
        SELECT order_id, MIN(created_at) AS first_item_at
        FROM order_items
        GROUP BY order_id
    ) oi ON oi.order_id = go.id
    WHERE go.order_type = 'offline';

    IF detected_offset IS NOT NULL AND detected_offset <> ROUND(expected_offset) THEN
        RAISE EXCEPTION 'offline orders were written with a UTC offset of % hours, but app.legacy_order_timezone (%) is % hours; set app.legacy_order_timezone to the TZ order-service ran with',
            detected_offset, legacy_tz, ROUND(expected_offset);
    END IF;

    EXECUTE format($sql$
        ALTER TABLE guest_orders
            ALTER COLUMN created_at DROP DEFAULT,
            ALTER COLUMN created_at TYPE TIMESTAMPTZ USING (
                CASE
                    WHEN order_type = 'offline' THEN created_at AT TIME ZONE %1$L
                    ELSE created_at AT TIME ZONE 'UTC'
                END
            ),
            ALTER COLUMN created_at SET DEFAULT NOW(),
            ALTER COLUMN paid_at TYPE TIMESTAMPTZ USING paid_at AT TIME ZONE %1$L,
            ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at AT TIME ZONE %1$L,
            ALTER COLUMN cancelled_at TYPE TIMESTAMPTZ USING cancelled_at AT TIME ZONE %1$L,
            ALTER COLUMN deleted_at TYPE TIMESTAMPTZ USING deleted_at AT TIME ZONE %1$L,
            ALTER COLUMN last_modified_at TYPE TIMESTAMPTZ USING last_modified_at AT TIME ZONE %1$L,
            ALTER COLUMN anonymized_at TYPE TIMESTAMPTZ USING (
                CASE
                    WHEN customer_name = 'Anonymized customer' THEN anonymized_at AT TIME ZONE 'UTC'
                    ELSE anonymized_at AT TIME ZONE %1$L
                END
            )
    $sql$, legacy_tz);
END $$;

-- Analytics count PAID and COMPLETE orders by the moment they were paid (falling back to completion,
-- then creation for rows missing both)
CREATE INDEX IF NOT EXISTS idx_guest_orders_revenue_date
    ON guest_orders (tenant_id, (COALESCE(paid_at, completed_at, created_at)))
    WHERE status IN ('PAID', 'COMPLETE');
