# Browser task comparison

Generated: 2026-10-10T15:31:27Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 1 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 7/7 | 5498.86 | 0.00 | 25157 | 11.93 | 33.48 | unavailable |
| chromium_cdp | 7/7 | 12584.65 | 593.50 | 18255 | 2.18 | 12.25 | 1644.38 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 736.47 | 0.00 | 736.47 | 20353 | 64185 |
| ai_go | 1 | public_2 | warm | true | 79.56 | 0.00 | 79.56 | 440 | 577 |
| ai_go | 1 | public_3 | warm | true | 1375.63 | 0.00 | 1375.63 | 34028 | 735464 |
| ai_go | 1 | public_4 | warm | true | 1547.20 | 0.00 | 1547.20 | 27283 | 305078 |
| ai_go | 1 | public_5 | warm | true | 575.92 | 0.00 | 575.92 | 25157 | 34845 |
| ai_go | 1 | public_6 | warm | true | 325.96 | 0.00 | 325.96 | 13861 | 286486 |
| ai_go | 1 | public_7 | warm | true | 858.10 | 0.00 | 858.10 | 33015 | 203910 |
| chromium_cdp | 1 | public_1 | cold | true | 3275.16 | 593.50 | 3868.66 | 17008 | 62705 |
| chromium_cdp | 1 | public_2 | warm | true | 115.54 | 0.00 | 115.54 | 2851 | 2358 |
| chromium_cdp | 1 | public_3 | warm | true | 3292.80 | 0.00 | 3292.80 | 30967 | 1002301 |
| chromium_cdp | 1 | public_4 | warm | true | 2126.66 | 0.00 | 2126.66 | 18255 | 305516 |
| chromium_cdp | 1 | public_5 | warm | true | 988.90 | 0.00 | 988.90 | 22415 | 34879 |
| chromium_cdp | 1 | public_6 | warm | true | 1776.47 | 0.00 | 1776.47 | 9656 | 320340 |
| chromium_cdp | 1 | public_7 | warm | true | 415.62 | 0.00 | 415.62 | 28124 | 121444 |

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

This run overlapped the full test suite. Its navigation assertions remain evidence, but its timing and memory samples are not an idle-machine comparison.
