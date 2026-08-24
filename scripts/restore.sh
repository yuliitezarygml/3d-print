#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "Usage: ./scripts/restore.sh backups/file.dump" >&2
  exit 1
fi
if [ ! -f "$1" ]; then
  echo "Backup file not found: $1" >&2
  exit 1
fi

if command -v docker >/dev/null 2>&1; then
  docker compose exec -T postgres pg_restore -U "${POSTGRES_USER:-printforge}" -d "${POSTGRES_DB:-printforge}" --clean --if-exists < "$1"
else
  pg_restore --dbname "$DATABASE_URL" --clean --if-exists "$1"
fi

