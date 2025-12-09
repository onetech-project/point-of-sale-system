#!/usr/bin/env bash
# Wrapper to run cleanup_event_records.sql against DATABASE_URL
set -e

DB_URL=${DATABASE_URL:-"postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable"}
SQL_FILE="$(dirname "$0")/cleanup_event_records.sql"

if command -v psql >/dev/null 2>&1; then
  echo "Running cleanup query against $DB_URL"
  PGPASSWORD=$(echo "$DB_URL" | sed -n 's/.*:\/\/[^:]*:\([^@]*\)@.*/\1/p')
  # Note: for local runs, prefer setting DATABASE_URL and using a proper psql connection string
  psql "$DB_URL" -f "$SQL_FILE"
else
  echo "psql not found in PATH. Please run the SQL script manually against your database."
  exit 1
fi
