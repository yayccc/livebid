#!/usr/bin/env bash
set -euo pipefail

# 可通过环境变量覆盖这些参数，便于本地联调时快速调整。
CONTAINER_NAME="${SRS_CONTAINER_NAME:-livebid-srs}"
IMAGE="${SRS_IMAGE:-ossrs/srs:5}"

RTMP_PORT="${SRS_RTMP_PORT:-1935}"
HTTP_API_PORT="${SRS_HTTP_API_PORT:-1985}"
HTTP_SERVER_PORT="${SRS_HTTP_SERVER_PORT:-8088}"
RTC_PORT="${SRS_RTC_PORT:-8000}"
RTC_CANDIDATE="${SRS_RTC_CANDIDATE:-127.0.0.1}"

API_GATEWAY_BASE_URL="${SRS_CALLBACK_BASE_URL:-http://host.docker.internal:58080}"
CONFIG_FILE="${SRS_CONFIG_FILE:-/tmp/livebid-srs.conf}"

cat >"${CONFIG_FILE}" <<EOF
listen              1935;
max_connections     1000;

http_api {
    enabled         on;
    listen          1985;
}

http_server {
    enabled         on;
    listen          8080;
    dir             ./objs/nginx/html;
}

rtc_server {
    enabled         on;
    listen          8000;
    candidate       ${RTC_CANDIDATE};
}

vhost __defaultVhost__ {
    min_latency on;
    tcp_nodelay on;

    publish {
        mr off;
    }

    play {
        gop_cache off;
        queue_length 10;
        mw_latency 100;
    }

    rtc {
        enabled     on;
        rtmp_to_rtc on;
    }

    http_hooks {
        enabled      on;
        on_publish   ${API_GATEWAY_BASE_URL}/api/srs/callbacks/publish;
        on_unpublish ${API_GATEWAY_BASE_URL}/api/srs/callbacks/unpublish;
    }
}
EOF

docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

docker run -d \
  --name "${CONTAINER_NAME}" \
  --add-host=host.docker.internal:host-gateway \
  -p "${RTMP_PORT}:1935" \
  -p "${HTTP_API_PORT}:1985" \
  -p "${HTTP_SERVER_PORT}:8080" \
  -p "${RTC_PORT}:8000/udp" \
  -v "${CONFIG_FILE}:/usr/local/srs/conf/livebid.conf:ro" \
  "${IMAGE}" \
  ./objs/srs -c conf/livebid.conf

echo "SRS started: ${CONTAINER_NAME}"
echo "Config:      ${CONFIG_FILE}"
echo "RTMP:        rtmp://127.0.0.1:${RTMP_PORT}/live/{stream_name}?token={stream_code}"
echo "HTTP API:    http://127.0.0.1:${HTTP_API_PORT}"
echo "HTTP FILE:   http://127.0.0.1:${HTTP_SERVER_PORT}"
echo "RTC UDP:     ${RTC_PORT}"
echo "Callback:    ${API_GATEWAY_BASE_URL}/api/srs/callbacks/{publish,unpublish}"
