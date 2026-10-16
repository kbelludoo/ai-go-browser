# Browser task comparison

Generated: 2026-10-11T18:32:31Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 5 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 35/35 | 1557.04 | 0.10 | 313 | 18.49 | 31.22 | 372.23 |
| chromium_cdp | 35/35 | 1782.54 | 358.06 | 244 | 7.60 | 18.58 | 403.22 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.20 | 0.50 | 2.71 | 389 | 207 |
| ai_go | 1 | large_html | warm | true | 21.08 | 0.00 | 21.08 | 288 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.97 | 0.00 | 0.97 | 313 | 140 |
| ai_go | 1 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 311 | 130 |
| ai_go | 1 | link_cookie | warm | true | 1.01 | 0.00 | 1.01 | 293 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 631.13 | 0.00 | 631.13 | 633 | 697 |
| ai_go | 1 | async_dom | warm | true | 906.92 | 0.00 | 906.92 | 326 | 342 |
| chromium_cdp | 1 | static | cold | true | 49.75 | 361.16 | 410.92 | 309 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 60.70 | 0.00 | 60.70 | 220 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 49.08 | 0.00 | 49.08 | 302 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 121.54 | 0.00 | 121.54 | 244 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 115.81 | 0.00 | 115.81 | 218 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 186.92 | 0.00 | 186.92 | 377 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 843.74 | 0.00 | 843.74 | 206 | 327 |
| chromium_cdp | 2 | static | cold | true | 50.04 | 356.18 | 406.22 | 309 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 64.33 | 0.00 | 64.33 | 220 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 52.86 | 0.00 | 52.86 | 302 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 119.36 | 0.00 | 119.36 | 244 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 108.78 | 0.00 | 108.78 | 218 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 187.80 | 0.00 | 187.80 | 377 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 851.46 | 0.00 | 851.46 | 206 | 327 |
| ai_go | 2 | static | cold | true | 1.61 | 0.00 | 1.61 | 389 | 207 |
| ai_go | 2 | large_html | warm | true | 21.51 | 0.00 | 21.51 | 288 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 313 | 140 |
| ai_go | 2 | get_form | warm | true | 0.53 | 0.00 | 0.53 | 311 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.56 | 0.00 | 0.56 | 293 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 654.84 | 0.00 | 654.84 | 633 | 697 |
| ai_go | 2 | async_dom | warm | true | 906.66 | 0.00 | 906.66 | 326 | 342 |
| ai_go | 3 | static | cold | true | 1.11 | 0.00 | 1.11 | 389 | 207 |
| ai_go | 3 | large_html | warm | true | 19.79 | 0.00 | 19.79 | 288 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.56 | 0.00 | 0.56 | 313 | 140 |
| ai_go | 3 | get_form | warm | true | 0.52 | 0.00 | 0.52 | 311 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.56 | 0.00 | 0.56 | 293 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 615.04 | 0.00 | 615.04 | 633 | 697 |
| ai_go | 3 | async_dom | warm | true | 906.35 | 0.00 | 906.35 | 326 | 342 |
| chromium_cdp | 3 | static | cold | true | 49.97 | 364.33 | 414.30 | 309 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 59.09 | 0.00 | 59.09 | 220 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 48.17 | 0.00 | 48.17 | 302 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 124.04 | 0.00 | 124.04 | 244 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 111.32 | 0.00 | 111.32 | 218 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 187.21 | 0.00 | 187.21 | 377 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 846.73 | 0.00 | 846.73 | 206 | 327 |
| chromium_cdp | 4 | static | cold | true | 49.08 | 355.30 | 404.37 | 309 | 192 |
| chromium_cdp | 4 | large_html | warm | true | 63.54 | 0.00 | 63.54 | 220 | 2097334 |
| chromium_cdp | 4 | unicode | warm | true | 48.90 | 0.00 | 48.90 | 302 | 125 |
| chromium_cdp | 4 | get_form | warm | true | 119.46 | 0.00 | 119.46 | 244 | 115 |
| chromium_cdp | 4 | link_cookie | warm | true | 102.98 | 0.00 | 102.98 | 218 | 115 |
| chromium_cdp | 4 | spa_click_fill | warm | true | 189.26 | 0.00 | 189.26 | 377 | 493 |
| chromium_cdp | 4 | async_dom | warm | true | 846.89 | 0.00 | 846.89 | 206 | 327 |
| ai_go | 4 | static | cold | true | 1.64 | 0.00 | 1.64 | 389 | 207 |
| ai_go | 4 | large_html | warm | true | 30.52 | 0.00 | 30.52 | 288 | 2097349 |
| ai_go | 4 | unicode | warm | true | 0.00 | 0.00 | 0.00 | 313 | 140 |
| ai_go | 4 | get_form | warm | true | 0.54 | 0.00 | 0.54 | 311 | 130 |
| ai_go | 4 | link_cookie | warm | true | 0.57 | 0.00 | 0.57 | 293 | 130 |
| ai_go | 4 | spa_click_fill | warm | true | 608.39 | 0.00 | 608.39 | 633 | 697 |
| ai_go | 4 | async_dom | warm | true | 907.77 | 0.00 | 907.77 | 326 | 342 |
| ai_go | 5 | static | cold | true | 1.18 | 0.00 | 1.18 | 389 | 207 |
| ai_go | 5 | large_html | warm | true | 21.03 | 0.00 | 21.03 | 288 | 2097349 |
| ai_go | 5 | unicode | warm | true | 0.55 | 0.00 | 0.55 | 313 | 140 |
| ai_go | 5 | get_form | warm | true | 0.56 | 0.00 | 0.56 | 311 | 130 |
| ai_go | 5 | link_cookie | warm | true | 0.54 | 0.00 | 0.54 | 293 | 130 |
| ai_go | 5 | spa_click_fill | warm | true | 612.15 | 0.00 | 612.15 | 633 | 697 |
| ai_go | 5 | async_dom | warm | true | 905.76 | 0.00 | 905.76 | 326 | 342 |
| chromium_cdp | 5 | static | cold | true | 52.03 | 353.33 | 405.36 | 309 | 192 |
| chromium_cdp | 5 | large_html | warm | true | 62.27 | 0.00 | 62.27 | 220 | 2097334 |
| chromium_cdp | 5 | unicode | warm | true | 45.58 | 0.00 | 45.58 | 302 | 125 |
| chromium_cdp | 5 | get_form | warm | true | 96.69 | 0.00 | 96.69 | 244 | 115 |
| chromium_cdp | 5 | link_cookie | warm | true | 118.34 | 0.00 | 118.34 | 218 | 115 |
| chromium_cdp | 5 | spa_click_fill | warm | true | 193.65 | 0.00 | 193.65 | 377 | 493 |
| chromium_cdp | 5 | async_dom | warm | true | 845.02 | 0.00 | 845.02 | 206 | 327 |

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
