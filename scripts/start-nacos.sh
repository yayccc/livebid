#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="${NACOS_CONTAINER_NAME:-nacos-standalone}"
IMAGE="${NACOS_IMAGE:-nacos/nacos-server:v3.0.3}"
CONSOLE_PORT="${NACOS_CONSOLE_PORT:-8080}"
HTTP_PORT="${NACOS_HTTP_PORT:-8848}"
GRPC_PORT="${NACOS_GRPC_PORT:-9848}"
RAFT_PORT="${NACOS_RAFT_PORT:-9849}"
DATA_VOLUME="${NACOS_DATA_VOLUME:-nacos_data}"
LOGS_VOLUME="${NACOS_LOGS_VOLUME:-nacos_logs}"
AUTH_ENABLE="${NACOS_AUTH_ENABLE:-false}"
AUTH_TOKEN="${NACOS_AUTH_TOKEN:-MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=}"
AUTH_IDENTITY_KEY="${NACOS_AUTH_IDENTITY_KEY:-nacos}"
AUTH_IDENTITY_VALUE="${NACOS_AUTH_IDENTITY_VALUE:-nacos}"

docker_args=(
  run -d
  --name "$CONTAINER_NAME"
  --restart unless-stopped
  -p "$CONSOLE_PORT:8080"
  -p "$HTTP_PORT:8848"
  -p "$GRPC_PORT:9848"
  -p "$RAFT_PORT:9849"
  -e MODE=standalone
  -e NACOS_AUTH_ENABLE="$AUTH_ENABLE"
  -e NACOS_AUTH_TOKEN="$AUTH_TOKEN"
  -e NACOS_AUTH_IDENTITY_KEY="$AUTH_IDENTITY_KEY"
  -e NACOS_AUTH_IDENTITY_VALUE="$AUTH_IDENTITY_VALUE"
  -v "$DATA_VOLUME:/home/nacos/data"
  -v "$LOGS_VOLUME:/home/nacos/logs"
)

if docker ps --format '{{.Names}}' | grep -Fxq "$CONTAINER_NAME"; then
  echo "Nacos container '$CONTAINER_NAME' is already running."
  exit 0
fi

if docker ps -a --format '{{.Names}}' | grep -Fxq "$CONTAINER_NAME"; then
  echo "Starting existing Nacos container '$CONTAINER_NAME'..."
  docker start "$CONTAINER_NAME"
else
  echo "Creating Nacos container '$CONTAINER_NAME' from image '$IMAGE'..."
  docker "${docker_args[@]}" "$IMAGE"
fi

echo
echo "Nacos is starting. Check logs with:"
echo "  docker logs -f $CONTAINER_NAME"
echo
echo "Open console:"
echo "  http://localhost:$HTTP_PORT/nacos"
echo "  http://localhost:$CONSOLE_PORT"
