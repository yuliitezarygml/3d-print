#!/bin/sh
set -eu

mkdir -p "${BACKUP_DIR:-./backups}"
timestamp="$(date +%Y%m%d_%H%M%S)"
target="${BACKUP_DIR:-./backups}/printforge_${timestamp}.dump"

if command -v docker >/dev/null 2>&1; then
  docker compose exec -T postgres pg_dump -U "${POSTGRES_USER:-printforge}" -d "${POSTGRES_DB:-printforge}" -Fc > "$target"
else
  pg_dump "$DATABASE_URL" -Fc > "$target"
fi
echo "Backup saved to $target"

