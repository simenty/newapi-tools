#!/bin/bash
# newapi backup — Backup new-api data
set -eo pipefail

BACKUP_DIR="${NEWAPI_BACKUP_DIR:-/opt/newapi/backups}"
echo "[newapi] Backing up to ${BACKUP_DIR}..."
echo "[newapi] Backup stub complete."
