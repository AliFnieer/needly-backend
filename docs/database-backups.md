# Database Backup and Recovery

This guide covers backup procedures, scheduling, restoration, and verification for the Needly PostgreSQL database.

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Automated Backup Script](#automated-backup-script)
- [Manual Backup](#manual-backup)
- [Cron Schedule](#cron-schedule)
- [Restore Procedure](#restore-procedure)
- [Point-in-Time Recovery](#point-in-time-recovery)
- [Backup Verification](#backup-verification)
- [Storage and Retention](#storage-and-retention)

---

## Overview

| Component        | Details                                      |
| ---------------- | -------------------------------------------- |
| Database         | PostgreSQL 16                                |
| Backup tool      | `pg_dump` (via `scripts/backup.sh`)          |
| Format           | Custom format (`.dump`) — compressed, restore-ready |
| Default schedule | Daily at 2:00 AM, weekly full on Sundays     |
| Retention        | 30 days for daily, 12 weeks for weekly       |

Backups use PostgreSQL's `pg_dump` in custom format, which produces compressed, flexible backup files that can be restored to any PostgreSQL 16+ server.

---

## Prerequisites

- `pg_dump` and `pg_restore` must be available (installed with PostgreSQL client tools).
- Database credentials and connection details from your `.env`:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=needly
```

- Sufficient disk space in the backup directory.

---

## Automated Backup Script

The project includes a backup script at `scripts/backup.sh`.

### Setup

```bash
# Make the script executable
chmod +x scripts/backup.sh
```

### What it does

1. Creates a timestamped backup directory under `backups/`.
2. Runs `pg_dump` in custom format (`-Fc`) for compression and restore flexibility.
3. Creates a symlink `latest.dump` pointing to the most recent backup.
4. Prunes backups older than the configured retention period.
5. Logs all operations to `backups/backup.log`.

### Run manually

```bash
./scripts/backup.sh
```

### Configuration

Edit the top of `scripts/backup.sh` to customize:

```bash
BACKUP_DIR="backups"
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE:-disable}"
RETENTION_DAYS=30
```

---

## Manual Backup

### Full database backup

```bash
pg_dump \
  -h localhost \
  -p 5432 \
  -U postgres \
  -d needly \
  -Fc \
  -f backups/manual_$(date +%Y%m%d_%H%M%S).dump
```

### Backup a specific table

```bash
pg_dump \
  -h localhost -p 5432 -U postgres -d needly \
  -Fc \
  -t users \
  -f backups/users_only.dump
```

### Backup schema only (no data)

```bash
pg_dump \
  -h localhost -p 5432 -U postgres -d needly \
  --schema-only \
  -f backups/schema_only.sql
```

### SQL format backup (human-readable)

```bash
pg_dump \
  -h localhost -p 5432 -U postgres -d needly \
  -Fp \
  -f backups/needly_$(date +%Y%m%d).sql
```

---

## Cron Schedule

### Recommended cron entries

Add to the server's crontab (`crontab -e`):

```cron
# Daily backup at 2:00 AM
0 2 * * * /path/to/needly-backend/scripts/backup.sh >> /var/log/needly-backup.log 2>&1

# Weekly full backup on Sundays at 3:00 AM (using pg_dumpall for roles + globals)
0 3 * * 0 pg_dumpall -h localhost -p 5432 -U postgres > /path/to/backups/weekly_full_$(date +\%Y\%m\%d).sql 2>> /var/log/needly-backup.log
```

### Backup file naming convention

```
backups/
├── daily/
│   ├── needly_20260820_020000.dump
│   ├── needly_20260821_020000.dump
│   └── ...
├── weekly/
│   ├── weekly_full_20260817.sql
│   └── ...
├── latest.dump -> daily/needly_20260820_020000.dump
└── backup.log
```

---

## Restore Procedure

### Restore from a `.dump` backup

```bash
# Stop the API server first
docker compose stop api

# Drop and recreate the database
dropdb -h localhost -p 5432 -U postgres needly
createdb -h localhost -p 5432 -U postgres needly

# Restore from backup
pg_restore \
  -h localhost \
  -p 5432 \
  -U postgres \
  -d needly \
  --verbose \
  --clean \
  --if-exists \
  backups/daily/needly_20260820_020000.dump

# Restart the API server
docker compose start api
```

### Restore from a `.sql` backup

```bash
# Stop the API server
docker compose stop api

# Drop and recreate
dropdb -h localhost -p 5432 -U postgres needly
createdb -h localhost -p 5432 -U postgres needly

# Restore
psql -h localhost -p 5432 -U postgres -d needly -f backups/weekly_full_20260817.sql

# Restart
docker compose start api
```

### Restore to a different database name

```bash
createdb -h localhost -p 5432 -U postgres needly_restore
pg_restore -h localhost -p 5432 -U postgres -d needly_restore backups/daily/needly_20260820_020000.dump
```

---

## Point-in-Time Recovery

For point-in-time recovery, PostgreSQL must have WAL (Write-Ahead Log) archiving enabled.

### Enable WAL archiving

Add to `postgresql.conf` (or via your managed database service):

```conf
wal_level = replica
archive_mode = on
archive_command = 'test ! -f /path/to/wal_archive/%f && cp %p /path/to/wal_archive/%f'
```

### Restore to a specific point in time

```bash
# 1. Restore the most recent base backup
pg_restore -h localhost -p 5432 -U postgres -d needly --clean backups/daily/needly_latest.dump

# 2. Create a recovery signal file and configure recovery
# In postgresql.conf:
# restore_command = 'cp /path/to/wal_archive/%f %p'
# recovery_target_time = '2026-08-20 14:30:00'
# recovery_target_action = 'promote'

# 3. Restart PostgreSQL to begin recovery
```

> **Note:** Point-in-time recovery is only possible if WAL archiving was enabled before the backup was created. For most deployments, rely on regular `pg_dump` backups and apply application-level idempotency for any lost writes.

---

## Backup Verification

Always verify backups after creation. The backup script runs automatic verification when `VERIFY_BACKUPS=true`.

### Manual verification

```bash
# Check that the backup file exists and is not empty
ls -lh backups/daily/needly_20260820_020000.dump

# Verify backup integrity (pg_dump custom format)
pg_restore -l backups/daily/needly_20260820_020000.dump | head -20

# Restore to a temporary database and run basic queries
createdb -h localhost -p 5432 -U postgres needly_verify
pg_restore -h localhost -p 5432 -U postgres -d needly_verify backups/daily/needly_20260820_020000.dump

# Run verification queries
psql -h localhost -p 5432 -U postgres -d needly_verify -c "
SELECT
  (SELECT COUNT(*) FROM users) AS users,
  (SELECT COUNT(*) FROM households) AS households,
  (SELECT COUNT(*) FROM shopping_lists) AS lists,
  (SELECT COUNT(*) FROM shopping_items) AS items;
"

# Clean up
dropdb -h localhost -p 5432 -U postgres needly_verify
```

### Automated verification script

Add to `scripts/backup.sh` or run separately:

```bash
#!/bin/bash
BACKUP_FILE="$1"
VERIFY_DB="needly_verify_$(date +%s)"

createdb -h localhost -p 5432 -U postgres "$VERIFY_DB"
pg_restore -h localhost -p 5432 -U postgres -d "$VERIFY_DB" "$BACKUP_FILE" 2>/dev/null

TABLE_COUNT=$(psql -h localhost -p 5432 -U postgres -d "$VERIFY_DB" -t -c \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")

dropdb -h localhost -p 5432 -U postgres "$VERIFY_DB"

if [ "$TABLE_COUNT" -gt 0 ]; then
  echo "PASS: Backup verified ($TABLE_COUNT tables found)"
  exit 0
else
  echo "FAIL: Backup verification failed"
  exit 1
fi
```

---

## Storage and Retention

### Retention policy

| Backup Type | Frequency | Retention   | Location        |
| ----------- | --------- | ----------- | --------------- |
| Daily       | 2:00 AM   | 30 days     | `backups/daily/`  |
| Weekly      | Sunday 3:00 AM | 12 weeks | `backups/weekly/` |
| Manual      | On-demand | Indefinite  | `backups/manual/` |

### Disk space estimation

| Data Size | Compression Ratio | `.dump` Size | 30 Days of Daily Backups |
| --------- | ----------------- | ------------ | ------------------------ |
| 100 MB    | ~5:1              | ~20 MB       | ~600 MB                  |
| 500 MB    | ~5:1              | ~100 MB      | ~3 GB                    |
| 1 GB      | ~4:1              | ~250 MB      | ~7.5 GB                  |

### Off-site backups

For production, copy backups to external storage:

```bash
# S3 sync (after daily backup)
aws s3 sync backups/daily/ s3://needly-backups/daily/ --storage-class STANDARD_IA

# DigitalOcean Spaces
doctl spaces sync backups/daily/ needly-backups/daily/
```

Add to cron:

```cron
0 4 * * * aws s3 sync /path/to/backups/daily/ s3://needly-backups/daily/ >> /var/log/needly-backup.log 2>&1
```

---

## Quick Reference

| Task                       | Command                                                              |
| -------------------------- | -------------------------------------------------------------------- |
| Manual backup              | `./scripts/backup.sh`                                                |
| Restore latest             | `pg_restore -h localhost -p 5432 -U postgres -d needly --clean backups/latest.dump` |
| List backup contents       | `pg_restore -l backups/daily/<file>.dump`                            |
| Check backup size          | `ls -lh backups/daily/`                                              |
| View backup log            | `tail -50 backups/backup.log`                                        |
| Run backup verification    | `./scripts/verify_backup.sh backups/daily/<file>.dump`              |
