#!/usr/bin/env python3
"""Real local Python/PHP backends against the Go agent browser CLI.

No internet access is required or used. All test requests target 127.0.0.1.
Does not execute state-changing actions. Results are recorded in JSON.
"""
from __future__ import annotations

import argparse
import http.server
import json
import os
import pathlib
import shutil
import socket
import subprocess
import sys
import threading
import time
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


class PythonBackend(http.server.BaseHTTPRequestHandler):
    hits = {"api": 0, "post": 0}

    def log_message(self, *_args):
        pass

    def do_GET(self):
        from urllib.parse import urlparse, parse_qs
        url = urlparse(self.path)
        headers = {"Content-Type": "text/html; charset=utf-8"}
        if url.path == "/":
            headers["Set-Cookie"] = "session=python-session; Path=/; HttpOnly"
            body = '''<!doctype html><html><head><title>Python real</title>
            <link rel="alternate" type="application/json" title="Python JSON feed" href="/feed.json">
            </head><body><h1>Python real</h1>
            <button id="products" onclick="fetch('/api/products')">Ver produtos Python</button>
            <form action="/search" method="GET"><input name="q" value="antes"><button>Pesquisar Python</button></form>
            <section hx-get="/fragment" aria-label="Fragmento Python">Fragmento</section>
            <form action="/danger" method="POST"><input name="delete" value="1"><button>Apagar</button></form>
            <a href="http://localhost:12345/private">Fora da allowlist</a>
            </body></html>'''
        elif url.path == "/api/products":
            self.hits["api"] += 1
            headers["Content-Type"] = "application/json"
            if "session=python-session" not in self.headers.get("Cookie", ""):
                self.send_response(401)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b'{"error":"missing cookie"}')
                return
            body = json.dumps({"backend": "python", "products": ["Livro", "Caneta"], "api_token": "secret-value"})
        elif url.path == "/feed.json":
            headers["Content-Type"] = "application/json"
            body = json.dumps({"backend": "python", "feed": {"item": 123}})
        elif url.path == "/search":
            from html import escape
            q = parse_qs(url.query).get("q", [""])[0]
            body = "<title>Pesquisa Python</title><h1>Pesquisa Python</h1><p>Consulta: " + escape(q) + "</p>"
        elif url.path == "/fragment":
            body = "<p>Fragmento vindo de Python real</p>"
        else:
            self.send_error(404)
            return
        data = body.encode("utf-8")
        self.send_response(200)
        for k, v in headers.items():
            self.send_header(k, v)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        self.hits["post"] += 1
        self.send_response(403)
        self.end_headers()


class Agent:
    def __init__(self, binary):
        self.p = subprocess.Popen([str(binary), "-allow-hosts=127.0.0.1"],
                                  stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE, text=True, bufsize=1)
        self.max_rss = 0.0

    def command(self, cmd, **kw):
        payload = json.dumps({"cmd": cmd, **kw}, ensure_ascii=False)
        self.p.stdin.write(payload + "\n")
        self.p.stdin.flush()
        line = self.p.stdout.readline()
        if not line:
            raise RuntimeError("GoBrowser terminated: " + self.p.stderr.read()[-1000:])
        result = json.loads(line)
        self.max_rss = max(self.max_rss, result.get("memory", {}).get("rss_mib", 0))
        return result

    def close(self):
        try:
            if self.p.poll() is None:
                self.command("quit")
                self.p.wait(timeout=3)
        except Exception:
            self.p.kill()
        finally:
            if self.p.poll() is None:
                self.p.kill()
            self.p.communicate(timeout=3)


def need(condition, message):
    if not condition:
        raise AssertionError(message)


def action(actions, label, kind=None):
    matches = [a for a in actions if a["label"].casefold() == label.casefold() and (kind is None or a["kind"] == kind)]
    need(len(matches) == 1, f"Expected one {kind or ''} action labeled {label!r}, found {len(matches)}")
    return matches[0]


