#!/bin/bash
# newapi config — Manage configuration
set -eo pipefail

echo "[newapi] Current configuration:"
echo "  Home:         ${NEWAPI_HOME:-/opt/newapi}"
echo "  Port:         ${NEWAPI_PORT:-3000}"
echo "  Image:        ${NEWAPI_DOCKER_IMAGE:-calciumion/new-api:latest}"
echo "  Backup Dir:   ${NEWAPI_BACKUP_DIR:-/opt/newapi/backups}"
echo "  Compose:      ${NEWAPI_DOCKER_COMPOSE_CMD:-docker compose}"
echo "[newapi] Config stub complete."
