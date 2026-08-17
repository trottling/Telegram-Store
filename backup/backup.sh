#!/bin/sh
set -e
set -o pipefail

TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
FILENAME="backup_${TIMESTAMP}.sql.gz"
BACKUP_DIR="/backups"

echo "[$(date)] старт бэкапа ${FILENAME}"

PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h db -U "$POSTGRES_USER" -d "$POSTGRES_NAME" | gzip > "${BACKUP_DIR}/${FILENAME}"

echo "[$(date)] локальный дамп готов: ${FILENAME}"

if [ -n "$S3_REMOTE" ] && [ -n "$S3_BUCKET" ]; then
  rclone copy "${BACKUP_DIR}/${FILENAME}" "${S3_REMOTE}:${S3_BUCKET}/"
  echo "[$(date)] загружено в ${S3_REMOTE}:${S3_BUCKET}"
fi

find "$BACKUP_DIR" -name "backup_*.sql.gz" -mtime "+${RETENTION_DAYS:-7}" -delete

echo "[$(date)] бэкап завершён"
