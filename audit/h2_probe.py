#!/usr/bin/env python3
"""Capture the HTTP/2 preface, SETTINGS and HEADERS frame a client sends.

Needs a certificate the client trusts. Generate a self-signed one and point the
client at it through SSL_CERT_FILE (Go honours that variable):

    openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 2 -nodes \
        -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
    python3 audit/h2_probe.py 8444 cert.pem key.pem &
    SSL_CERT_FILE=$PWD/cert.pem ./bin/aigobrowser -block-private=false -engine=http \
        -format=json https://localhost:8444/

The capture is written to /tmp/lab/h2.json and can be decoded with
`go run ./audit/h2decode /tmp/lab/h2.json`.
"""
import json
import os
import socket
import ssl
import sys

SETTING_NAMES = {1: "HEADER_TABLE_SIZE", 2: "ENABLE_PUSH", 3: "MAX_CONCURRENT_STREAMS",
                 4: "INITIAL_WINDOW_SIZE", 5: "MAX_FRAME_SIZE", 6: "MAX_HEADER_LIST_SIZE",
                 8: "ENABLE_CONNECT_PROTOCOL", 9: "NO_RFC7540_PRIORITIES"}
PREFACE = b"PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
OUT = "/tmp/lab/h2.json"


def capture(data):
    has_preface = data.startswith(PREFACE)
    body = data[len(PREFACE):] if has_preface else data
    settings, frames, p = {}, [], 0
    while p + 9 <= len(body):
        length = int.from_bytes(body[p:p + 3], "big")
        ftype, flags = body[p + 3], body[p + 4]
        payload = body[p + 9:p + 9 + length]
        frames.append({"type": ftype, "flags": flags, "len": length, "hex": payload.hex()})
        if ftype == 4:  # SETTINGS
            q = 0
            while q + 6 <= len(payload):
                sid = int.from_bytes(payload[q:q + 2], "big")
                settings[SETTING_NAMES.get(sid, hex(sid))] = int.from_bytes(payload[q + 2:q + 6], "big")
                q += 6
        p += 9 + length
    return has_preface, settings, frames


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8444
    cert = sys.argv[2] if len(sys.argv) > 2 else "cert.pem"
    key = sys.argv[3] if len(sys.argv) > 3 else "key.pem"
    os.makedirs(os.path.dirname(OUT), exist_ok=True)

    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(cert, key)
    ctx.set_alpn_protocols(["h2", "http/1.1"])

    srv = socket.socket()
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("127.0.0.1", port))
    srv.listen(5)
    print(f"listening on 127.0.0.1:{port}", flush=True)
    while True:
        raw, _ = srv.accept()
        try:
            conn = ctx.wrap_socket(raw, server_side=True)
            conn.settimeout(4)
            data = b""
            try:
                while len(data) < 200:
                    chunk = conn.recv(4096)
                    if not chunk:
                        break
                    data += chunk
            except socket.timeout:
                pass
            preface, settings, frames = capture(data)
            out = {"alpn": conn.selected_alpn_protocol(), "h2_preface": preface,
                   "settings": settings, "frames": frames}
            with open(OUT, "w") as fh:
                json.dump(out, fh, indent=2)
            print(json.dumps({"alpn": out["alpn"], "preface": preface,
                              "settings": settings,
                              "frames": [(f["type"], f["flags"], f["len"]) for f in frames]},
                             indent=2), flush=True)
            conn.close()
        except Exception as exc:
            print("error:", exc, flush=True)
