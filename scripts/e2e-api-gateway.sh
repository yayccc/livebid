#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:58080}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-10}"
RUN_ID="${RUN_ID:-$(date +%s%N)-$$-${RANDOM:-0}}"

SHOP_USERNAME="${SHOP_USERNAME:-e2e_shop_${RUN_ID}}"
USER_USERNAME="${USER_USERNAME:-e2e_user_${RUN_ID}}"
PASSWORD="${PASSWORD:-123456}"
SHOP_PHONE="${SHOP_PHONE:-}"
USER_PHONE="${USER_PHONE:-}"

TOTAL=0
FAILED=0
SHOP_TOKEN=""
USER_TOKEN=""
SHOP_ID=""
USER_ID=""
GOODS_ID=""
GOODS_SHOP_ID=""
DELETE_GOODS_ID=""
AUCTION_ID=""
CANCEL_AUCTION_ID=""
LIVE_ROOM_ID=""
STREAM_NAME=""
STREAM_CODE=""
ADDRESS_ID=""

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

json_get() {
  local expr="$1"
  python3 -c '
import json
import sys

expr = sys.argv[1]
try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    sys.exit(1)

cur = data
for part in expr.split("."):
    if part == "":
        continue
    if isinstance(cur, list):
        cur = cur[int(part)]
    else:
        cur = cur[part]
if cur is None:
    sys.exit(1)
print(cur)
' "$expr"
}

json_try_get() {
  local expr="$1"
  json_get "$expr" 2>/dev/null || true
}

make_json() {
  python3 - "$@" <<'PY'
import json
import sys

obj = {}
for item in sys.argv[1:]:
    key, value = item.split("=", 1)
    if value.startswith("@int:"):
        obj[key] = int(value[5:])
    elif value.startswith("@raw:"):
        obj[key] = json.loads(value[5:])
    else:
        obj[key] = value
print(json.dumps(obj, ensure_ascii=False))
PY
}

numeric_suffix() {
  printf '%s' "$RUN_ID" | tr -cd '0-9' | tail -c 10
}

e2e_phone() {
  local prefix="$1"
  local suffix
  suffix="$(numeric_suffix)"
  printf '%s%08d' "$prefix" "$((10#$suffix % 100000000))"
}

auth_header() {
  local token="$1"
  if [[ -n "$token" ]]; then
    printf 'Authorization: Bearer %s' "$token"
  else
    printf 'Authorization: Bearer '
  fi
}

request() {
  local name="$1"
  local method="$2"
  local path="$3"
  local expected="${4:-200}"
  local body="${5:-}"
  local token="${6:-}"

  TOTAL=$((TOTAL + 1))
  local response_file
  response_file="$(mktemp)"

  local status
  local -a args=(
    -sS
    --max-time "$TIMEOUT_SECONDS"
    -o "$response_file"
    -w "%{http_code}"
    -X "$method"
    "$BASE_URL$path"
  )
  if [[ -n "$body" ]]; then
    args+=(-H "Content-Type: application/json" -d "$body")
  fi
  if [[ -n "$token" ]]; then
    args+=(-H "$(auth_header "$token")")
  fi

  if ! status="$(curl "${args[@]}")"; then
    echo "FAIL $name curl error" >&2
    cat "$response_file" >&2 || true
    rm -f "$response_file"
    FAILED=$((FAILED + 1))
    return 1
  fi

  local response
  response="$(cat "$response_file")"
  rm -f "$response_file"

  if [[ "$status" != "$expected" ]]; then
    echo "FAIL $name expected=$expected actual=$status" >&2
    echo "$response" >&2
    FAILED=$((FAILED + 1))
    return 1
  fi

  echo "PASS $name [$status]" >&2
  printf '%s' "$response"
}

upload_request() {
  local name="$1"
  local path="$2"
  local expected="${3:-200}"
  local token="${4:-}"

  TOTAL=$((TOTAL + 1))
  local file response_file status
  file="$(mktemp)"
  response_file="$(mktemp)"
  printf 'livebid e2e upload %s\n' "$RUN_ID" > "$file"

  local -a args=(
    -sS
    --max-time "$TIMEOUT_SECONDS"
    -o "$response_file"
    -w "%{http_code}"
    -X POST
    -F "file=@${file};filename=e2e-${RUN_ID}.txt;type=text/plain"
    "$BASE_URL$path"
  )
  if [[ -n "$token" ]]; then
    args+=(-H "$(auth_header "$token")")
  fi

  if ! status="$(curl "${args[@]}")"; then
    echo "FAIL $name curl error" >&2
    cat "$response_file" >&2 || true
    rm -f "$file" "$response_file"
    FAILED=$((FAILED + 1))
    return 1
  fi

  local response
  response="$(cat "$response_file")"
  rm -f "$file" "$response_file"

  if [[ "$status" != "$expected" ]]; then
    echo "FAIL $name expected=$expected actual=$status" >&2
    echo "$response" >&2
    FAILED=$((FAILED + 1))
    return 1
  fi

  echo "PASS $name [$status]" >&2
  printf '%s' "$response"
}

