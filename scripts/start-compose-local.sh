#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-deployments/docker-compose.yml}"
COMPOSE_OVERRIDE_FILE="${COMPOSE_OVERRIDE_FILE:-}"
PUBLIC_HOST="${PUBLIC_HOST:-127.0.0.1}"
SRS_HOST="${SRS_HOST:-$PUBLIC_HOST}"

export SRS_PUBLIC_HOST="${SRS_PUBLIC_HOST:-$SRS_HOST}"
export SRS_RTC_CANDIDATE="${SRS_RTC_CANDIDATE:-$SRS_HOST}"
export SRS_CALLBACK_BASE_URL="${SRS_CALLBACK_BASE_URL:-http://api-gateway:58080}"
export API_GATEWAY_STORAGE_PUBLIC_BASE_URL="${API_GATEWAY_STORAGE_PUBLIC_BASE_URL:-http://$PUBLIC_HOST:9000/livebid}"

cd "$ROOT_DIR"

compose_args=(-f "$COMPOSE_FILE")
if [[ -n "$COMPOSE_OVERRIDE_FILE" ]]; then
  compose_args+=(-f "$COMPOSE_OVERRIDE_FILE")
fi

echo "Starting LiveBid Docker Compose for local debugging..."
echo "Compose file:       $COMPOSE_FILE"
echo "Compose override:   ${COMPOSE_OVERRIDE_FILE:-<none>}"
echo "PUBLIC_HOST:        $PUBLIC_HOST"
echo "SRS_PUBLIC_HOST:    $SRS_PUBLIC_HOST"
echo "SRS_RTC_CANDIDATE:  $SRS_RTC_CANDIDATE"
echo "SRS_CALLBACK_BASE:  $SRS_CALLBACK_BASE_URL"
echo "Storage public URL: $API_GATEWAY_STORAGE_PUBLIC_BASE_URL"
echo "Skip build:         ${LIVEBID_SKIP_BUILD:-0}"
echo

if [[ "${LIVEBID_SKIP_BUILD:-0}" == "1" ]]; then
  docker compose "${compose_args[@]}" up --no-build "$@"
else
  docker compose "${compose_args[@]}" up --build "$@"
fi
