#!/bin/bash
# newapi doctor — Run diagnostic checks
set -eo pipefail

echo "[newapi] Running diagnostics..."
echo "  Home:         ${NEWAPI_HOME:-/opt/newapi}"
echo "  Compose:      ${NEWAPI_DOCKER_COMPOSE_CMD:-docker compose}"

# TODO: V3.0 will replace this with Go health check logic
echo "[newapi] Doctor stub complete."