wait_for_gateway() {
  echo "Waiting for api-gateway: $BASE_URL/health"
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$BASE_URL/health" >/dev/null 2>&1; then
      echo "api-gateway is ready"
      return 0
    fi
    sleep 2
  done
  echo "api-gateway is not ready: $BASE_URL/health" >&2
  exit 1
}

main() {
  require_cmd curl
  require_cmd python3

  echo "LiveBid API Gateway E2E"
  echo "BASE_URL=$BASE_URL"
  echo "RUN_ID=$RUN_ID"
  echo

  wait_for_gateway

  if [[ -z "$SHOP_PHONE" ]]; then
    SHOP_PHONE="$(e2e_phone 13)"
  fi
  if [[ -z "$USER_PHONE" ]]; then
    USER_PHONE="$(e2e_phone 15)"
  fi

  request "health" GET "/health" 200 >/dev/null

  local resp body

  body="$(make_json username="$SHOP_USERNAME" password="$PASSWORD" shopName="E2E店铺${RUN_ID}" phone="$SHOP_PHONE" email="${SHOP_USERNAME}@example.com")"
  resp="$(request "shop register" POST "/api/shop/register" 200 "$body")"
  SHOP_ID="$(printf '%s' "$resp" | json_get "data.shopId")"

  body="$(make_json username="$SHOP_USERNAME" password="$PASSWORD")"
  resp="$(request "shop login" POST "/api/shop/login" 200 "$body")"
  SHOP_TOKEN="$(printf '%s' "$resp" | json_get "data.token")"

  resp="$(request "shop me" GET "/api/shop/me" 200 "" "$SHOP_TOKEN")"
  SHOP_ID="$(printf '%s' "$resp" | json_try_get "data.id" || true)"
  if [[ -z "$SHOP_ID" ]]; then
    SHOP_ID="$(printf '%s' "$resp" | json_get "data.shop.id")"
  fi

  body="$(make_json shopName="E2E店铺${RUN_ID}-updated" description="api gateway e2e shop")"
  request "shop update" PUT "/api/shop/${SHOP_ID}" 200 "$body" "$SHOP_TOKEN" >/dev/null

  body="$(make_json username="$USER_USERNAME" password="$PASSWORD" nickname="E2E用户${RUN_ID}" phone="$USER_PHONE" email="${USER_USERNAME}@example.com")"
  resp="$(request "user register" POST "/api/users/register" 200 "$body")"
  USER_ID="$(printf '%s' "$resp" | json_get "data.userId")"

  body="$(make_json username="$USER_USERNAME" password="$PASSWORD")"
  resp="$(request "user login" POST "/api/users/login" 200 "$body")"
  USER_TOKEN="$(printf '%s' "$resp" | json_get "data.token")"

  request "user get public" GET "/api/users/${USER_ID}" 200 >/dev/null

  body="$(make_json nickname="E2E用户${RUN_ID}-updated" gender="@int:1" birthday="1990-01-01")"
  request "user update current" PUT "/api/users/${USER_ID}" 200 "$body" "$USER_TOKEN" >/dev/null

  body="$(make_json receiverName="张三" receiverPhone="13800138000" province="浙江省" city="杭州市" district="西湖区" detailAddress="文三路${RUN_ID}号" postalCode="310000" isDefault="@int:1")"
  resp="$(request "user address create" POST "/api/users/address" 200 "$body" "$USER_TOKEN")"
  ADDRESS_ID="$(printf '%s' "$resp" | json_get "data.addressId")"

  request "user address list" GET "/api/users/address/list?page=1&page_size=10" 200 "" "$USER_TOKEN" >/dev/null
  request "user address get" GET "/api/users/address/${ADDRESS_ID}" 200 "" "$USER_TOKEN" >/dev/null

  body="$(make_json receiverName="李四" receiverPhone="13900139000" province="浙江省" city="杭州市" district="滨江区" detailAddress="江南大道${RUN_ID}号" postalCode="310000" isDefault="@int:0")"
  request "user address update" PUT "/api/users/address/${ADDRESS_ID}" 200 "$body" "$USER_TOKEN" >/dev/null
  request "user address set default" PUT "/api/users/address/${ADDRESS_ID}/default" 200 "" "$USER_TOKEN" >/dev/null

  upload_request "file upload public" "/api/files/upload" 200 >/dev/null
  upload_request "user avatar upload" "/api/users/avatar/upload" 200 "$USER_TOKEN" >/dev/null

  body="$(make_json title="E2E商品${RUN_ID}" cover_url="http://example.com/e2e.png" description="api gateway e2e goods")"
  resp="$(request "goods create" POST "/api/goods" 200 "$body" "$SHOP_TOKEN")"
  GOODS_ID="$(printf '%s' "$resp" | json_get "data.goods.id")"
  GOODS_SHOP_ID="$(printf '%s' "$resp" | json_get "data.goods.shop_id")"
  if [[ "$GOODS_SHOP_ID" != "$SHOP_ID" ]]; then
    echo "FAIL goods create shop mismatch expected=${SHOP_ID} actual=${GOODS_SHOP_ID}" >&2
    exit 1
  fi

  body="$(make_json title="E2E商品${RUN_ID}-updated" cover_url="http://example.com/e2e-updated.png" description="api gateway e2e goods updated")"
  request "goods update" PUT "/api/goods/${GOODS_ID}" 200 "$body" "$SHOP_TOKEN" >/dev/null
  request "goods put on sale" PUT "/api/goods/${GOODS_ID}/on-sale" 200 "" "$SHOP_TOKEN" >/dev/null
  request "goods get public" GET "/api/goods/${GOODS_ID}" 200 >/dev/null
  request "goods list public" GET "/api/goods?page=1&page_size=10&keyword=E2E" 200 >/dev/null

  body="$(python3 - "$GOODS_ID" <<'PY'
import json
import sys
print(json.dumps({"ids": [int(sys.argv[1])]}))
PY
)"
  request "goods batch get" POST "/api/goods/batch" 200 "$body" >/dev/null
  request "goods list shop" GET "/api/goods/shop/list?page=1&page_size=10" 200 "" "$SHOP_TOKEN" >/dev/null

  body="$(make_json title="E2E待删除商品${RUN_ID}" description="api gateway e2e delete goods")"
  resp="$(request "goods create for delete" POST "/api/goods" 200 "$body" "$SHOP_TOKEN")"
  DELETE_GOODS_ID="$(printf '%s' "$resp" | json_get "data.goods.id")"
  request "goods delete" DELETE "/api/goods/${DELETE_GOODS_ID}" 200 "" "$SHOP_TOKEN" >/dev/null

  body="$(make_json title="E2E直播间${RUN_ID}" description="api gateway e2e live room")"
  resp="$(request "live room create" POST "/api/live/rooms" 200 "$body" "$SHOP_TOKEN")"
  LIVE_ROOM_ID="$(printf '%s' "$resp" | json_get "data.live_room.id")"
  STREAM_CODE="$(printf '%s' "$resp" | json_get "data.initial_stream_code")"
  STREAM_NAME="$(printf '%s' "$resp" | json_get "data.rtmp_push_url" | python3 -c 'import sys; print(sys.stdin.read().strip().rstrip("/").split("/")[-1])')"

  request "live room get public" GET "/api/live/rooms/${LIVE_ROOM_ID}" 200 >/dev/null
  request "live room list public" GET "/api/live/rooms?page=1&page_size=10" 200 >/dev/null
  request "live start" POST "/api/live/rooms/${LIVE_ROOM_ID}/start" 200 "" "$SHOP_TOKEN" >/dev/null
  request "live stream info" GET "/api/live/rooms/${LIVE_ROOM_ID}/stream" 200 "" "$SHOP_TOKEN" >/dev/null

  body="$(make_json action="on_publish" client_id="e2e-${RUN_ID}" ip="127.0.0.1" app="live" stream="$STREAM_NAME" param="?stream_code=${STREAM_CODE}")"
  request "srs publish callback" POST "/api/srs/callbacks/publish" 200 "$body" >/dev/null

  body="$(make_json action="on_unpublish" client_id="e2e-${RUN_ID}" ip="127.0.0.1" app="live" stream="$STREAM_NAME" param="")"
  request "srs unpublish callback" POST "/api/srs/callbacks/unpublish" 200 "$body" >/dev/null

  local start_time end_time
  start_time="$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(minutes=1)).isoformat().replace("+00:00", "Z"))
