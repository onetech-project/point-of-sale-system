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

apply_sql_file() {
    local filename="$1"
    local label="$2"
    local migration_file="$MIGRATIONS_DIR/$filename"

    if [ ! -f "$migration_file" ]; then
        echo "❌ Migration file not found: $migration_file"
        return 1
    fi

    echo "  Applying $label..."
    PGPASSWORD="$DB_PASSWORD" psql \
        -v ON_ERROR_STOP=1 \
        -h "$DB_HOST" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "$migration_file"
}

apply_targeted_schema_patches() {
    apply_sql_file "000060_add_offline_orders.up.sql" "offline-order schema"
    apply_sql_file "000061_add_payment_terms.up.sql" "payment terms schema"
    apply_sql_file "000062_add_payment_records.up.sql" "payment records schema"
    apply_sql_file "000063_add_event_outbox.up.sql" "event outbox schema"
    apply_sql_file "000069_platform_owner_flow.up.sql" "platform-owner schema"
    apply_sql_file "000070_platform_command_center.up.sql" "platform command-center schema"
    echo "✅ Schema patches applied"
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
        apply_targeted_schema_patches
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
apply_targeted_schema_patches
