#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-deployments/docker-compose.yml}"
SRS_HOST="${SRS_HOST:-172.21.103.73}"

export SRS_PUBLIC_HOST="${SRS_PUBLIC_HOST:-$SRS_HOST}"
export SRS_RTC_CANDIDATE="${SRS_RTC_CANDIDATE:-$SRS_HOST}"
export SRS_CALLBACK_BASE_URL="${SRS_CALLBACK_BASE_URL:-http://api-gateway:58080}"

cd "$ROOT_DIR"

echo "Starting LiveBid Docker Compose for local debugging..."
echo "Compose file:       $COMPOSE_FILE"
echo "SRS_PUBLIC_HOST:    $SRS_PUBLIC_HOST"
echo "SRS_RTC_CANDIDATE:  $SRS_RTC_CANDIDATE"
echo "SRS_CALLBACK_BASE:  $SRS_CALLBACK_BASE_URL"
echo

docker compose -f "$COMPOSE_FILE" up --build "$@"
