# Browser task comparison

Generated: 2026-10-10T08:20:06Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 1 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 7/7 | 5071.68 | 0.00 | 41145 | 12.04 | 31.72 | unavailable |
| chromium_cdp | 7/7 | 12501.85 | 563.58 | 18255 | 2.17 | 12.43 | 1638.23 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 615.87 | 0.00 | 615.87 | 34203 | 64185 |
| ai_go | 1 | public_2 | warm | true | 72.79 | 0.00 | 72.79 | 610 | 577 |
| ai_go | 1 | public_3 | warm | true | 1592.46 | 0.00 | 1592.46 | 306420 | 735463 |
| ai_go | 1 | public_4 | warm | true | 1432.45 | 0.00 | 1432.45 | 41145 | 305079 |
| ai_go | 1 | public_5 | warm | true | 796.94 | 0.00 | 796.94 | 42781 | 34706 |
| ai_go | 1 | public_6 | warm | true | 314.55 | 0.00 | 314.55 | 19927 | 286486 |
| ai_go | 1 | public_7 | warm | true | 246.61 | 0.00 | 246.61 | 170458 | 203910 |
| chromium_cdp | 1 | public_1 | cold | true | 3191.73 | 563.58 | 3755.31 | 17008 | 62705 |
| chromium_cdp | 1 | public_2 | warm | true | 112.07 | 0.00 | 112.07 | 2851 | 2358 |
| chromium_cdp | 1 | public_3 | warm | true | 3418.07 | 0.00 | 3418.07 | 30967 | 1002300 |
| chromium_cdp | 1 | public_4 | warm | true | 1982.21 | 0.00 | 1982.21 | 18255 | 305516 |
| chromium_cdp | 1 | public_5 | warm | true | 944.99 | 0.00 | 944.99 | 22202 | 34740 |
| chromium_cdp | 1 | public_6 | warm | true | 1920.22 | 0.00 | 1920.22 | 9656 | 320340 |
| chromium_cdp | 1 | public_7 | warm | true | 368.97 | 0.00 | 368.97 | 28124 | 121448 |

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
- Public checks require a nonempty document and reject reported HTTP errors and a small set of obvious challenge/error phrases. They do not prove semantic completion or distinguish every challenge from usable content; no forms or account actions are performed.
