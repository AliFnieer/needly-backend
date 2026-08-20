#!/bin/bash
# Cron setup for automated backups
# Add to crontab: crontab -e
# Run daily at 2 AM: 0 2 * * * /path/to/needly-backend/scripts/backup-cron.sh >> /var/log/needly-backup.log 2>&1

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Source environment
if [[ -f "${SCRIPT_DIR}/../.env" ]]; then
  set -a
  source "${SCRIPT_DIR}/../.env"
  set +a
fi

exec "${SCRIPT_DIR}/backup.sh" "${SCRIPT_DIR}/../backups"