def run_tests(agent, base, backend, recorder, counters=None):
    def test(label, fn):
        start = time.perf_counter()
        try:
            info = fn()
            recorder.append({"backend": backend, "test": label, "passed": True,
                             "duration_ms": round((time.perf_counter() - start) * 1000, 2),
                             "info": info})
        except Exception as exc:
            recorder.append({"backend": backend, "test": label, "passed": False,
                             "duration_ms": round((time.perf_counter() - start) * 1000, 2),
                             "error": str(exc)})

    def open_root():
        r = agent.command("open", url=base + "/")
        need(r.get("ok"), f"open failed: {r.get('error')}")
        need(backend in r["page"]["title"].lower(), "Unexpected page title")
        return r

    def extract():
        result = agent.command("discover")
        need(result.get("ok"), str(result))
        return result["actions"]

    def discover_only():
        open_root()
        before = dict(counters) if counters else None
        ac = extract()
        # For Python, the action resolver must not fetch the API to discover it.
        if before:
            need(counters["api"] == before["api"] and counters["post"] == before["post"], "discover sent unintended API/POST requests")
        need(len(ac) >= 5, "Too few actions were discovered")
        need(any(a["kind"] == "form" for a in ac), "Missing form action")
        return {"actions": len(ac), "api_hits_before_run": counters["api"] if counters else "not instrumented"}

    def run_json():
        open_root()
        ac = extract()
        a = action(ac, "Ver produtos " + backend.capitalize(), "api")
        need(a["method"] == "GET" and a["executable"], "Expected GET action")
        denied = agent.command("run", ref=a["ref"])
        need(not denied["ok"] and "confirm" in denied.get("error", ""), "inferred action must require confirmation")
        r = agent.command("run", ref=a["ref"], confirm=True)
        need(r.get("ok"), f"API run failed: {r.get('error')}")
        resource = r.get("resource") or {}
        need(resource.get("status") == 200, "API HTTP status is not 200")
        val = resource.get("data") or {}
        need(val.get("backend") == backend, "Bad API data")
        need(val.get("api_token") == "[redacted]", "Sensitive token not masked")
        return {"data": val, "url": resource.get("url")}

    def run_feed():
        open_root()
        ac = extract()
        a = action(ac, backend.capitalize() + " JSON feed", "hypermedia")
        response = agent.command("run", ref=a["ref"], confirm=True)
        need(response.get("ok"), str(response))
        data = response["resource"]["data"]
        need(data.get("feed", {}).get("item") == 123, "Feed did not return expected JSON")
        return {"feed": data}

    def run_search():
        open_root()
        ac = extract()
        a = next((a for a in ac if a["kind"] == "form" and a["method"] == "GET"), None)
        need(a is not None, "Missing GET form")
        # Get the input ref from snapshot, then fill before re-discover/run.
        current = agent.command("snapshot")
        forms = current["page"].get("forms", [])
        f = next((f for f in forms if f["method"] == "GET"), None)
        need(f is not None, "Missing GET form snapshot")
        field = next((v for v in f["fields"] if v.get("name") == "q"), None)
        need(field is not None, "Missing q input")
        updated = agent.command("fill", ref=field["ref"], value="consulta 42")
        need(updated.get("ok"), "Cannot fill form")
        ac = extract()
        a = next(a for a in ac if a["kind"] == "form" and a["method"] == "GET")
        response = agent.command("run", ref=a["ref"])
        need(response.get("ok"), str(response))
        text = (response.get("page") or {}).get("text", "")
        need("consulta 42" in text, f"Form search didn't use input value: {text[:200]}")
        return {"text": text[:120]}

    def run_htmx():
        open_root()
        ac = extract()
        a = next((a for a in ac if a.get("source") == "html:hx-get"), None)
        need(a is not None, "HTMX GET not discovered")
        r = agent.command("run", ref=a["ref"], confirm=True)
        need(r.get("ok"), str(r))
        body = (r.get("resource") or {}).get("body", "")
        need((backend + " real").casefold() in body.casefold(), f"Bad HTMX content: {body!r}")
        return {"preview": body[:160]}

    def refuse_post():
        open_root()
        ac = extract()
        post = next((a for a in ac if a["method"] == "POST"), None)
        need(post is not None, "Missing POST form")
        r = agent.command("run", ref=post["ref"], confirm=True)
        need(not r.get("ok"), "POST must be blocked unless explicitly allowed")
        return {"refused_as_expected": r.get("error")}

    def refuse_host():
        open_root()
        ac = extract()
        link = next((a for a in ac if a["kind"] == "link" and "localhost" in a.get("url", "")), None)
        need(link is not None, "Missing off-allowlist link")
        r = agent.command("run", ref=link["ref"])
        need(not r.get("ok") and "allowlist" in r.get("error", ""), "Off-allowlist URL not rejected")
        return {"refused_as_expected": r.get("error")}

    for name, fn in [
        ("read-only discovery", discover_only),
        ("button -> direct JSON with cookies", run_json),
        ("hypermedia JSON", run_feed),
        ("GET form with filled input", run_search),
        ("HTMX HTTP fragment", run_htmx),
        ("POST denied", refuse_post),
        ("off-allowlist URL denied", refuse_host),
    ]:
        test(name, fn)


