#!/usr/bin/env bash
# scripts/db_backup_sqlite.sh
# SQLite automated backup script for bgpsql (DATA-7)
# Run via daily cron or systemd timer.

set -euo pipefail

DB_PATH="${BGPSQL_DB_PATH:-/var/lib/bgpsql/bgp.db}"
GCS_BUCKET="${BGPSQL_GCS_BUCKET:-gs://bgpinfo}"
NODE_NAME="${BGPSQL_NODE_NAME:-$(hostname -s)}"
DATE_STR="$(date +%F)"
BACKUP_TMP="/var/tmp/bgp-${NODE_NAME}-${DATE_STR}.db"
ZSTD_TMP="${BACKUP_TMP}.zst"

cleanup() {
    rm -f "${BACKUP_TMP}" "${ZSTD_TMP}"
}
trap cleanup EXIT ERR INT TERM

echo "[$(date -u -Iseconds)] Starting SQLite backup of ${DB_PATH} on ${NODE_NAME}..."

if [ ! -f "${DB_PATH}" ]; then
    echo "ERROR: Database file ${DB_PATH} not found!" >&2
    exit 1
fi

# 1. Hot online backup using SQLite's online backup API (.backup command)
sqlite3 "${DB_PATH}" ".backup '${BACKUP_TMP}'"

# 2. Verify integrity of the backup copy
CHECK_RESULT=$(sqlite3 -readonly "${BACKUP_TMP}" "PRAGMA integrity_check;")
if [ "${CHECK_RESULT}" != "ok" ]; then
    echo "ERROR: SQLite integrity check failed on ${BACKUP_TMP}: ${CHECK_RESULT}" >&2
    exit 1
fi
echo "[$(date -u -Iseconds)] Integrity check OK."

# 3. Compress using zstd -19
zstd -19 -f "${BACKUP_TMP}" -o "${ZSTD_TMP}"

# 4. Upload to GCS using gcloud storage cp
DEST_URI="${GCS_BUCKET}/db-${NODE_NAME}-${DATE_STR}.db.zst"
gcloud storage cp "${ZSTD_TMP}" "${DEST_URI}"

echo "[$(date -u -Iseconds)] Successfully uploaded backup to ${DEST_URI}"
