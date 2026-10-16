# Browser task comparison

Generated: 2026-10-10T15:35:33Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 3 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 3945.95 | 0.00 | 25149 | 12.55 | 33.96 | unavailable |
| chromium_cdp | 21/21 | 12277.74 | 560.35 | 18255 | 2.18 | 12.48 | 1754.91 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 431.44 | 0.00 | 431.44 | 20353 | 64185 |
| ai_go | 1 | public_2 | warm | true | 142.04 | 0.00 | 142.04 | 440 | 577 |
| ai_go | 1 | public_3 | warm | true | 1592.43 | 0.00 | 1592.43 | 34028 | 735464 |
| ai_go | 1 | public_4 | warm | true | 1861.21 | 0.00 | 1861.21 | 27284 | 305079 |
| ai_go | 1 | public_5 | warm | true | 569.76 | 0.00 | 569.76 | 25149 | 34841 |
| ai_go | 1 | public_6 | warm | true | 80.18 | 0.00 | 80.18 | 13861 | 286486 |
| ai_go | 1 | public_7 | warm | true | 71.74 | 0.00 | 71.74 | 33015 | 203910 |
| chromium_cdp | 1 | public_1 | cold | true | 3423.19 | 556.75 | 3979.94 | 17008 | 62707 |
| chromium_cdp | 1 | public_2 | warm | true | 150.82 | 0.00 | 150.82 | 2851 | 2358 |
| chromium_cdp | 1 | public_3 | warm | true | 2554.26 | 0.00 | 2554.26 | 30967 | 1002301 |
| chromium_cdp | 1 | public_4 | warm | true | 2021.05 | 0.00 | 2021.05 | 18255 | 305516 |
| chromium_cdp | 1 | public_5 | warm | true | 958.81 | 0.00 | 958.81 | 22407 | 34875 |
| chromium_cdp | 1 | public_6 | warm | true | 2920.36 | 0.00 | 2920.36 | 8792 | 286772 |
| chromium_cdp | 1 | public_7 | warm | true | 614.33 | 0.00 | 614.33 | 28124 | 121444 |
| chromium_cdp | 2 | public_1 | cold | true | 3435.86 | 556.38 | 3992.24 | 17008 | 62705 |
| chromium_cdp | 2 | public_2 | warm | true | 117.70 | 0.00 | 117.70 | 2851 | 2358 |
| chromium_cdp | 2 | public_3 | warm | true | 2454.38 | 0.00 | 2454.38 | 29613 | 895546 |
| chromium_cdp | 2 | public_4 | warm | true | 2216.48 | 0.00 | 2216.48 | 18255 | 305516 |
| chromium_cdp | 2 | public_5 | warm | true | 1030.14 | 0.00 | 1030.14 | 22407 | 34875 |
| chromium_cdp | 2 | public_6 | warm | true | 1601.60 | 0.00 | 1601.60 | 9656 | 320340 |
| chromium_cdp | 2 | public_7 | warm | true | 366.32 | 0.00 | 366.32 | 28124 | 121444 |
| ai_go | 2 | public_1 | cold | true | 429.41 | 0.00 | 429.41 | 20353 | 64185 |
| ai_go | 2 | public_2 | warm | true | 52.60 | 0.00 | 52.60 | 440 | 577 |
| ai_go | 2 | public_3 | warm | true | 1320.57 | 0.00 | 1320.57 | 34028 | 735464 |
| ai_go | 2 | public_4 | warm | true | 1020.61 | 0.00 | 1020.61 | 27284 | 305079 |
| ai_go | 2 | public_5 | warm | true | 577.83 | 0.00 | 577.83 | 25149 | 34841 |
| ai_go | 2 | public_6 | warm | true | 86.98 | 0.00 | 86.98 | 13861 | 286486 |
| ai_go | 2 | public_7 | warm | true | 57.38 | 0.00 | 57.38 | 33015 | 203910 |
| ai_go | 3 | public_1 | cold | true | 425.55 | 0.00 | 425.55 | 20353 | 64185 |
| ai_go | 3 | public_2 | warm | true | 59.71 | 0.00 | 59.71 | 440 | 577 |
| ai_go | 3 | public_3 | warm | true | 1317.71 | 0.00 | 1317.71 | 34028 | 735464 |
| ai_go | 3 | public_4 | warm | true | 1022.09 | 0.00 | 1022.09 | 27283 | 305078 |
| ai_go | 3 | public_5 | warm | true | 559.99 | 0.00 | 559.99 | 25149 | 34841 |
| ai_go | 3 | public_6 | warm | true | 100.03 | 0.00 | 100.03 | 13861 | 286486 |
| ai_go | 3 | public_7 | warm | true | 58.59 | 0.00 | 58.59 | 33015 | 203910 |
| chromium_cdp | 3 | public_1 | cold | true | 3207.74 | 567.91 | 3775.65 | 17008 | 62707 |
| chromium_cdp | 3 | public_2 | warm | true | 115.40 | 0.00 | 115.40 | 2851 | 2358 |
| chromium_cdp | 3 | public_3 | warm | true | 2738.19 | 0.00 | 2738.19 | 30967 | 1002301 |
| chromium_cdp | 3 | public_4 | warm | true | 2039.13 | 0.00 | 2039.13 | 18255 | 305585 |
| chromium_cdp | 3 | public_5 | warm | true | 992.89 | 0.00 | 992.89 | 22407 | 34875 |
| chromium_cdp | 3 | public_6 | warm | true | 1844.67 | 0.00 | 1844.67 | 9656 | 320340 |
| chromium_cdp | 3 | public_7 | warm | true | 348.85 | 0.00 | 348.85 | 28124 | 121450 |

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
- Public checks require a nonempty document and reject reported HTTP errors and a small set of obvious challenge/error phrases. They do not prove semantic completion or distinguish every challenge from usable content; no forms or account actions are performed.
