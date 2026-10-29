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

        selector = send({"cmd": "query", "selector": "a"})
        assert selector["ok"] and len(selector["matches"]) == 1, selector
        assert selector["matches"][0]["href"].endswith("/next"), selector
        by_attr = send({"cmd": "query", "selector": "input[name=q]"})
        assert by_attr["ok"] and len(by_attr["matches"]) == 1, by_attr
        bad_selector = send({"cmd": "query", "selector": "a:hover"})
        assert not bad_selector["ok"], bad_selector

        linked = send({"cmd": "follow", "ref": "l1"})
        assert linked["ok"] and linked["page"]["title"] == "Cookie kept", linked

        back_step = send({"cmd": "back"})
        assert back_step["ok"] and back_step["page"]["title"] == "Session established", back_step

        clicked = send({"cmd": "click", "ref": "l1"})
        assert clicked["ok"] and clicked["page"]["title"] == "Cookie kept", clicked

        start = send({"cmd": "open", "url": base + "/landing"})
        assert start["ok"] and start["page"]["title"] == "Session established", start

        # Form GET
        form_get = send({"cmd": "submit", "ref": "f1", "fields": ["q=from-agent"]})
        assert form_get["ok"] and form_get["page"]["title"] == "Cookie form works", form_get
        assert form_get["page"]["headings"][0]["text"] == "Query from-agent", form_get

        # Return to landing and submit Form POST
        back_to_landing = send({"cmd": "back"})
        assert back_to_landing["ok"] and back_to_landing["page"]["title"] == "Session established", back_to_landing

        form_post = send({"cmd": "submit", "ref": "f2", "fields": ["user=odin-hero"]})
        assert form_post["ok"] and form_post["page"]["title"] == "POST Login OK", form_post
        assert form_post["page"]["headings"][0]["text"] == "Welcome odin-hero", form_post

        snap = send({"cmd": "snapshot"})
        assert snap["ok"] and snap["page"]["url"].endswith("/login")

        rejected = send({"cmd": "follow", "ref": "l99"})
        assert not rejected["ok"] and rejected["error"] == "link not found", rejected

        bad = send({"cmd": "open", "url": base + "/bad-scheme"})
        assert not bad["ok"] and "redirect rejected" in bad["error"], bad

        still = send({"cmd": "snapshot"})
        assert still["ok"] and still["page"]["title"] == "POST Login OK"

        # Repeated snapshots and navigation must release transient arena data.
        def rss_kib():
            try:
                with open(f"/proc/{p.pid}/status", encoding="utf-8") as fh:
                    for line in fh:
                        if line.startswith("VmRSS:"):
                            return int(line.split()[1])
            except OSError:
                return 0
            return 0

        baseline = rss_kib()
        peak = baseline
        for _ in range(160):
            snap = send({"cmd": "snapshot"})
            assert snap["ok"] and snap["page"]["title"] == "POST Login OK"
            peak = max(peak, rss_kib())
        for _ in range(20):
            page = send({"cmd": "open", "url": base + "/landing"})
            assert page["ok"]
            link = send({"cmd": "follow", "ref": "l1"})
            assert link["ok"] and link["page"]["title"] == "Cookie kept"
            peak = max(peak, rss_kib())
        if baseline:
            delta = peak - baseline
            print(f"Agent RSS baseline={baseline} KiB peak={peak} KiB delta={delta} KiB")
            assert delta < 32768, f"RSS growth above 32 MiB across repeated operations: {delta}"

        goodbye = send({"cmd": "quit"})
        assert goodbye["ok"]
        p.stdin.close()
        assert p.wait(timeout=10) == 0, p.stderr.read()

    print("JSONL agent: GET, POST, cookies, refs, forms, snapshot, back, click PASS")


if __name__ == "__main__":
    main()
