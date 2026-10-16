# AI Go versus Playwright with the same Chromium executable

This command compares AI Go and the installed Playwright library on the unchanged `local-v1` corpus from `comparebrowser`: static content, 2 MiB HTML, Unicode, a GET form, a cookie-preserving link, SPA fill/click and delayed DOM content. It runs no public websites, requires no credentials or LLM, and installs no dependencies or browser binaries.

```powershell
go build -o bin/compareplaywright.exe ./cmd/compareplaywright
.\bin\compareplaywright.exe -runs 3 -json benchmark-results/playwright-comparison.json -markdown benchmark-results/playwright-comparison.md
```

Node must be on `PATH`, or provided with `-node`. The command probes project packages and the bundled Node layout. If Playwright is elsewhere, pass `-playwright-module "C:\path\to\node_modules\playwright"`. Missing dependencies produce an explicit error rather than an automatic installation. Pass `-browser` to select the same Chromium/Chrome/Edge executable for both engines.

Playwright uses its normal headless launch defaults and a fresh browser context. AI Go uses its existing stealth configuration, so launch flags differ between the stacks despite the shared executable. [The official launch API supports `executablePath`, while cautioning that arbitrary Chromium versions are not guaranteed compatible](https://playwright.dev/docs/api/class-browsertype#browser-type-launch). The report records the actual Playwright, Node and Chromium versions exercised.

AI Go now starts headless Chromium with `--no-startup-window`, then creates its one controlled page through CDP. Headless sessions with a temporary profile also disable background networking, component extensions with background pages, component updates, default apps, extensions, sync and service autorun, while enabling metrics recording only. These switches are also used by [Playwright's Chromium launcher](https://github.com/microsoft/playwright/blob/main/packages/playwright-core/src/server/chromium/chromiumSwitches.ts). Persistent profiles and visible sessions retain their extension and synchronization settings. The change does not add experimental feature lists or change JavaScript, rendering, sandboxing, certificates or service workers.

The first three-run report, produced before this launch adjustment, is preserved as `benchmark-results/playwright-comparison-initial-defaults.json` and `.md`. It records the earlier configuration and should not be presented as a measurement of the current launcher. Rebuilding this command and running it again writes the current comparison to the standard report paths.

Playwright performs [native locator input and click operations](https://playwright.dev/docs/api/class-locator#locator-fill). Navigation waits for `load`; task completion polls the expected visible text every 50 ms. There is no fixed settling delay. Both adapters execute the same task values and assertions. The Playwright compact snapshot uses exactly the existing CDP extraction expression, capped at 5000 Unicode code points and 200 links. AI Go's snapshot schema differs, so byte counts should be interpreted with that distinction.

Each repetition runs each engine in a fresh Go worker, with engine order alternating. The Go server hosting fixtures is present for both. A Playwright worker also launches Node. Reports show task corpus time including engine construction, engine setup time and complete worker wall time. The last metric includes runtime startup, fixture setup, post-task memory samples, cleanup and serialization for both engines; it avoids treating `Browser.New` alone as the complete cold cost of AI Go. AI Go can launch Chromium lazily on its first dynamic task.

Memory readings separate the Go worker, Node and Chromium. [Node's `process.memoryUsage()` supplies V8 heap use and resident process RSS](https://nodejs.org/api/process.html#processmemoryusage); Chromium process identity comes from a browser CDP session, then the shared OS sampler reads its descendant tree. GoSys remains runtime reservation rather than RSS. These are post-task samples, not continuous peaks. Shared pages can appear in multiple Chromium working sets, and separate maximum readings must not be added and presented as a simultaneous physical-memory peak.

AI Go identifies its Chromium root from the launched process; Playwright identifies its root through CDP `SystemInfo.getProcessInfo`. The shared sampler follows OS parent-child relationships and includes every descendant, such as browser helpers, rather than filtering by executable name. Existing reports retain the sum and source but do not retain per-process IDs, counts or roles. They therefore show the measured difference without attributing it to a specific renderer, extension or utility process. Per-task samples also occur at different browser ages because AI Go launches Chromium lazily.

Any failed assertion remains in both JSON and Markdown and produces a failing exit code after the reports are written. Tasks skipped after a worker failure remain in the planned denominator and carry `not_run: true`; absent memory readings remain absent. Results describe this local corpus and do not establish universal site access or universal superiority.

```powershell
go test ./pkg/benchmark ./cmd/compareplaywright
$env:AI_BROWSER_PLAYWRIGHT_INTEGRATION = '1'
go test ./pkg/benchmark -run TestPlaywrightLocalTasksIntegration -v
```
