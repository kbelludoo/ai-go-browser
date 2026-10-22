#!/usr/bin/env python3
"""Measure the real RSS of N concurrent agents on the same page.

    python3 audit/agents.py ./bin/aigobrowser http://127.0.0.1:8098/real.html 20

Prints peak summed RSS, mean RSS per agent and the extrapolation to 80 agents.
Linux only (reads /proc/<pid>/status).
"""
import statistics
import subprocess
import sys
import time


def rss_kib(pid):
    try:
        with open(f"/proc/{pid}/status") as fh:
            for line in fh:
                if line.startswith("VmRSS:"):
                    return int(line.split()[1])
    except FileNotFoundError:
        pass
    return None


def main():
    binary = sys.argv[1] if len(sys.argv) > 1 else "./bin/aigobrowser"
    url = sys.argv[2] if len(sys.argv) > 2 else "http://127.0.0.1:8099/"
    n = int(sys.argv[3]) if len(sys.argv) > 3 else 20

    args = [binary, "-block-private=false", "-engine=http", "-format=json", url]
    procs = [subprocess.Popen(args, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
             for _ in range(n)]

    peak_total, peak_each = 0, []
    started = time.time()
    while any(p.poll() is None for p in procs):
        values = [v for v in (rss_kib(p.pid) for p in procs if p.poll() is None) if v]
        if sum(values) > peak_total:
            peak_total, peak_each = sum(values), [v / 1024 for v in values]
        time.sleep(0.02)

    codes = {p.wait() for p in procs}
    elapsed = (time.time() - started) * 1000
    print(f"agents={n} wall={elapsed:.0f} ms exit_codes={codes}")
    print(f"peak total RSS = {peak_total / 1024:.1f} MiB")
    if peak_each:
        mean = statistics.mean(peak_each)
        print(f"mean per agent = {mean:.1f} MiB (max {max(peak_each):.1f} MiB)")
        print(f"80 agents at that mean = {mean * 80 / 1024:.2f} GiB")


if __name__ == "__main__":
    main()
