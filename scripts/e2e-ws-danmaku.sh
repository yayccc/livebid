#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:58080}"
WS_URL="${WS_URL:-ws://127.0.0.1:58081/ws/live}"
WS_HEALTH_URL="${WS_HEALTH_URL:-http://127.0.0.1:58081/health}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-10}"
RUN_ID="${RUN_ID:-$(date +%s%N)-$$-${RANDOM:-0}}"

SHOP_USERNAME="${SHOP_USERNAME:-e2e_dm_shop_${RUN_ID}}"
USER_USERNAME="${USER_USERNAME:-e2e_dm_user_${RUN_ID}}"
PASSWORD="${PASSWORD:-123456}"
SHOP_PHONE="${SHOP_PHONE:-}"
USER_PHONE="${USER_PHONE:-}"
CONTENT="${CONTENT:-弹幕E2E${RUN_ID:0:8}}"

SHOP_TOKEN=""
USER_TOKEN=""
LIVE_ROOM_ID=""

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
data = json.load(sys.stdin)
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

make_json() {
  python3 - "$@" <<'PY'
import json
import sys

obj = {}
for item in sys.argv[1:]:
    key, value = item.split("=", 1)
    if value.startswith("@int:"):
        obj[key] = int(value[5:])
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

request() {
  local name="$1"
  local method="$2"
  local path="$3"
  local expected="${4:-200}"
  local body="${5:-}"
  local token="${6:-}"

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
    args+=(-H "Authorization: Bearer $token")
  fi

  if ! status="$(curl "${args[@]}")"; then
    echo "FAIL $name curl error" >&2
    cat "$response_file" >&2 || true
    rm -f "$response_file"
    exit 1
  fi
  local response
  response="$(cat "$response_file")"
  rm -f "$response_file"
  if [[ "$status" != "$expected" ]]; then
    echo "FAIL $name expected=$expected actual=$status" >&2
    echo "$response" >&2
    exit 1
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

wait_for_ws_gateway() {
  echo "Waiting for ws-gateway: $WS_HEALTH_URL"
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$WS_HEALTH_URL" >/dev/null 2>&1; then
      echo "ws-gateway is ready"
      return 0
    fi
    sleep 2
  done
  echo "ws-gateway is not ready: $WS_HEALTH_URL" >&2
  exit 1
}

cleanup() {
  if [[ -n "$SHOP_TOKEN" && -n "$LIVE_ROOM_ID" ]]; then
    curl -fsS --max-time "$TIMEOUT_SECONDS" -X POST -H "Authorization: Bearer $SHOP_TOKEN" \
      "$BASE_URL/api/live/rooms/${LIVE_ROOM_ID}/end" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

run_ws_check() {
  python3 - "$WS_URL" "$LIVE_ROOM_ID" "$USER_TOKEN" "$CONTENT" <<'PY'
import base64
import hashlib
import json
import os
import socket
import ssl
import struct
import sys
import time
import urllib.parse

ws_url, room_id, token, content = sys.argv[1:5]

class WebSocketClient:
    def __init__(self, url, room_id, token, timeout=10):
        parsed = urllib.parse.urlparse(url)
        query = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
        query.extend([("room_id", room_id), ("token", token)])
        self.path = parsed.path or "/"
        self.path += "?" + urllib.parse.urlencode(query)
        self.host = parsed.hostname or "127.0.0.1"
        self.port = parsed.port or (443 if parsed.scheme == "wss" else 80)
        raw = socket.create_connection((self.host, self.port), timeout=timeout)
        self.sock = ssl.create_default_context().wrap_socket(raw, server_hostname=self.host) if parsed.scheme == "wss" else raw
        self.sock.settimeout(timeout)
        self._handshake()

    def _handshake(self):
        key = base64.b64encode(os.urandom(16)).decode()
        request = (
            f"GET {self.path} HTTP/1.1\r\n"
            f"Host: {self.host}:{self.port}\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Key: {key}\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            "\r\n"
        )
        self.sock.sendall(request.encode("ascii"))
        response = b""
        while b"\r\n\r\n" not in response:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise RuntimeError("websocket handshake closed")
            response += chunk
        if not response.startswith(b"HTTP/1.1 101"):
            raise RuntimeError(response.decode("utf-8", "replace"))
        expected = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest()).decode()
        if expected.lower() not in response.decode("utf-8", "replace").lower():
            raise RuntimeError("websocket accept key mismatch")

    def _read_exact(self, n):
        out = b""
        while len(out) < n:
            chunk = self.sock.recv(n - len(out))
            if not chunk:
                raise RuntimeError("websocket closed")
            out += chunk
        return out

    def recv_json(self, deadline):
        while time.time() < deadline:
            first, second = self._read_exact(2)
            opcode = first & 0x0F
            masked = second & 0x80
            length = second & 0x7F
            if length == 126:
                length = struct.unpack("!H", self._read_exact(2))[0]
            elif length == 127:
                length = struct.unpack("!Q", self._read_exact(8))[0]
            mask = self._read_exact(4) if masked else b""
            payload = self._read_exact(length)
            if masked:
                payload = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
            if opcode == 8:
                raise RuntimeError("websocket close frame received")
            if opcode == 9:
                self._send_frame(payload, 10)
                continue
            if opcode == 1:
                return json.loads(payload.decode("utf-8"))
        raise TimeoutError("timed out waiting for websocket message")

    def _send_frame(self, payload, opcode):
        if isinstance(payload, str):
            payload = payload.encode("utf-8")
        mask = os.urandom(4)
        header = bytearray([0x80 | opcode])
        length = len(payload)
        if length < 126:
            header.append(0x80 | length)
        elif length < 65536:
            header.append(0x80 | 126)
            header.extend(struct.pack("!H", length))
        else:
            header.append(0x80 | 127)
            header.extend(struct.pack("!Q", length))
        masked = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
        self.sock.sendall(bytes(header) + mask + masked)

    def send_json(self, value):
        self._send_frame(json.dumps(value, ensure_ascii=False), 1)

    def close(self):
        try:
            self._send_frame(b"", 8)
        finally:
            self.sock.close()

def wait_for(client, predicate, label):
    deadline = time.time() + 10
    while time.time() < deadline:
        msg = client.recv_json(deadline)
        if predicate(msg):
            return msg
    raise TimeoutError(label)

observer = WebSocketClient(ws_url, room_id, token)
observer_connect = wait_for(observer, lambda m: m.get("type") == "response" and m.get("request_type") == "connect" and m.get("code") == 0, "observer connect response")
if "recent_danmaku" not in observer_connect.get("data", {}):
    raise RuntimeError(f"observer connect response missing recent_danmaku: {observer_connect}")

sender = WebSocketClient(ws_url, room_id, token)
connect = wait_for(sender, lambda m: m.get("type") == "response" and m.get("request_type") == "connect" and m.get("code") == 0, "sender connect response")
if "recent_danmaku" not in connect.get("data", {}):
    raise RuntimeError(f"connect response missing recent_danmaku: {connect}")

request_id = "req_dm_" + str(int(time.time() * 1000))
sender.send_json({
    "type": "send_danmaku",
    "request_id": request_id,
    "timestamp": int(time.time() * 1000),
    "data": {"content": content},
})
seen_response = False
seen_sender_broadcast = False
seen_observer_broadcast = False
deadline = time.time() + 10
while time.time() < deadline and not (seen_response and seen_sender_broadcast):
    msg = sender.recv_json(deadline)
    if msg.get("type") == "response" and msg.get("request_id") == request_id:
        if msg.get("code") != 0 or not msg.get("data", {}).get("message_id"):
            raise RuntimeError(f"unexpected send_danmaku response: {msg}")
        seen_response = True
    if msg.get("type") == "danmaku_created" and msg.get("data", {}).get("content") == content:
        seen_sender_broadcast = True
deadline = time.time() + 10
while time.time() < deadline and not seen_observer_broadcast:
    msg = observer.recv_json(deadline)
    if msg.get("type") == "danmaku_created" and msg.get("data", {}).get("content") == content:
        seen_observer_broadcast = True
sender.close()
observer.close()
if not seen_response or not seen_sender_broadcast or not seen_observer_broadcast:
    raise RuntimeError(f"missing response={seen_response} sender_broadcast={seen_sender_broadcast} observer_broadcast={seen_observer_broadcast}")

client = WebSocketClient(ws_url, room_id, token)
connect = wait_for(client, lambda m: m.get("type") == "response" and m.get("request_type") == "connect" and m.get("code") == 0, "reconnect response")
recent = connect.get("data", {}).get("recent_danmaku", [])
client.close()
if not any(item.get("content") == content for item in recent):
    raise RuntimeError(f"recent_danmaku does not include sent content: {recent}")
print("PASS websocket danmaku send, room broadcast, recent")
PY
}

main() {
  require_cmd curl
  require_cmd python3
  wait_for_gateway
  wait_for_ws_gateway

  if [[ -z "$SHOP_PHONE" ]]; then
    SHOP_PHONE="$(e2e_phone 13)"
  fi
  if [[ -z "$USER_PHONE" ]]; then
    USER_PHONE="$(e2e_phone 15)"
  fi

  local body resp
  body="$(make_json username="$SHOP_USERNAME" password="$PASSWORD" shopName="弹幕E2E店铺${RUN_ID}" phone="$SHOP_PHONE" email="${SHOP_USERNAME}@example.com")"
  request "shop register" POST "/api/shop/register" 200 "$body" >/dev/null

  body="$(make_json username="$SHOP_USERNAME" password="$PASSWORD")"
  resp="$(request "shop login" POST "/api/shop/login" 200 "$body")"
  SHOP_TOKEN="$(printf '%s' "$resp" | json_get "data.token")"

  body="$(make_json username="$USER_USERNAME" password="$PASSWORD" nickname="弹幕用户${RUN_ID}" phone="$USER_PHONE" email="${USER_USERNAME}@example.com")"
  request "user register" POST "/api/users/register" 200 "$body" >/dev/null

  body="$(make_json username="$USER_USERNAME" password="$PASSWORD")"
  resp="$(request "user login" POST "/api/users/login" 200 "$body")"
  USER_TOKEN="$(printf '%s' "$resp" | json_get "data.token")"

  body="$(make_json title="弹幕E2E直播间${RUN_ID}" description="ws danmaku e2e")"
  resp="$(request "live room create" POST "/api/live/rooms" 200 "$body" "$SHOP_TOKEN")"
  LIVE_ROOM_ID="$(printf '%s' "$resp" | json_get "data.live_room.id")"

  request "live start" POST "/api/live/rooms/${LIVE_ROOM_ID}/start" 200 "" "$SHOP_TOKEN" >/dev/null

  echo "Testing websocket danmaku: WS_URL=$WS_URL room_id=$LIVE_ROOM_ID"
  run_ws_check
}

main "$@"
