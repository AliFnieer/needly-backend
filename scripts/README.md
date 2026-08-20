# Backup & Restore Scripts

PostgreSQL backup and restore utilities for the Needly backend.

## Environment Variables

All scripts read these environment variables (defaults in parentheses):

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | *(none)* | PostgreSQL password |
| `DB_NAME` | `needly` | Database name |
| `BACKUP_RETENTION_DAYS` | `7` | Days to keep old backups (backup.sh only) |

## Scripts

### backup.sh

Creates a gzipped SQL dump of the database.

```bash
./scripts/backup.sh [backup_dir]
```

- `backup_dir` defaults to `./backups`
- Automatically deletes backups older than `BACKUP_RETENTION_DAYS`
- Reports file size and remaining backup count

**Example:**
```bash
DB_PASSWORD=secret ./scripts/backup.sh
```

### restore.sh

Restores a backup by dropping and recreating the database.

```bash
./scripts/restore.sh <backup_file>
```

- Prompts for confirmation before proceeding
- Supports both `.sql.gz` and plain `.sql` files

**Example:**
```bash
DB_PASSWORD=secret ./scripts/restore.sh ./backups/needly_20260820_020000.sql.gz
```

### backup-cron.sh

Wrapper for scheduled backups via cron. Sources `.env` from the project root.

```bash
# Add to crontab:
crontab -e

# Run daily at 2 AM:
0 2 * * * /path/to/needly-backend/scripts/backup-cron.sh >> /var/log/needly-backup.log 2>&1
```

### test-restore.sh

Validates a backup by restoring it to a temporary database, counting tables, then cleaning up.

```bash
./scripts/test-restore.sh <backup_file>
```

**Example:**
```bash
DB_PASSWORD=secret ./scripts/test-restore.sh ./backups/needly_20260820_020000.sql.gz
```

## Quick Start

```bash
# Set credentials in .env or export them
export DB_PASSWORD=yourpassword

# Create a backup
./scripts/backup.sh

# Validate the backup
./scripts/test-restore.sh ./backups/needly_*.sql.gz

# Restore (destructive!)
./scripts/restore.sh ./backups/needly_20260820_020000.sql.gz
```
