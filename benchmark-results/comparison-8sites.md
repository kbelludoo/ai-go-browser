# Browser task comparison

Generated: 2026-10-16T03:22:23Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 1 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 8/8 | 8059.72 | 0.50 | 22496 | 27.50 | 50.75 | unavailable |
| chromium_cdp | 8/8 | 12479.07 | 392.94 | 17842 | 2.38 | 12.48 | 1194.52 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 1391.55 | 0.50 | 1392.05 | 20414 | 64185 |
| ai_go | 1 | public_2 | warm | true | 706.53 | 0.00 | 706.53 | 24578 | 34563 |
| ai_go | 1 | public_3 | warm | true | 663.37 | 0.00 | 663.37 | 20377 | 51294 |
| ai_go | 1 | public_4 | warm | true | 125.83 | 0.00 | 125.83 | 26615 | 333358 |
| ai_go | 1 | public_5 | warm | true | 1120.51 | 0.00 | 1120.51 | 3858 | 3741 |
| ai_go | 1 | public_6 | warm | true | 2323.89 | 0.00 | 2323.89 | 38509 | 2266686 |
| ai_go | 1 | public_7 | warm | true | 1673.80 | 0.00 | 1673.80 | 24918 | 572476 |
| ai_go | 1 | public_8 | warm | true | 53.74 | 0.00 | 53.74 | 12699 | 83785 |
| chromium_cdp | 1 | public_1 | cold | true | 2433.12 | 392.94 | 2826.06 | 17008 | 62707 |
| chromium_cdp | 1 | public_2 | warm | true | 939.24 | 0.00 | 939.24 | 21841 | 34544 |
| chromium_cdp | 1 | public_3 | warm | true | 980.39 | 0.00 | 980.39 | 16267 | 51009 |
| chromium_cdp | 1 | public_4 | warm | true | 331.75 | 0.00 | 331.75 | 24463 | 333325 |
| chromium_cdp | 1 | public_5 | warm | true | 1109.91 | 0.00 | 1109.91 | 3789 | 3722 |
| chromium_cdp | 1 | public_6 | warm | true | 3285.14 | 0.00 | 3285.14 | 38692 | 2779290 |
| chromium_cdp | 1 | public_7 | warm | true | 2051.70 | 0.00 | 2051.70 | 18677 | 573666 |
| chromium_cdp | 1 | public_8 | warm | true | 954.88 | 0.00 | 954.88 | 9422 | 84936 |

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