PY
)"
  end_time="$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(hours=1)).isoformat().replace("+00:00", "Z"))
PY
)"

  body="$(make_json goods_id="@int:${GOODS_ID}" shop_id="@int:${SHOP_ID}" room_id="@int:${LIVE_ROOM_ID}" start_price="@int:100" bid_increment="@int:10" seal_price="@int:1000" start_time="$start_time" end_time="$end_time")"
  resp="$(request "auction create" POST "/api/auctions" 200 "$body" "$SHOP_TOKEN")"
  AUCTION_ID="$(printf '%s' "$resp" | json_get "data.auction.id")"

  request "auction get" GET "/api/auctions/${AUCTION_ID}" 200 >/dev/null
  request "auction get by goods" GET "/api/auctions/by-goods/${GOODS_ID}" 200 >/dev/null
  request "auction list shop" GET "/api/auctions?shop_id=${SHOP_ID}&page=1&page_size=10" 200 >/dev/null

  body="$(make_json shop_id="@int:${SHOP_ID}" start_price="@int:120" bid_increment="@int:20" seal_price="@int:1200" start_time="$start_time" end_time="$end_time")"
  request "auction update" PUT "/api/auctions/${AUCTION_ID}" 200 "$body" "$SHOP_TOKEN" >/dev/null

  body="$(make_json shop_id="@int:${SHOP_ID}")"
  request "auction start" POST "/api/auctions/${AUCTION_ID}/start" 200 "$body" "$SHOP_TOKEN" >/dev/null

  body="$(make_json user_id="@int:${USER_ID}" room_id="@int:${LIVE_ROOM_ID}" bid_price="@int:160" request_id="e2e-${RUN_ID}-bid-1")"
  request "auction place bid" POST "/api/auctions/${AUCTION_ID}/bids" 200 "$body" "$USER_TOKEN" >/dev/null
  request "auction list bids" GET "/api/auctions/${AUCTION_ID}/bids?page=1&page_size=10" 200 >/dev/null

  body="$(make_json shop_id="@int:${SHOP_ID}")"
  request "auction finish" POST "/api/auctions/${AUCTION_ID}/finish" 200 "$body" "$SHOP_TOKEN" >/dev/null
  request "live end" POST "/api/live/rooms/${LIVE_ROOM_ID}/end" 200 "" "$SHOP_TOKEN" >/dev/null
  request "goods put off sale" PUT "/api/goods/${GOODS_ID}/off-sale" 200 "" "$SHOP_TOKEN" >/dev/null

  body="$(make_json title="E2E取消拍卖商品${RUN_ID}" description="api gateway e2e cancel auction goods")"
  resp="$(request "goods create for cancel auction" POST "/api/goods" 200 "$body" "$SHOP_TOKEN")"
  local cancel_goods_id
  cancel_goods_id="$(printf '%s' "$resp" | json_get "data.goods.id")"

  body="$(make_json goods_id="@int:${cancel_goods_id}" shop_id="@int:${SHOP_ID}" room_id="@int:${LIVE_ROOM_ID}" start_price="@int:100" bid_increment="@int:10")"
  resp="$(request "auction create for cancel" POST "/api/auctions" 200 "$body" "$SHOP_TOKEN")"
  CANCEL_AUCTION_ID="$(printf '%s' "$resp" | json_get "data.auction.id")"

  body="$(make_json shop_id="@int:${SHOP_ID}")"
  request "auction cancel" POST "/api/auctions/${CANCEL_AUCTION_ID}/cancel" 200 "$body" "$SHOP_TOKEN" >/dev/null
  request "auction delete canceled" DELETE "/api/auctions/${CANCEL_AUCTION_ID}?shop_id=${SHOP_ID}" 200 "" "$SHOP_TOKEN" >/dev/null

  request "user address delete" DELETE "/api/users/address/${ADDRESS_ID}" 200 "" "$USER_TOKEN" >/dev/null

  echo
  echo "E2E summary: total=${TOTAL}, failed=${FAILED}"
  echo "shop_id=${SHOP_ID}, user_id=${USER_ID}, goods_id=${GOODS_ID}, auction_id=${AUCTION_ID}, live_room_id=${LIVE_ROOM_ID}"
  if [[ "$FAILED" -gt 0 ]]; then
    exit 1
  fi
}

main "$@"
