# Browser task comparison

Generated: 2026-10-11T16:56:18Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 2012.17 | 0.00 | 311 | 16.20 | 30.66 | 797.91 |
| chromium_cdp | 21/21 | 2040.11 | 559.28 | 244 | 5.24 | 18.68 | 811.09 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 1.60 | 0.00 | 1.60 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 20.57 | 0.00 | 20.57 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.99 | 0.00 | 0.99 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 1.00 | 0.00 | 1.00 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1100.12 | 0.00 | 1100.12 | 615 | 589 |
| ai_go | 1 | async_dom | warm | true | 887.54 | 0.00 | 887.54 | 308 | 342 |
| chromium_cdp | 1 | static | cold | true | 46.52 | 553.97 | 600.49 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 51.14 | 0.00 | 51.14 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 51.12 | 0.00 | 51.12 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 116.45 | 0.00 | 116.45 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 155.01 | 0.00 | 155.01 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 202.60 | 0.00 | 202.60 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 835.06 | 0.00 | 835.06 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 52.48 | 547.00 | 599.48 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 55.01 | 0.00 | 55.01 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 51.49 | 0.00 | 51.49 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 113.92 | 0.00 | 113.92 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 191.50 | 0.00 | 191.50 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 168.40 | 0.00 | 168.40 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 841.53 | 0.00 | 841.53 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.64 | 0.00 | 1.64 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 21.02 | 0.00 | 21.02 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 1.01 | 0.00 | 1.01 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1107.35 | 0.00 | 1107.35 | 615 | 589 |
| ai_go | 2 | async_dom | warm | true | 875.40 | 0.00 | 875.40 | 308 | 342 |
| ai_go | 3 | static | cold | true | 1.62 | 0.00 | 1.62 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 20.59 | 0.00 | 20.59 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.03 | 0.00 | 0.03 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 0.52 | 0.00 | 0.52 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1070.30 | 0.00 | 1070.30 | 615 | 589 |
| ai_go | 3 | async_dom | warm | true | 924.68 | 0.00 | 924.68 | 308 | 342 |
| chromium_cdp | 3 | static | cold | true | 47.14 | 576.86 | 624.01 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 57.02 | 0.00 | 57.02 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 54.62 | 0.00 | 54.62 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 124.41 | 0.00 | 124.41 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 161.03 | 0.00 | 161.03 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 198.24 | 0.00 | 198.24 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 867.78 | 0.00 | 867.78 | 206 | 327 |

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
