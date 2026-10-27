#!/usr/bin/env python3
"""Exercise the Odin agent as a long-running JSONL process."""
import json
import subprocess
import sys


def main():
    with subprocess.Popen(
        ["./ai-odin-browser", "agent", "--allow-private"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
    ) as p:
        def send(cmd):
            p.stdin.write(json.dumps(cmd) + "\n")
            p.stdin.flush()
            line = p.stdout.readline()
            if not line:
                raise AssertionError("Agent closed unexpectedly: " + p.stderr.read())
            return json.loads(line)

        base = "http://127.0.0.1:18766"

        start = send({"cmd": "open", "url": base + "/start"})
        assert start["ok"] and start["page"]["title"] == "Session established", start

        linked = send({"cmd": "follow", "ref": "l1"})
        assert linked["ok"] and linked["page"]["title"] == "Cookie kept", linked

        start = send({"cmd": "open", "url": base + "/landing"})
        assert start["ok"] and start["page"]["title"] == "Session established", start

        form = send({"cmd": "submit", "ref": "f1", "fields": ["q=from-agent"]})
        assert form["ok"] and form["page"]["title"] == "Cookie form works", form
        assert form["page"]["headings"][0]["text"] == "Query from-agent", form

        snap = send({"cmd": "snapshot"})
        assert snap["ok"] and snap["page"]["url"].endswith("/search?q=from-agent")

        rejected = send({"cmd": "follow", "ref": "l99"})
        assert not rejected["ok"] and rejected["error"] == "link not found", rejected

        bad = send({"cmd": "open", "url": base + "/bad-scheme"})
        assert not bad["ok"] and "redirect rejected" in bad["error"], bad

        still = send({"cmd": "snapshot"})
        assert still["ok"] and still["page"]["title"] == "Cookie form works"

        goodbye = send({"cmd": "quit"})
        assert goodbye["ok"]
        p.stdin.close()
        assert p.wait(timeout=10) == 0, p.stderr.read()

    print("JSONL agent: cookies, refs, forms, snapshot and errors PASS")


if __name__ == "__main__":
    main()
