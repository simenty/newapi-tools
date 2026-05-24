#!/bin/bash
# newapi update — Update to latest image
set -eo pipefail

echo "[newapi] Updating new-api..."
echo "  Image:        ${NEWAPI_DOCKER_IMAGE:-calciumion/new-api:latest}"
echo "  Compose:      ${NEWAPI_DOCKER_COMPOSE_CMD:-docker compose}"

# TODO: V3.0 will replace this with Go docker.ImagePull() + ContainerRestart()
echo "[newapi] Update stub complete."
