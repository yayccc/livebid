#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="${MYSQL_CONTAINER_NAME:-mysql8}"
IMAGE="${MYSQL_IMAGE:-mysql:8.0}"
ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-123456}"
DATABASE="${MYSQL_DATABASE:-mydb}"
HOST_PORT="${MYSQL_HOST_PORT:-3306}"
VOLUME_NAME="${MYSQL_VOLUME:-mysql_data}"

if docker ps --format '{{.Names}}' | grep -Fxq "$CONTAINER_NAME"; then
  echo "MySQL container '$CONTAINER_NAME' is already running."
  exit 0
fi

if docker ps -a --format '{{.Names}}' | grep -Fxq "$CONTAINER_NAME"; then
  echo "Starting existing MySQL container '$CONTAINER_NAME'..."
  docker start "$CONTAINER_NAME"
else
  echo "Creating MySQL container '$CONTAINER_NAME' from image '$IMAGE'..."
  docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$HOST_PORT:3306" \
    -e MYSQL_ROOT_PASSWORD="$ROOT_PASSWORD" \
    -e MYSQL_DATABASE="$DATABASE" \
    -v "$VOLUME_NAME:/var/lib/mysql" \
    "$IMAGE"
fi

echo
echo "MySQL is starting. Check logs with:"
echo "  docker logs -f $CONTAINER_NAME"
echo
echo "Connect with:"
echo "  docker exec -it $CONTAINER_NAME mysql -uroot -p"
