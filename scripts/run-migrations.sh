#!/bin/bash

# Run database migrations for POS services.
# Prefers local golang-migrate CLI, falls back to dockerized migrate image.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/backend/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "❌ Migrations directory not found: $MIGRATIONS_DIR"
    exit 1
fi

# Load root environment if available
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

DB_USER=${POSTGRES_USER:-pos_user}
DB_PASSWORD=${POSTGRES_PASSWORD:-pos_password}
DB_NAME=${POSTGRES_DB:-pos_db}
DB_HOST=${POSTGRES_HOST:-localhost}
DB_PORT=${POSTGRES_PORT:-5432}

# Ensure local development host works even if service .env uses docker hostname.
if [ "$DB_HOST" = "postgres" ]; then
    DB_HOST="localhost"
fi

DATABASE_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

apply_event_outbox_schema_patch() {
    echo "  Creating event_outbox table..."

    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -U "$DB_USER" \
        -d "$DB_NAME" <<'SQL'
CREATE TABLE IF NOT EXISTS event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON event_outbox (created_at)
WHERE published_at IS NULL;

COMMENT ON TABLE event_outbox IS 'Transactional outbox for reliable Kafka event publishing';
COMMENT ON COLUMN event_outbox.event_type IS 'Event type identifier (e.g., offline_order.created)';
COMMENT ON COLUMN event_outbox.event_key IS 'Kafka partition key (e.g., order_id for ordering)';
COMMENT ON COLUMN event_outbox.event_payload IS 'Full event payload as JSON';
COMMENT ON COLUMN event_outbox.topic IS 'Target Kafka topic (e.g., offline-orders-audit)';
COMMENT ON COLUMN event_outbox.published_at IS 'Timestamp when successfully published to Kafka (NULL = pending)';
SQL
}

apply_offline_orders_schema_patch() {
    echo "  Patching guest_orders table for offline orders..."

    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -U "$DB_USER" \
        -d "$DB_NAME" <<'SQL'
ALTER TABLE guest_orders
ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'online',
ADD COLUMN IF NOT EXISTS data_consent_given BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS consent_method VARCHAR(20),
ADD COLUMN IF NOT EXISTS recorded_by_user_id UUID REFERENCES users (id),
ADD COLUMN IF NOT EXISTS last_modified_by_user_id UUID REFERENCES users (id),
ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS deleted_by_user_id UUID REFERENCES users (id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'check_order_type'
    ) THEN
        ALTER TABLE guest_orders
        ADD CONSTRAINT check_order_type CHECK (order_type IN ('online', 'offline'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'check_consent_method'
    ) THEN
        ALTER TABLE guest_orders
        ADD CONSTRAINT check_consent_method CHECK (
            consent_method IS NULL
            OR consent_method IN ('verbal', 'written', 'digital')
        );
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_guest_orders_type_status
ON guest_orders (order_type, status, tenant_id);

CREATE INDEX IF NOT EXISTS idx_guest_orders_recorded_by
ON guest_orders (recorded_by_user_id)
WHERE order_type = 'offline';

CREATE INDEX IF NOT EXISTS idx_offline_orders_pending_payment
ON guest_orders (tenant_id, created_at DESC)
WHERE order_type = 'offline' AND status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_guest_orders_offline_deleted
ON guest_orders (tenant_id, deleted_at)
WHERE order_type = 'offline' AND deleted_at IS NOT NULL;
SQL

    echo "✅ Offline-order schema patch applied"
}

apply_offline_payments_schema_patch() {
    echo "  Creating payment terms and payment records tables..."

    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -U "$DB_USER" \
        -d "$DB_NAME" <<'SQL'
CREATE TABLE IF NOT EXISTS payment_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES guest_orders(id) ON DELETE CASCADE,
    total_amount INTEGER NOT NULL CHECK (total_amount > 0),
    down_payment_amount INTEGER CHECK (
        down_payment_amount >= 0
        AND down_payment_amount < total_amount
    ),
    installment_count INTEGER CHECK (installment_count >= 0),
    installment_amount INTEGER CHECK (installment_amount >= 0),
    payment_schedule JSONB,
    total_paid INTEGER NOT NULL DEFAULT 0 CHECK (
        total_paid >= 0
        AND total_paid <= total_amount
    ),
    remaining_balance INTEGER NOT NULL CHECK (remaining_balance >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    CONSTRAINT check_payment_structure
        CHECK (
            (down_payment_amount IS NULL AND installment_count = 0) OR
            (down_payment_amount >= 0 AND installment_count > 0)
        ),
    CONSTRAINT check_remaining_balance
        CHECK (remaining_balance = total_amount - total_paid)
);

CREATE TABLE IF NOT EXISTS payment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL,
    payment_number INTEGER NOT NULL,
    amount_paid INTEGER NOT NULL CHECK (amount_paid > 0),
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    payment_method VARCHAR(50) NOT NULL,
    remaining_balance_after INTEGER NOT NULL CHECK (remaining_balance_after >= 0),
    recorded_by_user_id UUID NOT NULL REFERENCES users (id),
    notes TEXT,
    receipt_number VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT check_payment_method
        CHECK (payment_method IN ('cash', 'card', 'bank_transfer', 'check', 'other'))
);

CREATE INDEX IF NOT EXISTS idx_payment_terms_order_id ON payment_terms (order_id);
CREATE INDEX IF NOT EXISTS idx_payment_terms_balance ON payment_terms (remaining_balance, order_id)
WHERE remaining_balance > 0;
CREATE INDEX IF NOT EXISTS idx_payment_records_order_id ON payment_records (order_id, payment_date DESC);
CREATE INDEX IF NOT EXISTS idx_payment_records_date ON payment_records (payment_date DESC);
CREATE INDEX IF NOT EXISTS idx_payment_records_recorded_by ON payment_records (recorded_by_user_id);
SQL

    echo "✅ Offline payment schema patch applied"
}

run_migrate() {
    local output
    local exit_code
    set +e
    output="$($@ 2>&1)"
    exit_code=$?
    set -e

    if [ $exit_code -eq 0 ]; then
        echo "$output"
        return 0
    fi

    echo "$output"

    if echo "$output" | grep -qi "duplicate migration file"; then
        echo "⚠️  Falling back to targeted schema patches..."
        apply_event_outbox_schema_patch
        apply_offline_orders_schema_patch
        apply_offline_payments_schema_patch
        echo "✅ Schema patches applied"
        return 0
    fi

    return $exit_code
}

echo "🗃️  Applying database migrations..."
echo "   DB: ${DB_NAME}@${DB_HOST}:${DB_PORT}"

if command -v migrate >/dev/null 2>&1; then
    run_migrate migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    echo "✅ Migration step completed using local migrate CLI"
    exit 0
fi

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    run_migrate docker run --rm \
        --network host \
        -v "$MIGRATIONS_DIR:/migrations" \
        migrate/migrate:v4.18.3 \
        -path=/migrations \
        -database "$DATABASE_URL" \
        up
    echo "✅ Migration step completed using dockerized migrate"
    exit 0
fi

# Last-resort fallback for local dev when migrate tooling is unavailable.
echo "⚠️  Falling back to targeted schema patches..."
apply_event_outbox_schema_patch
apply_offline_orders_schema_patch
apply_offline_payments_schema_patch
echo "✅ Schema patches applied"
