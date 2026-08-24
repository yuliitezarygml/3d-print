#!/bin/sh
set -eu

backup_dir="${BACKUP_DIR:-./backups}"
mkdir -p "$backup_dir"
timestamp="$(date +%Y%m%d_%H%M%S)"
target="$backup_dir/printforge_${timestamp}.tar.gz"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

if command -v docker >/dev/null 2>&1; then
  docker compose exec -T postgres pg_dump -U "${POSTGRES_USER:-printforge}" -d "${POSTGRES_DB:-printforge}" -Fc > "$work_dir/database.dump"
  if docker volume inspect printforge_uploads >/dev/null 2>&1; then
    docker run --rm -v printforge_uploads:/source:ro -v "$work_dir:/backup" alpine:3.22 sh -c 'cd /source && tar czf /backup/uploads.tar.gz .'
  fi
else
  pg_dump "$DATABASE_URL" -Fc > "$work_dir/database.dump"
  if [ "${STORAGE_DRIVER:-local}" = "local" ] && [ -d "${UPLOAD_DIR:-./uploads}" ]; then
    tar czf "$work_dir/uploads.tar.gz" -C "${UPLOAD_DIR:-./uploads}" .
  fi
fi
printf '{"createdAt":"%s","storageDriver":"%s","formatVersion":1}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${STORAGE_DRIVER:-local}" > "$work_dir/manifest.json"
tar czf "$target" -C "$work_dir" database.dump manifest.json $(test -f "$work_dir/uploads.tar.gz" && printf uploads.tar.gz)
echo "Backup saved to $target"
