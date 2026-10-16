# Browser task comparison

Generated: 2026-10-11T17:21:19Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 5 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 35/35 | 1575.59 | 0.00 | 313 | 16.55 | 31.20 | 370.74 |
| chromium_cdp | 35/35 | 1810.05 | 372.67 | 244 | 5.49 | 18.70 | 404.69 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.16 | 0.00 | 2.16 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 22.16 | 0.00 | 22.16 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 0.57 | 0.00 | 0.57 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 661.39 | 0.00 | 661.39 | 633 | 697 |
| ai_go | 1 | async_dom | warm | true | 906.51 | 0.00 | 906.51 | 326 | 342 |
| chromium_cdp | 1 | static | cold | true | 52.97 | 383.78 | 436.75 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 63.25 | 0.00 | 63.25 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 51.10 | 0.00 | 51.10 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 123.84 | 0.00 | 123.84 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 117.42 | 0.00 | 117.42 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 191.36 | 0.00 | 191.36 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 854.45 | 0.00 | 854.45 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 49.86 | 360.86 | 410.72 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 67.14 | 0.00 | 67.14 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 51.68 | 0.00 | 51.68 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 117.43 | 0.00 | 117.43 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 109.75 | 0.00 | 109.75 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 191.17 | 0.00 | 191.17 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 850.01 | 0.00 | 850.01 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.67 | 0.00 | 1.67 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 21.40 | 0.00 | 21.40 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.56 | 0.00 | 0.56 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.97 | 0.00 | 0.97 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 626.55 | 0.00 | 626.55 | 633 | 697 |
| ai_go | 2 | async_dom | warm | true | 907.93 | 0.00 | 907.93 | 326 | 342 |
| ai_go | 3 | static | cold | true | 2.01 | 0.00 | 2.01 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 21.98 | 0.00 | 21.98 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 1.00 | 0.00 | 1.00 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 625.02 | 0.00 | 625.02 | 633 | 697 |
| ai_go | 3 | async_dom | warm | true | 907.25 | 0.00 | 907.25 | 326 | 342 |
| chromium_cdp | 3 | static | cold | true | 62.18 | 404.12 | 466.29 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 59.87 | 0.00 | 59.87 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 46.01 | 0.00 | 46.01 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 94.40 | 0.00 | 94.40 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 117.29 | 0.00 | 117.29 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 191.18 | 0.00 | 191.18 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 848.75 | 0.00 | 848.75 | 206 | 327 |
| chromium_cdp | 4 | static | cold | true | 59.39 | 363.62 | 423.00 | 309 | 192 |
| chromium_cdp | 4 | large_html | warm | true | 59.80 | 0.00 | 59.80 | 220 | 2097334 |
| chromium_cdp | 4 | unicode | warm | true | 47.37 | 0.00 | 47.37 | 302 | 125 |
| chromium_cdp | 4 | get_form | warm | true | 115.09 | 0.00 | 115.09 | 244 | 115 |
| chromium_cdp | 4 | link_cookie | warm | true | 112.96 | 0.00 | 112.96 | 218 | 115 |
| chromium_cdp | 4 | spa_click_fill | warm | true | 188.67 | 0.00 | 188.67 | 377 | 493 |
| chromium_cdp | 4 | async_dom | warm | true | 849.68 | 0.00 | 849.68 | 206 | 327 |
| ai_go | 4 | static | cold | true | 1.66 | 0.00 | 1.66 | 389 | 207 |
| ai_go | 4 | large_html | warm | true | 20.68 | 0.00 | 20.68 | 288 | 2097349 |
| ai_go | 4 | unicode | warm | true | 0.57 | 0.00 | 0.57 | 313 | 140 |
| ai_go | 4 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 4 | link_cookie | warm | true | 0.56 | 0.00 | 0.56 | 293 | 130 |
| ai_go | 4 | spa_click_fill | warm | true | 675.80 | 0.00 | 675.80 | 633 | 697 |
| ai_go | 4 | async_dom | warm | true | 907.63 | 0.00 | 907.63 | 326 | 342 |
| ai_go | 5 | static | cold | true | 1.66 | 0.00 | 1.66 | 389 | 207 |
| ai_go | 5 | large_html | warm | true | 21.60 | 0.00 | 21.60 | 288 | 2097349 |
| ai_go | 5 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 5 | get_form | warm | true | 1.01 | 0.00 | 1.01 | 311 | 130 |
| ai_go | 5 | link_cookie | warm | true | 0.00 | 0.00 | 0.00 | 293 | 130 |
| ai_go | 5 | spa_click_fill | warm | true | 629.01 | 0.00 | 629.01 | 633 | 697 |
| ai_go | 5 | async_dom | warm | true | 907.59 | 0.00 | 907.59 | 326 | 342 |
| chromium_cdp | 5 | static | cold | true | 58.12 | 350.99 | 409.12 | 309 | 192 |
| chromium_cdp | 5 | large_html | warm | true | 62.07 | 0.00 | 62.07 | 220 | 2097334 |
| chromium_cdp | 5 | unicode | warm | true | 41.75 | 0.00 | 41.75 | 302 | 125 |
| chromium_cdp | 5 | get_form | warm | true | 121.69 | 0.00 | 121.69 | 244 | 115 |
| chromium_cdp | 5 | link_cookie | warm | true | 119.71 | 0.00 | 119.71 | 218 | 115 |
| chromium_cdp | 5 | spa_click_fill | warm | true | 190.12 | 0.00 | 190.12 | 377 | 493 |
| chromium_cdp | 5 | async_dom | warm | true | 849.37 | 0.00 | 849.37 | 206 | 327 |

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
