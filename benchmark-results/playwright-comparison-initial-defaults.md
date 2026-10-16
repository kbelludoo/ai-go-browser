# AI Go and Playwright task comparison

Generated: 2026-10-10T15:57:59Z. Fixture corpus: local-v1 unchanged. Repetitions per engine: 3.

Same Chromium executable: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`. Environment: windows/amd64, go1.27.0.

Playwright: 1.62.1; Node: v24.19.0; Chromium: 155.0.4283.45.

| Engine | Assertions passed | Mean task corpus incl. engine setup (ms) | Mean complete worker wall time (ms) | Mean engine setup (ms) | Median compact JSON bytes |
|---|---:|---:|---:|---:|---:|
| ai_go | 21/21 | 1925.68 | 2438.50 | 0.01 | 311 |
| playwright_chromium | 21/21 | 2483.89 | 2855.28 | 847.52 | 236 |

## Separate post-task memory samples

The Go worker includes the fixture server. The Playwright worker additionally owns Node. Values are maximum sampled readings, not continuous peaks; do not sum separate maxima. Chromium sums can include shared pages more than once.

| Engine | Go worker heap (MiB) | Go worker RSS (MiB) | Node V8 heap (MiB) | Node RSS (MiB) | Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|
| ai_go | 16.38 | 31.01 | unavailable | unavailable | 860.54 |
| playwright_chromium | 7.22 | 18.23 | 55.83 | 118.54 | 522.46 |

## Task results

Cold is the first task after engine construction. AI Go may still start Chromium lazily on a later dynamic task.

| Engine | Run | Task | Phase | Passed | Not run | Task ms | Engine setup ms | JSON bytes |
|---|---:|---|---|---|---|---:|---:|---:|
| ai_go | 1 | static | cold | true | false | 1.06 | 0.04 | 389 |
| ai_go | 1 | large_html | warm | true | false | 20.90 | 0.00 | 288 |
| ai_go | 1 | unicode | warm | true | false | 0.00 | 0.00 | 313 |
| ai_go | 1 | get_form | warm | true | false | 0.52 | 0.00 | 311 |
| ai_go | 1 | link_cookie | warm | true | false | 0.07 | 0.00 | 293 |
| ai_go | 1 | spa_click_fill | warm | true | false | 1037.82 | 0.00 | 615 |
| ai_go | 1 | async_dom | warm | true | false | 877.15 | 0.00 | 308 |
| playwright_chromium | 1 | static | cold | true | false | 60.85 | 835.22 | 309 |
| playwright_chromium | 1 | large_html | warm | true | false | 75.06 | 0.00 | 220 |
| playwright_chromium | 1 | unicode | warm | true | false | 57.60 | 0.00 | 238 |
| playwright_chromium | 1 | get_form | warm | true | false | 236.24 | 0.00 | 236 |
| playwright_chromium | 1 | link_cookie | warm | true | false | 106.42 | 0.00 | 218 |
| playwright_chromium | 1 | spa_click_fill | warm | true | false | 248.78 | 0.00 | 369 |
| playwright_chromium | 1 | async_dom | warm | true | false | 858.62 | 0.00 | 206 |
| playwright_chromium | 2 | static | cold | true | false | 60.79 | 846.11 | 309 |
| playwright_chromium | 2 | large_html | warm | true | false | 76.18 | 0.00 | 220 |
| playwright_chromium | 2 | unicode | warm | true | false | 54.70 | 0.00 | 238 |
| playwright_chromium | 2 | get_form | warm | true | false | 220.41 | 0.00 | 236 |
| playwright_chromium | 2 | link_cookie | warm | true | false | 115.99 | 0.00 | 218 |
| playwright_chromium | 2 | spa_click_fill | warm | true | false | 246.10 | 0.00 | 369 |
| playwright_chromium | 2 | async_dom | warm | true | false | 859.79 | 0.00 | 206 |
| ai_go | 2 | static | cold | true | false | 1.61 | 0.00 | 389 |
| ai_go | 2 | large_html | warm | true | false | 19.63 | 0.00 | 288 |
| ai_go | 2 | unicode | warm | true | false | 0.53 | 0.00 | 313 |
| ai_go | 2 | get_form | warm | true | false | 0.54 | 0.00 | 311 |
| ai_go | 2 | link_cookie | warm | true | false | 0.56 | 0.00 | 293 |
| ai_go | 2 | spa_click_fill | warm | true | false | 1038.15 | 0.00 | 615 |
| ai_go | 2 | async_dom | warm | true | false | 838.48 | 0.00 | 308 |
| ai_go | 3 | static | cold | true | false | 1.61 | 0.00 | 389 |
| ai_go | 3 | large_html | warm | true | false | 19.73 | 0.00 | 288 |
| ai_go | 3 | unicode | warm | true | false | 0.54 | 0.00 | 313 |
| ai_go | 3 | get_form | warm | true | false | 0.55 | 0.00 | 311 |
| ai_go | 3 | link_cookie | warm | true | false | 0.00 | 0.00 | 293 |
| ai_go | 3 | spa_click_fill | warm | true | false | 1048.10 | 0.00 | 615 |
| ai_go | 3 | async_dom | warm | true | false | 869.46 | 0.00 | 308 |
| playwright_chromium | 3 | static | cold | true | false | 56.44 | 861.24 | 309 |
| playwright_chromium | 3 | large_html | warm | true | false | 75.66 | 0.00 | 220 |
| playwright_chromium | 3 | unicode | warm | true | false | 56.29 | 0.00 | 238 |
| playwright_chromium | 3 | get_form | warm | true | false | 228.41 | 0.00 | 236 |
| playwright_chromium | 3 | link_cookie | warm | true | false | 119.68 | 0.00 | 218 |
| playwright_chromium | 3 | spa_click_fill | warm | true | false | 241.65 | 0.00 | 369 |
| playwright_chromium | 3 | async_dom | warm | true | false | 853.45 | 0.00 | 206 |

## Interpretation limits

- Results describe these seven local tasks, not universal website access or superiority. No public sites, credentials, LLMs, model tokens or authenticated APIs are used.
- The same Chromium executable path is passed to both engines. AI Go uses its existing stealth launch configuration; Playwright uses its normal launch defaults. Launch flags therefore differ between the stacks. Playwright uses native locator fill/click actions, page.goto(waitUntil: load), and a 50 ms assertion poll without arbitrary settling delays. AI Go enables AutoRenderJS and explicit RenderJS for dynamic fixtures.
- Playwright's official documentation cautions that a custom executable may not be compatible with every Playwright version. The report records the actual Node, Playwright and Chromium versions tested; this command does not download browsers or install packages.
- Playwright and AI Go each run in a fresh Go worker per repetition; engine order alternates. Playwright's worker additionally owns one Node process. The Go fixture server is present in both workers. Worker wall time includes process startup, fixture setup, tasks, post-task memory sampling, cleanup and serialization; engine/task time excludes post-task memory sampling and cleanup.
- Playwright engine setup includes launching Node, loading Playwright and launching Chromium. AI Go engine setup is Browser.New in an already running worker, with Chromium launched lazily on its first dynamic task. Worker wall time is therefore the separate cold end-to-end comparison.
- The Playwright compact snapshot uses exactly the existing CDP baseline extraction expression, with visible text capped at 5000 Unicode code points and links at 200. AI Go's page schema contains different fields; UTF-8 JSON byte counts are observed sizes, not a claim of equivalent schemas or model token counts.
- Go heap and GoSys include the Go fixture/controller overhead; for AI Go they also include its adapter. Node heap is V8 heapUsed; Node RSS is process.memoryUsage().rss. GoSys is not RSS. Maximum values are maxima of post-task samples, not continuously observed peaks.
- Go worker RSS, Node RSS and Chromium tree RSS are reported separately. The tree sums resident working sets and can double-count shared pages. Separate sampled maxima must not be added and presented as a simultaneous physical-memory peak. Unavailable metrics remain null.
- Very short local tasks may round to zero at host timer resolution. Timing varies with hardware, caches, Chromium build and concurrent workload. Multiple repetitions improve reproducibility but do not eliminate these factors.
- Not-run tasks remain in the planned assertion denominator and are identified separately. Startup and transport failures preserve elapsed time when measurable; absent Node memory readings are omitted rather than recorded as zero.

## Official API references

- [API reference](https://playwright.dev/docs/api/class-browsertype#browser-type-launch)
- [API reference](https://playwright.dev/docs/api/class-page#page-goto)
- [API reference](https://playwright.dev/docs/api/class-locator#locator-fill)
- [API reference](https://playwright.dev/docs/api/class-browser#browser-new-browser-cdp-session)
- [API reference](https://nodejs.org/api/process.html#processmemoryusage)
