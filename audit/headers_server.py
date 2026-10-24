#!/usr/bin/env python3
"""Capture the HTTP headers the pure-Go engine actually sends.

    python3 audit/headers_server.py 8099 &
    ./bin/aigobrowser -block-private=false -engine=http -format=json http://127.0.0.1:8099/

Every request is printed as JSON (request line + headers) on stdout.
"""
import http.server
import json
import socketserver
import sys

PAGE = b"""<!doctype html><html><head><title>Header fixture</title></head>
<body><h1>Fixture</h1><p>Hello from the local fixture.</p></body></html>"""


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        print(json.dumps({"requestline": self.requestline,
                          "headers": dict(self.headers.items())}, indent=2), flush=True)
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(PAGE)))
        self.end_headers()
        self.wfile.write(PAGE)

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8099
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("127.0.0.1", port), Handler) as httpd:
        print(f"listening on 127.0.0.1:{port}", flush=True)
        httpd.serve_forever()
