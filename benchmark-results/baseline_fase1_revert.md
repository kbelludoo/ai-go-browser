# Browser task comparison

Generated: 2026-10-11T18:31:11Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 5 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 35/35 | 1931.11 | 0.00 | 313 | 16.54 | 30.78 | 823.59 |
| chromium_cdp | 35/35 | 2216.55 | 534.46 | 244 | 7.60 | 18.67 | 856.76 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.15 | 0.00 | 2.15 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 20.77 | 0.00 | 20.77 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.50 | 0.00 | 0.50 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 0.54 | 0.00 | 0.54 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1075.43 | 0.00 | 1075.43 | 633 | 697 |
| ai_go | 1 | async_dom | warm | true | 878.46 | 0.00 | 878.46 | 326 | 342 |
| chromium_cdp | 1 | static | cold | true | 56.18 | 592.37 | 648.55 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 54.27 | 0.00 | 54.27 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 51.26 | 0.00 | 51.26 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 130.91 | 0.00 | 130.91 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 215.42 | 0.00 | 215.42 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 248.67 | 0.00 | 248.67 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 890.69 | 0.00 | 890.69 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 51.81 | 538.03 | 589.84 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 53.12 | 0.00 | 53.12 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 54.93 | 0.00 | 54.93 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 122.27 | 0.00 | 122.27 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 207.65 | 0.00 | 207.65 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 335.61 | 0.00 | 335.61 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 858.60 | 0.00 | 858.60 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.63 | 0.00 | 1.63 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 20.57 | 0.00 | 20.57 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.57 | 0.00 | 0.57 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.57 | 0.00 | 0.57 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.56 | 0.00 | 0.56 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1014.85 | 0.00 | 1014.85 | 633 | 697 |
| ai_go | 2 | async_dom | warm | true | 872.62 | 0.00 | 872.62 | 326 | 342 |
| ai_go | 3 | static | cold | true | 1.07 | 0.00 | 1.07 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 21.03 | 0.00 | 21.03 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.54 | 0.00 | 0.54 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1039.60 | 0.00 | 1039.60 | 633 | 697 |
| ai_go | 3 | async_dom | warm | true | 872.55 | 0.00 | 872.55 | 326 | 342 |
| chromium_cdp | 3 | static | cold | true | 73.68 | 479.30 | 552.98 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 52.82 | 0.00 | 52.82 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 45.27 | 0.00 | 45.27 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 116.56 | 0.00 | 116.56 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 208.92 | 0.00 | 208.92 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 399.58 | 0.00 | 399.58 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 856.55 | 0.00 | 856.55 | 206 | 327 |
| chromium_cdp | 4 | static | cold | true | 51.05 | 549.04 | 600.09 | 309 | 192 |
| chromium_cdp | 4 | large_html | warm | true | 108.48 | 0.00 | 108.48 | 220 | 2097334 |
| chromium_cdp | 4 | unicode | warm | true | 61.63 | 0.00 | 61.63 | 302 | 125 |
| chromium_cdp | 4 | get_form | warm | true | 222.09 | 0.00 | 222.09 | 244 | 115 |
| chromium_cdp | 4 | link_cookie | warm | true | 261.51 | 0.00 | 261.51 | 218 | 115 |
| chromium_cdp | 4 | spa_click_fill | warm | true | 201.27 | 0.00 | 201.27 | 377 | 493 |
| chromium_cdp | 4 | async_dom | warm | true | 858.82 | 0.00 | 858.82 | 206 | 327 |
| ai_go | 4 | static | cold | true | 1.11 | 0.00 | 1.11 | 389 | 207 |
| ai_go | 4 | large_html | warm | true | 20.95 | 0.00 | 20.95 | 288 | 2097349 |
| ai_go | 4 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 4 | get_form | warm | true | 0.55 | 0.00 | 0.55 | 311 | 130 |
| ai_go | 4 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 4 | spa_click_fill | warm | true | 974.33 | 0.00 | 974.33 | 633 | 697 |
| ai_go | 4 | async_dom | warm | true | 853.10 | 0.00 | 853.10 | 326 | 342 |
| ai_go | 5 | static | cold | true | 1.61 | 0.00 | 1.61 | 389 | 207 |
| ai_go | 5 | large_html | warm | true | 20.46 | 0.00 | 20.46 | 288 | 2097349 |
| ai_go | 5 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 5 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 5 | link_cookie | warm | true | 0.03 | 0.00 | 0.03 | 293 | 130 |
| ai_go | 5 | spa_click_fill | warm | true | 1049.90 | 0.00 | 1049.90 | 633 | 697 |
| ai_go | 5 | async_dom | warm | true | 907.87 | 0.00 | 907.87 | 326 | 342 |
| chromium_cdp | 5 | static | cold | true | 47.78 | 513.58 | 561.36 | 309 | 192 |
| chromium_cdp | 5 | large_html | warm | true | 58.00 | 0.00 | 58.00 | 220 | 2097334 |
| chromium_cdp | 5 | unicode | warm | true | 53.96 | 0.00 | 53.96 | 302 | 125 |
| chromium_cdp | 5 | get_form | warm | true | 117.75 | 0.00 | 117.75 | 244 | 115 |
| chromium_cdp | 5 | link_cookie | warm | true | 212.69 | 0.00 | 212.69 | 218 | 115 |
| chromium_cdp | 5 | spa_click_fill | warm | true | 226.26 | 0.00 | 226.26 | 377 | 493 |
| chromium_cdp | 5 | async_dom | warm | true | 844.34 | 0.00 | 844.34 | 206 | 327 |

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
