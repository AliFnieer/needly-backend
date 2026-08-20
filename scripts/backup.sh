#!/usr/bin/env bash
set -euo pipefail

# Database backup script for Needly
# Usage: ./scripts/backup.sh [daily|weekly|manual]

BACKUP_TYPE="${1:-manual}"
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/needly_${BACKUP_TYPE}_${TIMESTAMP}.sql.gz"

# Database connection (from env or defaults)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-needly}"

mkdir -p "${BACKUP_DIR}"

echo "Starting ${BACKUP_TYPE} backup of ${DB_NAME}..."

pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" \
  --no-owner --no-privileges | gzip > "${BACKUP_FILE}"

echo "Backup saved to ${BACKUP_FILE}"

# Cleanup old backups (keep 7 daily, 4 weekly)
find "${BACKUP_DIR}" -name "needly_daily_*" -mtime +7 -delete 2>/dev/null || true
find "${BACKUP_DIR}" -name "needly_weekly_*" -mtime +28 -delete 2>/dev/null || true

echo "Old backups cleaned up."
