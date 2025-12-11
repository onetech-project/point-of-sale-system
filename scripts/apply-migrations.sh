#!/usr/bin/env bash
set -euo pipefail

# Simple migration runner for local development.
# Applies SQL files under backend/migrations in filename order to DATABASE_URL.

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL not set. Please export DATABASE_URL (postgres connection string)."
  exit 1
fi

MIGRATIONS_DIR="$(dirname "$0")/../backend/migrations"

echo "Applying migrations from $MIGRATIONS_DIR to $DATABASE_URL"

for f in $(ls "$MIGRATIONS_DIR"/*.up.sql | sort); do
  echo "Applying $f"
  psql "$DATABASE_URL" -f "$f"
done

echo "Migrations applied."
