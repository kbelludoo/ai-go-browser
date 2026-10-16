# Browser task comparison

Generated: 2026-10-11T16:55:29Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 1962.30 | 0.00 | 311 | 16.13 | 37.79 | 792.36 |
| chromium_cdp | 21/21 | 2065.50 | 558.74 | 244 | 5.23 | 26.39 | 813.75 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.71 | 0.00 | 2.71 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 20.45 | 0.00 | 20.45 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 1.39 | 0.00 | 1.39 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 0.99 | 0.00 | 0.99 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1071.28 | 0.00 | 1071.28 | 615 | 589 |
| ai_go | 1 | async_dom | warm | true | 894.28 | 0.00 | 894.28 | 308 | 342 |
| chromium_cdp | 1 | static | cold | true | 47.38 | 547.14 | 594.52 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 57.14 | 0.00 | 57.14 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 64.10 | 0.00 | 64.10 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 162.73 | 0.00 | 162.73 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 139.47 | 0.00 | 139.47 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 214.34 | 0.00 | 214.34 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 846.51 | 0.00 | 846.51 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 45.82 | 579.33 | 625.15 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 54.57 | 0.00 | 54.57 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 47.33 | 0.00 | 47.33 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 116.07 | 0.00 | 116.07 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 151.43 | 0.00 | 151.43 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 199.83 | 0.00 | 199.83 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 857.11 | 0.00 | 857.11 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.62 | 0.00 | 1.62 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 20.90 | 0.00 | 20.90 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.52 | 0.00 | 0.52 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1022.46 | 0.00 | 1022.46 | 615 | 589 |
| ai_go | 2 | async_dom | warm | true | 891.99 | 0.00 | 891.99 | 308 | 342 |
| ai_go | 3 | static | cold | true | 1.70 | 0.00 | 1.70 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 20.93 | 0.00 | 20.93 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.01 | 0.00 | 0.01 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 0.53 | 0.00 | 0.53 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.57 | 0.00 | 0.57 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1033.38 | 0.00 | 1033.38 | 615 | 589 |
| ai_go | 3 | async_dom | warm | true | 900.10 | 0.00 | 900.10 | 308 | 342 |
| chromium_cdp | 3 | static | cold | true | 49.17 | 549.74 | 598.91 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 52.83 | 0.00 | 52.83 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 54.94 | 0.00 | 54.94 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 115.73 | 0.00 | 115.73 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 169.04 | 0.00 | 169.04 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 193.69 | 0.00 | 193.69 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 881.08 | 0.00 | 881.08 | 206 | 327 |

## Interpretation limits

- This corpus cannot prove access to every website or universal superiority. No external AI model is run and token counts are not measured.
- Each engine and repetition runs in a fresh isolated Go worker. The first case includes engine setup; subsequent cases reuse that engine. The first JS case may still include a lazy Chromium launch for AI Go.
- The baseline is real Chromium controlled through the repository's CDP/stealth launcher and NavigateReady. It waits for the navigation's lifecycle load event without a fixed settling delay, then polls task assertions; it is not a comparison against every agent framework.
- AI Go uses CompactSnapshots (CLI default), omitting automatic Markdown. Compact snapshot bytes are actual UTF-8 JSON lengths. Both adapters cap visible text at 5000 Unicode code points, links at 200 and headings at 100; schemas differ. Document bytes represent fetched/serialized HTML, not model input tokens or total network traffic.
- GoHeapAlloc and GoSys belong to the benchmark worker and include fixtures/adapter allocations. GoSys is runtime-reserved memory, not RSS. Harness RSS and Chromium tree RSS are separate OS samples after each task, not continuously measured peaks.
- Chromium tree RSS sums resident working sets of process descendants. Shared pages can be counted multiple times, processes can exit during sampling, and unavailable readings remain null rather than zero.
- Per-case contexts bound CDP operations and assertion polling; AI Go commands also use the browser's internal request/render timeouts and may exceed the requested case timeout.
- Results depend on hardware, OS, Chromium version, local caches and task order. The command alternates engine order across repetitions; aggregate medians across different task types describe this fixed corpus only.
- Very short local HTTP tasks can round to zero at the host timer resolution; these readings do not mean that the operation has no cost.
