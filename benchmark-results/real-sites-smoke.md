# Browser task comparison

Generated: 2026-10-11T18:48:46Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 1 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 2/3 | 13939.07 | 0.00 | 34653 | 36.67 | 58.43 | unavailable |
| chromium_cdp | 2/3 | 4819.85 | 408.94 | 33400 | 1.61 | 20.84 | 623.33 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | false | 10000.81 | 0.00 | 10000.81 | 0 | 0 |
| ai_go | 1 | public_2 | warm | true | 1748.06 | 0.00 | 1748.06 | 33849 | 2017426 |
| ai_go | 1 | public_3 | warm | true | 2190.20 | 0.00 | 2190.20 | 35457 | 1879661 |
| chromium_cdp | 1 | public_1 | cold | false | 22.78 | 408.94 | 431.71 | 0 | 0 |
| chromium_cdp | 1 | public_2 | warm | true | 2575.32 | 0.00 | 2575.32 | 31834 | 2202658 |
| chromium_cdp | 1 | public_3 | warm | true | 1812.81 | 0.00 | 1812.81 | 34966 | 2077345 |

## Errors

- ai_go, run 1, public_1: open: Get "https://en.wikipedia.org/wiki/Main_Page": dial tcp: lookup en.wikipedia.org: i/o timeout
- chromium_cdp, run 1, public_1: navigate https://en.wikipedia.org/wiki/Main_Page: net::ERR_NAME_NOT_RESOLVED

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
