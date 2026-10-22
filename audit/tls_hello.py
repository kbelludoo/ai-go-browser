#!/usr/bin/env python3
"""Capture the TLS ClientHello of a client and compute its JA3.

    python3 audit/tls_hello.py 8443 &
    ./bin/aigobrowser -block-private=false -engine=http -format=json https://localhost:8443/

Prints the JA3 string, its MD5, cipher list, extension order, ALPN and whether
GREASE values are present. Nothing else is needed: the connection is dropped
right after the ClientHello, so no certificate is required.
"""
import hashlib
import json
import socket
import struct
import sys

GREASE = {0x0A0A, 0x1A1A, 0x2A2A, 0x3A3A, 0x4A4A, 0x5A5A, 0x6A6A, 0x7A7A,
          0x8A8A, 0x9A9A, 0xAAAA, 0xBABA, 0xCACA, 0xDADA, 0xEAEA, 0xFAFA}


def parse(data):
    if data[0] != 22:
        raise ValueError("not a handshake record")
    hs = data[5:5 + struct.unpack("!H", data[3:5])[0]]
    if hs[0] != 1:
        raise ValueError("not a ClientHello")
    p = 4
    version = hs[p:p + 2]
    p += 2 + 32  # version + random
    p += 1 + hs[p]  # session id
    cs_len = int.from_bytes(hs[p:p + 2], "big")
    p += 2
    ciphers = [int.from_bytes(hs[p + i:p + i + 2], "big") for i in range(0, cs_len, 2)]
    p += cs_len
    p += 1 + hs[p]  # compression methods
    exts, curves, point_fmts, alpn, sigalgs = [], [], [], [], []
    if p < len(hs):
        end = p + 2 + int.from_bytes(hs[p:p + 2], "big")
        p += 2
        while p < end:
            et = int.from_bytes(hs[p:p + 2], "big")
            el = int.from_bytes(hs[p + 2:p + 4], "big")
            body = hs[p + 4:p + 4 + el]
            exts.append(et)
            if et == 10:  # supported_groups
                n = int.from_bytes(body[0:2], "big")
                curves = [int.from_bytes(body[2 + i:4 + i], "big") for i in range(0, n, 2)]
            elif et == 11:  # ec_point_formats
                point_fmts = list(body[1:1 + body[0]])
            elif et == 16:  # ALPN
                q = 2
                while q < len(body):
                    ln = body[q]
                    alpn.append(body[q + 1:q + 1 + ln].decode(errors="replace"))
                    q += 1 + ln
            elif et == 13:  # signature_algorithms
                n = int.from_bytes(body[0:2], "big")
                sigalgs = [int.from_bytes(body[2 + i:4 + i], "big") for i in range(0, n, 2)]
            p += 4 + el
    ja3 = "%d,%s,%s,%s,%s" % (
        int.from_bytes(version, "big"),
        "-".join(str(c) for c in ciphers if c not in GREASE),
        "-".join(str(e) for e in exts if e not in GREASE),
        "-".join(str(c) for c in curves if c not in GREASE),
        "-".join(str(f) for f in point_fmts),
    )
    return {
        "record_version": "0x" + version.hex(),
        "ja3": ja3,
        "ja3_md5": hashlib.md5(ja3.encode()).hexdigest(),
        "ciphers": [hex(c) for c in ciphers],
        "extensions": [hex(e) for e in exts],
        "curves": [hex(c) for c in curves],
        "sigalgs": [hex(s) for s in sigalgs],
        "alpn": alpn,
        "has_grease": any(v in GREASE for v in ciphers + exts + curves),
    }


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8443
    srv = socket.socket()
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("127.0.0.1", port))
    srv.listen(5)
    print(f"listening on 127.0.0.1:{port}", flush=True)
    while True:
        conn, _ = srv.accept()
        try:
            conn.settimeout(5)
            d = conn.recv(8192)
            need = 5 + struct.unpack("!H", d[3:5])[0]
            while len(d) < need:
                more = conn.recv(8192)
                if not more:
                    break
                d += more
            print(json.dumps(parse(d), indent=2), flush=True)
        except Exception as exc:  # keep serving
            print("error:", exc, flush=True)
        finally:
            conn.close()
