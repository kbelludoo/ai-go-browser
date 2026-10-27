#!/usr/bin/env python3
"""Same-runner, same-localhost-URL end-to-end HTTP+parser session comparison.

Go = original browser.New(...).Execute(open/snapshot); Odin = agent JSONL.
Both use real networking, different transport stacks and parser semantics.
Avoid implying feature parity from speed alone.
"""
import json
import os
import platform
import statistics
import subprocess
import time
from pathlib import Path

URL = "http://127.0.0.1:18767/fixture_generated.html"

def rss(pid):
    try:
        with open(f"/proc/{pid}/status",encoding="utf-8") as f:
            for line in f:
                if line.startswith("VmRSS:"):
                    return int(line.split()[1])
    except OSError:
        pass
    return None

def run_agent(exe, name, n):
    cmd=[exe,"agent"]
    if name=="odin":
        cmd.append("--allow-private")
    p=subprocess.Popen(cmd,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,bufsize=1)
    try:
        samples=[]
        last=None
        start_rss=rss(p.pid)
        rss_peak=start_rss or 0
        for i in range(n+4):
            request={"cmd":"open","url":URL}
            p.stdin.write(json.dumps(request)+"\n")
            p.stdin.flush()
            t0=time.perf_counter_ns()
            line=p.stdout.readline()
            delta=(time.perf_counter_ns()-t0)/1e6
            if not line:
                raise AssertionError(f"{name} closed unexpectedly at request {i}")
            reply=json.loads(line)
            if not reply.get("ok"):
                raise AssertionError(f"{name} HTTP browse failed: {reply}")
            pg=reply["page"]
            if pg.get("title")!="Benchmark Odin Go" or len(pg.get("links",[])) < 300:
                raise AssertionError(f"{name} parsed wrong page title/count: {pg.get('title')!r}, {len(pg.get('links',[]))}")
            if i>=4:
                samples.append(round(delta,3))
            last=pg
            rss_peak=max(rss_peak,rss(p.pid) or 0)
        p.stdin.write('{"cmd":"quit"}\n');p.stdin.flush()
        bye=json.loads(p.stdout.readline())
        assert bye.get("ok")
        p.stdin.close()
        assert p.wait(timeout=15)==0,p.stderr.read()
        return {
            "median_request_ms":round(statistics.median(samples),3),
            "min_request_ms":min(samples),
            "max_request_ms":max(samples),
            "rss_initial_kib":start_rss,
            "rss_peak_kib":rss_peak,
            "last_title":last["title"],
            "last_text_chars":len(last.get("text","")),
            "last_links":len(last.get("links",[])),
            "samples_ms":samples
        }
    finally:
        if p.poll() is None:
            p.kill()
            p.wait(timeout=5)

if __name__=="__main__":
    results={
      "scope":"real localhost HTTP GET + static HTML extraction + JSON serialization (not feature equivalent)",
      "url":URL,
      "requests_per_runtime":50,
      "samples_per_runtime":50,
      "machine":platform.platform(),
      "go":run_agent("./go-browser-reference","go",50),
      "odin":run_agent("./ai-odin-browser","odin",50),
    }
    print(json.dumps(results,ensure_ascii=False,indent=2))
    out=Path("odin/bench/live-results.json")
    out.write_text(json.dumps(results,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
