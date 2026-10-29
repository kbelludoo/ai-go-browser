#!/usr/bin/env python3
"""Deterministic local HTTP fixtures: cookies, explicit redirects, redirect cap.

Restricted to 127.0.0.1 for integration tests; not a public service.
"""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def do_POST(self):
        path = urlsplit(self.path).path
        cookie_ok = "session=odin-ok" in self.headers.get("Cookie", "")
        if not cookie_ok:
            self.send_response(403)
            self.end_headers()
            self.wfile.write(b"cookie missing")
            return
        if path == "/login":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length).decode("utf-8")
            data = parse_qs(body)
            user = data.get("user", [""])[0]
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(
                f"<html><head><title>POST Login OK</title></head><body><h1>Welcome {user}</h1></body></html>".encode()
            )
            return
        self.send_response(404)
        self.end_headers()

    def do_GET(self):
        path = urlsplit(self.path).path
        cookie_ok = "session=odin-ok" in self.headers.get("Cookie", "")
        if path == "/start":
            self.send_response(302)
            self.send_header("Set-Cookie", "session=odin-ok; Path=/; HttpOnly; SameSite=Lax")
            self.send_header("Location", "/landing")
            self.end_headers()
            return
        if path == "/loop":
            self.send_response(302)
            self.send_header("Location", "/loop")
            self.end_headers()
            return
        if path == "/bad-scheme":
            self.send_response(302)
            self.send_header("Location", "file:///etc/passwd")
            self.end_headers()
            return
        if path == "/bad-target":
            self.send_response(302)
            self.send_header("Location", "http://user:password@127.0.0.1:18766/landing")
            self.end_headers()
            return
        if not cookie_ok:
            self.send_response(403)
            self.end_headers()
            self.wfile.write(b"cookie missing")
            return
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        if path == "/landing":
            self.wfile.write(
                b"<html><head><title>Session established</title></head>"
                b"<body><h1>Authenticated session</h1>"
                b"<a href='/next'>Protected article</a>"
                b"<form method='GET' action='/search'><input name='q'>"
                b"<button>Submit</button></form>"
                b"<form method='POST' action='/login'><input name='user' value='odin-agent'>"
                b"<button>Login</button></form></body></html>"
            )
        elif path == "/next":
            self.wfile.write(
                b"<html><title>Cookie kept</title><h1>Protected article works</h1></html>"
            )
        elif path == "/search":
            value = parse_qs(urlsplit(self.path).query).get("q", [""])[0]
            self.wfile.write(
                f"<html><title>Cookie form works</title><h1>Query {value}</h1></html>".encode()
            )
        else:
            self.wfile.write(b"<html><title>404</title></html>")


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 18766), Handler).serve_forever()
