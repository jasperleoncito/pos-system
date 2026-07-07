#!/bin/sh
# Bootstraps the MinIO bucket and public-read policy for image prefixes.
set -e

mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"

mc mb --ignore-existing "local/$MINIO_BUCKET"

# Images are served through nginx /storage/ — anonymous download only.
mc anonymous set download "local/$MINIO_BUCKET"

echo "MinIO bucket '$MINIO_BUCKET' ready"
