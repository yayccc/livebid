#!/usr/bin/env python3
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib import request
from urllib.error import HTTPError
import json
import os


ROOT = Path(__file__).resolve().parent
SRS_API_URL = os.environ.get("SRS_WEBRTC_API_URL", "http://127.0.0.1:1985/rtc/v1/play/")


def rewrite_srs_request_body(body):
    try:
        payload = json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return body

    if isinstance(payload, dict):
        payload["api"] = SRS_API_URL
        return json.dumps(payload, separators=(",", ":")).encode("utf-8")

    return body


class Handler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(ROOT), **kwargs)

    def do_POST(self):
        if self.path != "/rtc/v1/play/":
            self.send_error(404)
            return

        body = self.rfile.read(int(self.headers.get("content-length", "0")))
        body = rewrite_srs_request_body(body)
        proxied = request.Request(
            SRS_API_URL,
            data=body,
            method="POST",
            headers={"Content-Type": "application/json"},
        )

        try:
            with request.urlopen(proxied, timeout=10) as response:
                payload = response.read()
                self.send_response(response.status)
                self.send_header("Content-Type", response.headers.get("Content-Type", "application/json"))
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)
        except HTTPError as exc:
            payload = exc.read()
            self.send_response(exc.status)
            self.send_header("Content-Type", exc.headers.get("Content-Type", "application/json"))
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
        except Exception as exc:
            payload = ("SRS proxy failed: " + str(exc)).encode("utf-8")
            self.send_response(502)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)


def main():
    host = os.environ.get("SRS_WEBRTC_TEST_HOST", "127.0.0.1")
    port = int(os.environ.get("SRS_WEBRTC_TEST_PORT", "18080"))
    server = ThreadingHTTPServer((host, port), Handler)
    print(f"SRS WebRTC test page: http://{host}:{port}")
    print(f"SRS API proxy: {SRS_API_URL}")
    print("Press Ctrl+C to stop.")
    server.serve_forever()


if __name__ == "__main__":
    main()
