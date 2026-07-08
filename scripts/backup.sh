#!/usr/bin/env bash
# Backs up Postgres (pg_dump, custom format) and MinIO objects (tar of
# the bucket) from the running compose stack into ./backups/<timestamp>/.
#
# Usage: bash scripts/backup.sh [compose-file]
set -euo pipefail

COMPOSE_FILE="${1:-docker-compose.yml}"
STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="backups/$STAMP"
mkdir -p "$DEST"

echo "==> Backing up Postgres"
docker compose -f "$COMPOSE_FILE" exec -T postgres \
  pg_dump -U pos -d pos --format=custom > "$DEST/pos.dump"

echo "==> Backing up MinIO objects"
# The minio image has no tar; read its volume from a throwaway alpine.
MINIO_VOLUME="$(docker volume ls -q | grep -m1 'minio-data')"
# MSYS_NO_PATHCONV keeps Git Bash on Windows from mangling /data.
MSYS_NO_PATHCONV=1 docker run --rm -v "$MINIO_VOLUME":/data alpine \
  tar -czf - -C /data . > "$DEST/minio-data.tar.gz"

echo "==> Done: $DEST"
ls -lh "$DEST"

# Restore notes:
#   Postgres: docker compose exec -T postgres pg_restore -U pos -d pos --clean < backups/<ts>/pos.dump
#   MinIO:    docker compose exec -T minio tar -xzf - -C /data < backups/<ts>/minio-data.tar.gz
