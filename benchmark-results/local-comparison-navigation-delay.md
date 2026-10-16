# Browser task comparison

Generated: 2026-10-10T08:14:30Z. Mode: local fixtures. Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 1947.34 | 0.00 | 362 | 16.26 | 30.89 | 860.83 |
| chromium_cdp | 21/21 | 3490.97 | 556.75 | 231 | 5.37 | 18.84 | 965.12 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | static | cold | true | 2.20 | 0.00 | 2.20 | 506 | 207 |
| ai_go | 1 | large_html | warm | true | 21.89 | 0.00 | 21.89 | 338 | 2097349 |
| ai_go | 1 | unicode | warm | true | 0.52 | 0.00 | 0.52 | 362 | 140 |
| ai_go | 1 | get_form | warm | true | 0.00 | 0.00 | 0.00 | 346 | 130 |
| ai_go | 1 | link_cookie | warm | true | 1.01 | 0.00 | 1.01 | 326 | 130 |
| ai_go | 1 | spa_click_fill | warm | true | 1065.70 | 0.00 | 1065.70 | 624 | 821 |
| ai_go | 1 | async_dom | warm | true | 853.64 | 0.00 | 853.64 | 369 | 571 |
| chromium_cdp | 1 | static | cold | true | 294.83 | 560.20 | 855.03 | 296 | 192 |
| chromium_cdp | 1 | large_html | warm | true | 306.46 | 0.00 | 306.46 | 207 | 2097334 |
| chromium_cdp | 1 | unicode | warm | true | 302.28 | 0.00 | 302.28 | 289 | 125 |
| chromium_cdp | 1 | get_form | warm | true | 368.91 | 0.00 | 368.91 | 231 | 115 |
| chromium_cdp | 1 | link_cookie | warm | true | 346.59 | 0.00 | 346.59 | 205 | 115 |
| chromium_cdp | 1 | spa_click_fill | warm | true | 424.28 | 0.00 | 424.28 | 364 | 493 |
| chromium_cdp | 1 | async_dom | warm | true | 881.14 | 0.00 | 881.14 | 193 | 327 |
| chromium_cdp | 2 | static | cold | true | 294.96 | 546.53 | 841.48 | 296 | 192 |
| chromium_cdp | 2 | large_html | warm | true | 306.13 | 0.00 | 306.13 | 207 | 2097334 |
| chromium_cdp | 2 | unicode | warm | true | 297.03 | 0.00 | 297.03 | 289 | 125 |
| chromium_cdp | 2 | get_form | warm | true | 364.15 | 0.00 | 364.15 | 231 | 115 |
| chromium_cdp | 2 | link_cookie | warm | true | 342.95 | 0.00 | 342.95 | 205 | 115 |
| chromium_cdp | 2 | spa_click_fill | warm | true | 417.41 | 0.00 | 417.41 | 364 | 493 |
| chromium_cdp | 2 | async_dom | warm | true | 873.16 | 0.00 | 873.16 | 193 | 327 |
| ai_go | 2 | static | cold | true | 1.55 | 0.00 | 1.55 | 506 | 207 |
| ai_go | 2 | large_html | warm | true | 21.19 | 0.00 | 21.19 | 338 | 2097349 |
| ai_go | 2 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 362 | 140 |
| ai_go | 2 | get_form | warm | true | 0.55 | 0.00 | 0.55 | 346 | 130 |
| ai_go | 2 | link_cookie | warm | true | 0.52 | 0.00 | 0.52 | 326 | 130 |
| ai_go | 2 | spa_click_fill | warm | true | 1045.19 | 0.00 | 1045.19 | 624 | 821 |
| ai_go | 2 | async_dom | warm | true | 884.39 | 0.00 | 884.39 | 369 | 571 |
| ai_go | 3 | static | cold | true | 1.66 | 0.00 | 1.66 | 506 | 207 |
| ai_go | 3 | large_html | warm | true | 21.92 | 0.00 | 21.92 | 338 | 2097349 |
| ai_go | 3 | unicode | warm | true | 0.54 | 0.00 | 0.54 | 362 | 140 |
| ai_go | 3 | get_form | warm | true | 0.52 | 0.00 | 0.52 | 346 | 130 |
| ai_go | 3 | link_cookie | warm | true | 0.57 | 0.00 | 0.57 | 326 | 130 |
| ai_go | 3 | spa_click_fill | warm | true | 1048.18 | 0.00 | 1048.18 | 624 | 821 |
| ai_go | 3 | async_dom | warm | true | 869.76 | 0.00 | 869.76 | 369 | 571 |
| chromium_cdp | 3 | static | cold | true | 291.73 | 563.51 | 855.24 | 296 | 192 |
| chromium_cdp | 3 | large_html | warm | true | 317.08 | 0.00 | 317.08 | 207 | 2097334 |
| chromium_cdp | 3 | unicode | warm | true | 368.13 | 0.00 | 368.13 | 289 | 125 |
| chromium_cdp | 3 | get_form | warm | true | 366.44 | 0.00 | 366.44 | 231 | 115 |
| chromium_cdp | 3 | link_cookie | warm | true | 345.49 | 0.00 | 345.49 | 205 | 115 |
| chromium_cdp | 3 | spa_click_fill | warm | true | 417.60 | 0.00 | 417.60 | 364 | 493 |
| chromium_cdp | 3 | async_dom | warm | true | 875.91 | 0.00 | 875.91 | 193 | 327 |

## Interpretation limits

- This corpus cannot prove access to every website or universal superiority. No external AI model is run and token counts are not measured.
- Each engine and repetition runs in a fresh isolated Go worker. The first case includes engine setup; subsequent cases reuse that engine. The first JS case may still include a lazy Chromium launch for AI Go.
- The baseline is Chromium controlled through the existing CDP/stealth launcher, including its navigation waits and patches; it is not an unmodified GUI browser or a comparison against other agent frameworks.
- The current Navigate wrapper adds a fixed 250 ms settling delay per navigation. Baseline task times include that delay; a tuned CDP client with condition-based waiting could reduce the static-page timing gap.
- Compact snapshot bytes are actual UTF-8 JSON lengths. Both adapters cap visible text at 5000 Unicode code points and links at 200; schemas differ. Document bytes represent fetched/serialized HTML, not model input tokens or total network traffic.
- GoHeapAlloc and GoSys belong to the benchmark worker and include fixtures/adapter allocations. GoSys is runtime-reserved memory, not RSS. Harness RSS and Chromium tree RSS are separate OS samples after each task, not continuously measured peaks.
- Chromium tree RSS sums resident working sets of process descendants. Shared pages can be counted multiple times, processes can exit during sampling, and unavailable readings remain null rather than zero.
- Per-case contexts bound CDP operations and assertion polling; AI Go commands also use the browser's internal request/render timeouts and may exceed the requested case timeout.
- Results depend on hardware, OS, Chromium version, local caches and task order. The command alternates engine order across repetitions; aggregate medians across different task types describe this fixed corpus only.
- Very short local HTTP tasks can round to zero at the host timer resolution; these readings do not mean that the operation has no cost.
