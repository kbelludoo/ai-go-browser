# Browser task comparison

Generated: 2026-10-10T08:19:25Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 2090.65 | 0.17 | 362 | 18.25 | 38.55 | 859.55 |
| chromium_cdp | 21/21 | 2290.92 | 599.37 | 244 | 5.25 | 26.82 | 882.18 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.06 | 0.00 | 2.06 | 506 | 207 |
| ai_go | 1 | large_html | warm | true | 20.32 | 0.00 | 20.32 | 338 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 362 | 140 |
| ai_go | 1 | get_form | warm | true | 1.00 | 0.00 | 1.00 | 346 | 130 |
| ai_go | 1 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 326 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1109.58 | 0.00 | 1109.58 | 624 | 821 |
| ai_go | 1 | async_dom | warm | true | 959.11 | 0.00 | 959.11 | 369 | 571 |
| chromium_cdp | 1 | static | cold | true | 49.37 | 650.92 | 700.29 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 52.13 | 0.00 | 52.13 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 49.83 | 0.00 | 49.83 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 120.97 | 0.00 | 120.97 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 161.08 | 0.00 | 161.08 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 238.43 | 0.00 | 238.43 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 859.42 | 0.00 | 859.42 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 55.69 | 581.29 | 636.98 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 56.23 | 0.00 | 56.23 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 55.54 | 0.00 | 55.54 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 163.54 | 0.00 | 163.54 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 204.60 | 0.00 | 204.60 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 442.97 | 0.00 | 442.97 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 861.01 | 0.00 | 861.01 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.56 | 0.00 | 1.56 | 506 | 207 |
| ai_go | 2 | large_html | warm | true | 23.97 | 0.00 | 23.97 | 338 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 362 | 140 |
| ai_go | 2 | get_form | warm | true | 0.98 | 0.00 | 0.98 | 346 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 326 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1308.16 | 0.00 | 1308.16 | 624 | 821 |
| ai_go | 2 | async_dom | warm | true | 883.64 | 0.00 | 883.64 | 369 | 571 |
| ai_go | 3 | static | cold | true | 1.08 | 0.52 | 1.60 | 506 | 207 |
| ai_go | 3 | large_html | warm | true | 19.99 | 0.00 | 19.99 | 338 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.04 | 0.00 | 0.04 | 362 | 140 |
| ai_go | 3 | get_form | warm | true | 0.55 | 0.00 | 0.55 | 346 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.53 | 0.00 | 0.53 | 326 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1040.92 | 0.00 | 1040.92 | 624 | 821 |
| ai_go | 3 | async_dom | warm | true | 897.96 | 0.00 | 897.96 | 369 | 571 |
| chromium_cdp | 3 | static | cold | true | 45.21 | 565.90 | 611.11 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 49.13 | 0.00 | 49.13 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 47.71 | 0.00 | 47.71 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 115.29 | 0.00 | 115.29 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 209.27 | 0.00 | 209.27 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 389.38 | 0.00 | 389.38 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 847.85 | 0.00 | 847.85 | 206 | 327 |

## Interpretation limits

- This corpus cannot prove access to every website or universal superiority. No external AI model is run and token counts are not measured.
- Each engine and repetition runs in a fresh isolated Go worker. The first case includes engine setup; subsequent cases reuse that engine. The first JS case may still include a lazy Chromium launch for AI Go.
- The baseline is real Chromium controlled through the repository's CDP/stealth launcher and NavigateReady. It waits for the navigation's lifecycle load event without a fixed settling delay, then polls task assertions; it is not a comparison against every agent framework.
- Compact snapshot bytes are actual UTF-8 JSON lengths. Both adapters cap visible text at 5000 Unicode code points and links at 200; schemas differ. Document bytes represent fetched/serialized HTML, not model input tokens or total network traffic.
- GoHeapAlloc and GoSys belong to the benchmark worker and include fixtures/adapter allocations. GoSys is runtime-reserved memory, not RSS. Harness RSS and Chromium tree RSS are separate OS samples after each task, not continuously measured peaks.
- Chromium tree RSS sums resident working sets of process descendants. Shared pages can be counted multiple times, processes can exit during sampling, and unavailable readings remain null rather than zero.
- Per-case contexts bound CDP operations and assertion polling; AI Go commands also use the browser's internal request/render timeouts and may exceed the requested case timeout.
- Results depend on hardware, OS, Chromium version, local caches and task order. The command alternates engine order across repetitions; aggregate medians across different task types describe this fixed corpus only.
- Very short local HTTP tasks can round to zero at the host timer resolution; these readings do not mean that the operation has no cost.
