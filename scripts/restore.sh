#!/bin/bash
# PostgreSQL restore script for Needly
# Usage: ./scripts/restore.sh <backup_file>
#
# Environment variables:
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

set -euo pipefail

BACKUP_FILE="${1:?Usage: ./scripts/restore.sh <backup_file>}"

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "Error: Backup file not found: $BACKUP_FILE"
  exit 1
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-needly}"

echo "WARNING: This will DROP and recreate the database '${DB_NAME}'."
echo "Source: ${BACKUP_FILE}"
read -p "Are you sure? (yes/no): " CONFIRM
if [[ "$CONFIRM" != "yes" ]]; then
  echo "Aborted."
  exit 0
fi

echo "[$(date -Iseconds)] Dropping and recreating database..."
PGPASSWORD="${DB_PASSWORD}" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
  -d postgres \
  -c "DROP DATABASE IF EXISTS ${DB_NAME};" \
  -c "CREATE DATABASE ${DB_NAME};"

echo "[$(date -Iseconds)] Restoring from ${BACKUP_FILE}..."
if [[ "$BACKUP_FILE" == *.gz ]]; then
  gunzip -c "$BACKUP_FILE" | PGPASSWORD="${DB_PASSWORD}" psql \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" --quiet
else
  PGPASSWORD="${DB_PASSWORD}" psql \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" --quiet \
    -f "$BACKUP_FILE"
fi

echo "[$(date -Iseconds)] Restore complete."
