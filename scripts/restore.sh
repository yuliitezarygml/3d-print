#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "Usage: ./scripts/restore.sh backups/printforge_DATE.tar.gz" >&2
  exit 1
fi
if [ ! -f "$1" ]; then
  echo "Backup file not found: $1" >&2
  exit 1
fi

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
case "$1" in
  *.tar.gz|*.tgz) tar xzf "$1" -C "$work_dir"; dump="$work_dir/database.dump" ;;
  *) dump="$1" ;;
esac

if command -v docker >/dev/null 2>&1; then
  docker compose exec -T postgres pg_restore -U "${POSTGRES_USER:-printforge}" -d "${POSTGRES_DB:-printforge}" --clean --if-exists < "$dump"
  if [ -f "$work_dir/uploads.tar.gz" ]; then
    docker run --rm -v printforge_uploads:/target -v "$work_dir:/backup:ro" alpine:3.22 sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf {} + && tar xzf /backup/uploads.tar.gz -C /target'
  fi
else
  pg_restore --dbname "$DATABASE_URL" --clean --if-exists "$dump"
  if [ -f "$work_dir/uploads.tar.gz" ]; then
    mkdir -p "${UPLOAD_DIR:-./uploads}"
    tar xzf "$work_dir/uploads.tar.gz" -C "${UPLOAD_DIR:-./uploads}"
  fi
fi
echo "Restore completed from $1"
