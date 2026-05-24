#!/bin/bash
# newapi install — Deploy new-api with Docker Compose
set -eo pipefail

echo "[newapi] Installing new-api..."
echo "  Home:         ${NEWAPI_HOME:-/opt/newapi}"
echo "  Port:         ${NEWAPI_PORT:-3000}"
echo "  Image:        ${NEWAPI_DOCKER_IMAGE:-calciumion/new-api:latest}"
echo "  Compose:      ${NEWAPI_DOCKER_COMPOSE_CMD:-docker compose}"

# TODO: V3.0 will replace this with Go docker.ComposeUp()
echo "[newapi] Install stub complete."
