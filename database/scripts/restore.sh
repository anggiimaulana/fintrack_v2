#!/usr/bin/env bash
# ============================================================
# restore.sh — restore a dump into a target database.
#   database/scripts/restore.sh [--db TARGET] [--file PATH]
#
# Defaults: --db fintrack, --file database/dump/fintrack_v2.dump
# The target database must already exist.
# ============================================================
set -euo pipefail

# --- locate pg_restore if not on PATH (Windows-friendly) ---
if ! command -v pg_restore >/dev/null 2>&1; then
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin "C:/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin /opt/homebrew/opt/postgresql*/bin; do
        [[ -x "$d/pg_restore" || -x "$d/pg_restore.exe" ]] && export PATH="$d:$PATH" && break
    done
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5435}"
PGDATABASE="${PGDATABASE:-fintrack}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"
DUMP_FILE="${DB_ROOT}/dump/fintrack_v2.dump"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --db)   PGDATABASE="$2"; shift 2 ;;
        --file) DUMP_FILE="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

export PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD

if [[ ! -f "${DUMP_FILE}" ]]; then
    echo "Dump not found: ${DUMP_FILE}" >&2
    exit 1
fi

# --- fall back to the running db container when no local pg_restore exists ---
DB_CONTAINER="${DB_CONTAINER:-fintrack_v2_db}"
RESTORE_VIA="local"
if ! command -v pg_restore >/dev/null 2>&1; then
    RESTORE_VIA=""
    for candidate in docker docker.exe "/mnt/c/Program Files/Docker/Docker/resources/bin/docker.exe"; do
        if command -v "${candidate}" >/dev/null 2>&1 && "${candidate}" exec "${DB_CONTAINER}" pg_restore --version >/dev/null 2>&1; then
            RESTORE_VIA="${candidate}"
            break
        fi
    done
    if [[ -z "${RESTORE_VIA}" ]]; then
        echo "pg_restore not found locally and container '${DB_CONTAINER}' is not reachable" >&2
        exit 1
    fi
fi

echo "==> Restoring ${DUMP_FILE} -> ${PGDATABASE} (via ${RESTORE_VIA})"
if [[ "${RESTORE_VIA}" == "local" ]]; then
    pg_restore --clean --if-exists --no-owner --no-privileges -d "${PGDATABASE}" "${DUMP_FILE}"
else
    DEST_FILE="${DUMP_FILE}"
    if command -v wslpath >/dev/null 2>&1; then
        DEST_FILE="$(wslpath -w "${DUMP_FILE}")"
    fi
    "${RESTORE_VIA}" cp "${DEST_FILE}" "${DB_CONTAINER}:/tmp/restore.dump"
    "${RESTORE_VIA}" exec "${DB_CONTAINER}" pg_restore -U "${PGUSER}" --clean --if-exists --no-owner --no-privileges -d "${PGDATABASE}" /tmp/restore.dump
    "${RESTORE_VIA}" exec "${DB_CONTAINER}" rm -f /tmp/restore.dump
fi
echo "==> Restore complete."
