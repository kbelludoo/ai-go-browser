#!/usr/bin/env python3
"""Fair-ish local static HTML parser+CSO comparison of actual Go and Odin binaries.

Same input bytes, same CI host, same iterations, warm ups. Measures whole
process including language runtime startup and allocation. Does not pretend
these parsers produce identical output: report parity differences separately.
"""
import argparse
import json
import os
import platform
import statistics
import subprocess
import time
from pathlib import Path

def run(cmd):
    start=time.perf_counter_ns()
    result=subprocess.run(cmd,check=True,capture_output=True,text=True)
    return (time.perf_counter_ns()-start)/1e6,result.stdout.strip()

def normalize(project):
    def clean_list(objs,fields):
        return [{k:item.get(k) for k in fields} for item in objs]
    return {
      "title":project.get("title",""),
      "description":project.get("description",""),
      "headings":clean_list(project.get("headings",[]),("level","text")),
      "links":clean_list(project.get("links",[]),("label","url")),
      "text":project.get("text",""),
      "cso":project.get("cso",""),
    }

def mem_run(cmd):
    # /usr/bin/time -f reads maximum resident set size (kilobytes) on Linux.
    r=subprocess.run(["/usr/bin/time","-f","%M"]+cmd,check=True,capture_output=True,text=True)
    return int(r.stderr.strip().splitlines()[-1]),r.stdout.strip()

def compare(binary_go,binary_odin,fixture,url,iterations,reps):
    cmds={
        "go":[binary_go,"parse-file",fixture,url],
        "odin":[binary_odin,"parse-file",fixture,url],
    }
    proj={}
    for name,cmd in cmds.items():
        _ms,data=run(cmd)
        proj[name]=normalize(json.loads(data))
    differences={}
    for field in proj["go"]:
        if proj["go"][field]!=proj["odin"][field]:
            if isinstance(proj["go"][field],str):
                differences[field]={"go_len":len(proj["go"][field]),"odin_len":len(proj["odin"][field]),
                  "go_sample":proj["go"][field][:130],"odin_sample":proj["odin"][field][:130]}
            else:
                differences[field]={"go":proj["go"][field][:10],"odin":proj["odin"][field][:10]}
    samples={}
    rss={}
    checksums={}
    for name,exe in [("go",binary_go),("odin",binary_odin)]:
        cmd=[exe,"bench-file",fixture,url,str(iterations)]
        for _ in range(2):
            run(cmd)
        samples[name]=[]
        for _ in range(reps):
            ms,out=run(cmd)
            samples[name].append(round(ms,2))
            checksums[name]=out
        rss[name],_=mem_run(cmd)
    go=statistics.median(samples["go"])
    odin=statistics.median(samples["odin"])
    result={
      "scope":"static HTML parsing and CSO only, no JS/CDP/network",
      "fixture":fixture,
      "source_bytes":os.stat(fixture).st_size,
      "url":url,
      "iterations_per_run":iterations,
      "repetitions":reps,
      "python":platform.python_version(),
      "system":platform.platform(),
      "arch":platform.machine(),
      "timing":"wall-clock per full benchmark process (includes startup)",
      "median_ms":{"go":round(go,2),"odin":round(odin,2)},
      "samples_ms":samples,
      "peak_rss_kib":rss,
      "odin_to_go_time_ratio":round(odin/go,3) if go else None,
      "checksums":checksums,
      "field_parity":{f: f not in differences for f in proj["go"]},
      "differences":differences,
    }
    return result

if __name__=="__main__":
    p=argparse.ArgumentParser()
    p.add_argument("--go",default="./go-browser-reference")
    p.add_argument("--odin",default="./ai-odin-browser")
    p.add_argument("--fixture",default="odin/fixtures/index.html")
    p.add_argument("--url",default="https://example.com/index.html")
    p.add_argument("--iterations",type=int,default=1200)
    p.add_argument("--reps",type=int,default=5)
    p.add_argument("--out",default="odin/bench/results.json")
    args=p.parse_args()
    res=compare(args.go,args.odin,args.fixture,args.url,args.iterations,args.reps)
    print(json.dumps(res,ensure_ascii=False,indent=2))
    if args.out:
        Path(args.out).parent.mkdir(parents=True,exist_ok=True)
        Path(args.out).write_text(json.dumps(res,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
