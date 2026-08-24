#!/bin/sh
set -eu

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);
SQL

for migration in /migrations/*.sql; do
  version="$(basename "$migration")"
  applied="$(psql "$DATABASE_URL" -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version'")"
  if [ "$applied" = "1" ]; then
    continue
  fi
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 --single-transaction -f "$migration"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(version) VALUES ('$version')"
done

