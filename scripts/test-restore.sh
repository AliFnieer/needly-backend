#!/bin/bash
# Test restore script — validates a backup can be restored to a test database
# Usage: ./scripts/test-restore.sh <backup_file>
#
# Creates a temporary test database, restores the backup, runs basic validation,
# then drops the test database.

set -euo pipefail

BACKUP_FILE="${1:?Usage: ./scripts/test-restore.sh <backup_file>}"
TEST_DB="needly_restore_test_$$"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"

echo "[$(date -Iseconds)] Testing backup restore: ${BACKUP_FILE}"

# Create test database
PGPASSWORD="${DB_PASSWORD:-}" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
  -c "CREATE DATABASE ${TEST_DB};" 2>/dev/null

# Restore
if [[ "$BACKUP_FILE" == *.gz ]]; then
  gunzip -c "$BACKUP_FILE" | PGPASSWORD="${DB_PASSWORD:-}" psql \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB" --quiet 2>/dev/null
else
  PGPASSWORD="${DB_PASSWORD:-}" psql \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB" --quiet \
    -f "$BACKUP_FILE" 2>/dev/null
fi

# Validate
TABLE_COUNT=$(PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
  -d "$TEST_DB" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')

echo "[$(date -Iseconds)] Restored ${TABLE_COUNT} tables"

# Cleanup
PGPASSWORD="${DB_PASSWORD:-}" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
  -c "DROP DATABASE IF EXISTS ${TEST_DB};" 2>/dev/null

if [[ "$TABLE_COUNT" -gt 0 ]]; then
  echo "[$(date -Iseconds)] Backup validation PASSED (${TABLE_COUNT} tables restored)"
  exit 0
else
  echo "[$(date -Iseconds)] Backup validation FAILED (no tables found)"
  exit 1
fi
