#!/usr/bin/env bash
set -euo pipefail

HOST="${SRS_WEBRTC_TEST_HOST:-127.0.0.1}"
PORT="${SRS_WEBRTC_TEST_PORT:-18080}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../tools/srs-webrtc-player" && pwd)"

STREAM_URL="${1:-${SRS_WEBRTC_STREAM_URL:-}}"
if [[ -z "${STREAM_URL}" ]]; then
  echo "Usage: scripts/test-srs-webrtc.sh 'webrtc://<srs-host>/live/<stream_name>'" >&2
  echo "Example: scripts/test-srs-webrtc.sh 'webrtc://172.21.103.73/live/live_318034030933057537'" >&2
  exit 1
fi

if [[ -z "${SRS_WEBRTC_API_URL:-}" ]]; then
  SRS_WEBRTC_API_URL="$(
    python3 - "${STREAM_URL}" "${SRS_HTTP_API_PORT:-1985}" <<'PY'
import sys
from urllib.parse import urlparse

stream_url, api_port = sys.argv[1:3]
parsed = urlparse(stream_url)
host = parsed.hostname or "127.0.0.1"
print(f"http://{host}:{api_port}/rtc/v1/play/")
PY
  )"
fi

export SRS_WEBRTC_API_URL

open_url="$(
  python3 - "${HOST}" "${PORT}" "${STREAM_URL}" <<'PY'
import sys
from urllib.parse import quote

host, port, stream = sys.argv[1:4]
print(f"http://{host}:{port}/?stream={quote(stream, safe='')}")
PY
)"

echo "SRS WebRTC test page:"
echo "${open_url}"
echo
echo "Press Ctrl+C to stop."

cd "${ROOT}"
python3 server.py
