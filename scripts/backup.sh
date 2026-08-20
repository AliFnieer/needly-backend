#!/bin/bash
# Automated PostgreSQL backup script for Needly
# Usage: ./scripts/backup.sh [backup_dir]
#
# Environment variables:
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
#   BACKUP_RETENTION_DAYS (default: 7)

set -euo pipefail

BACKUP_DIR="${1:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/needly_${TIMESTAMP}.sql.gz"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-needly}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"

mkdir -p "$BACKUP_DIR"

echo "[$(date -Iseconds)] Starting backup: ${DB_NAME}@${DB_HOST}:${DB_PORT}"

PGPASSWORD="${DB_PASSWORD}" pg_dump \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --no-owner \
  --no-privileges \
  --clean \
  --if-exists \
  | gzip > "$BACKUP_FILE"

FILESIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo "[$(date -Iseconds)] Backup complete: ${BACKUP_FILE} (${FILESIZE})"

# Clean old backups
echo "[$(date -Iseconds)] Cleaning backups older than ${RETENTION_DAYS} days..."
find "$BACKUP_DIR" -name "needly_*.sql.gz" -mtime "+${RETENTION_DAYS}" -delete

REMAINING=$(find "$BACKUP_DIR" -name "needly_*.sql.gz" | wc -l)
echo "[$(date -Iseconds)] ${REMAINING} backup(s) remaining"
