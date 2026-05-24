#!/bin/bash
# newapi status — Show container status
set -eo pipefail

echo "[newapi] Checking status..."
echo "  Home:         ${NEWAPI_HOME:-/opt/newapi}"
echo "  Compose:      ${NEWAPI_DOCKER_COMPOSE_CMD:-docker compose}"

# TODO: V3.0 will replace this with Go docker.Client.ContainerList()
echo "[newapi] Status stub complete."