def wait_ready(url, p=None):
    for _ in range(80):
        if p and p.poll() is not None:
            raise RuntimeError(f"PHP server exited {p.returncode}: {p.stderr.read()[-200:]}")
        try:
            with urllib.request.urlopen(url, timeout=.2) as response:
                if response.status == 200:
                    return
        except Exception:
            time.sleep(.05)
    raise RuntimeError(f"Server not ready: {url}")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--browser", type=pathlib.Path, default=ROOT / "bin/ai-go-browser-linux-amd64")
    ap.add_argument("--out", type=pathlib.Path, default=ROOT / "integration/results.json")
    ap.add_argument("--repeat", type=int, default=1, help="Repeat each backend suite to catch state bugs")
    ap.add_argument("--no-php", action="store_true", help="Skip PHP backend")
    args = ap.parse_args()
    if args.repeat < 1 or args.repeat > 50:
        ap.error("--repeat must be in [1,50]")
    if not args.browser.is_file():
        ap.error(f"Browser not found: {args.browser}. Compile using go build -o bin/ai-go-browser-linux-amd64 ./cmd/aigobrowser")

    tests = []
    agent = Agent(args.browser)
    php = None
    pyserver = None
    try:
        pyserver = http.server.ThreadingHTTPServer(("127.0.0.1", 0), PythonBackend)
        pyserver.daemon_threads = True
        threading.Thread(target=pyserver.serve_forever, daemon=True).start()
        pybase = f"http://127.0.0.1:{pyserver.server_port}"
        for _ in range(args.repeat):
            run_tests(agent, pybase, "python", tests, PythonBackend.hits)

        if not args.no_php:
            if shutil.which("php") is None:
                print("PHP not installed; skipping actual PHP backend", file=sys.stderr)
            else:
                port = free_port()
                php = subprocess.Popen(["php", "-S", f"127.0.0.1:{port}", "-t", str(ROOT / "integration/php")],
                                       stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
                phpbase = f"http://127.0.0.1:{port}"
                wait_ready(phpbase, php)
                for _ in range(args.repeat):
                    run_tests(agent, phpbase, "php", tests)
    finally:
        agent.close()
        if php is not None:
            php.terminate()
            try:
                php.wait(timeout=3)
            except subprocess.TimeoutExpired:
                php.kill()
                php.wait()
            php.stderr.close()
        if pyserver is not None:
            pyserver.shutdown()
            pyserver.server_close()

    out = {
        "target": "GoBrowser v0.5 action resolver",
        "scope": "actual local servers; no public internet used",
        "python_backend": "Python http.server (actual Python HTTP server)",
        "php_backend": "PHP built-in server (actual PHP runtime)" if shutil.which('php') and not args.no_php else "skipped",
        "passed": sum(t["passed"] for t in tests),
        "failed": sum(not t["passed"] for t in tests),
        "total": len(tests),
        "go_process_peak_sampled_rss_mib": round(agent.max_rss, 2),
        "python_api_hits": PythonBackend.hits["api"],
        "python_post_hits": PythonBackend.hits["post"],
        "tests": tests,
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(out, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    for t in tests:
        print(("PASS" if t["passed"] else "FAIL"), t["backend"], t["test"], f"({t['duration_ms']} ms)", t.get("error", ""))
    print(f"\n{out['passed']}/{out['total']} passed; Go RSS sampled peak {agent.max_rss:.2f} MiB; output: {args.out}")
    return 1 if out["failed"] else 0


if __name__ == "__main__":
    sys.exit(main())
