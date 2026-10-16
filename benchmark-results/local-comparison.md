# Browser task comparison

Generated: 2026-10-10T15:30:01Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 1974.84 | 0.01 | 311 | 18.25 | 38.85 | 849.55 |
| chromium_cdp | 21/21 | 2012.95 | 570.04 | 244 | 7.58 | 27.12 | 874.28 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.10 | 0.03 | 2.13 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 22.88 | 0.00 | 22.88 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 1.00 | 0.00 | 1.00 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1108.74 | 0.00 | 1108.74 | 615 | 589 |
| ai_go | 1 | async_dom | warm | true | 887.51 | 0.00 | 887.51 | 308 | 342 |
| chromium_cdp | 1 | static | cold | true | 48.92 | 586.33 | 635.24 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 53.73 | 0.00 | 53.73 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 46.15 | 0.00 | 46.15 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 114.74 | 0.00 | 114.74 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 140.80 | 0.00 | 140.80 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 186.86 | 0.00 | 186.86 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 828.25 | 0.00 | 828.25 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 45.73 | 569.34 | 615.07 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 51.12 | 0.00 | 51.12 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 50.51 | 0.00 | 50.51 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 116.67 | 0.00 | 116.67 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 155.57 | 0.00 | 155.57 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 197.43 | 0.00 | 197.43 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 839.48 | 0.00 | 839.48 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.07 | 0.00 | 1.07 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 21.77 | 0.00 | 21.77 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.53 | 0.00 | 0.53 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1063.70 | 0.00 | 1063.70 | 615 | 589 |
| ai_go | 2 | async_dom | warm | true | 836.82 | 0.00 | 836.82 | 308 | 342 |
| ai_go | 3 | static | cold | true | 1.09 | 0.00 | 1.09 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 21.86 | 0.00 | 21.86 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.54 | 0.00 | 0.54 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1074.07 | 0.00 | 1074.07 | 615 | 589 |
| ai_go | 3 | async_dom | warm | true | 879.75 | 0.00 | 879.75 | 308 | 342 |
| chromium_cdp | 3 | static | cold | true | 43.96 | 554.45 | 598.41 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 53.29 | 0.00 | 53.29 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 55.08 | 0.00 | 55.08 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 113.89 | 0.00 | 113.89 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 147.00 | 0.00 | 147.00 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 196.15 | 0.00 | 196.15 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 843.43 | 0.00 | 843.43 | 206 | 327 |

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
