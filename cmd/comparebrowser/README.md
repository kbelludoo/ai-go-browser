# Reproducible browser task comparison

`comparebrowser` compares AI Go with a Chromium CDP baseline using the same Chromium binary and local HTTP fixtures. It checks the result of each task rather than treating navigation or HTTP 200 as proof of completion.

```powershell
go build -o comparebrowser.exe ./cmd/comparebrowser
.\comparebrowser.exe -runs 3 -json benchmark-results/comparison.json -markdown benchmark-results/comparison.md
```

Pass `-browser "C:\path\to\chrome.exe"` to select a binary. By default the command finds the installed Chromium/Chrome/Edge browser. The default fixtures do not access external websites and require no API key or LLM.

The seven tasks cover visible static content, a 2 MiB HTML payload, Unicode, GET form input/submission, a link whose destination requires a cookie, SPA input and a click with an asynchronous update, and DOM content inserted 800 ms after navigation. Dynamic fixtures use AI Go's explicit `RenderJS` option; `AutoRenderJS` is also enabled. Values, selectors and expected results are identical for both adapters.

Every engine and repetition runs in a separate worker process. Engine order alternates between repetitions. The first row includes setup time and is labelled `cold`; later rows are `warm` and reuse the engine. AI Go starts Chromium lazily, so a warm row can still contain its first Chromium launch; the JSON field `chromium_started_in_task` identifies this. Timings include actions, snapshot extraction and assertion polling; OS memory sampling runs afterwards.

The baseline uses `LaunchStealth`, `NavigateReady` and `Evaluate` from this repository to control real Chromium. It extracts a compact DOM snapshot with visible text, headings, links, buttons and forms, rather than sending raw HTML as its only option. AI Go uses `CompactSnapshots`, matching the CLI default and omitting automatic Markdown generation. Both snapshots cap text at 5000 Unicode code points, links at 200 and headings at 100. Snapshot schema differences are reported as a limitation. The document byte metric is fetched or serialized HTML; it is not total network traffic or a token count.

`NavigateReady` waits for the lifecycle load event associated with the current navigation's loader. It adds no fixed settling delay; readiness for asynchronous content is determined by polling the same task assertion. The legacy `Navigate` method and its 250 ms delay are not used by the primary comparison.

Memory columns separate live Go heap, Go runtime reservation (`GoSys`), the worker's resident OS working set and Chromium's descendant process tree. Tree working sets are summed from the same post-task sampling pass; shared pages may be counted more than once. Maximum sampled values are not continuously observed peaks. Missing readings remain `null`/`unavailable`, not zero. Workers also retain the fixture server and its payload, so the Go measurements include this fixed harness cost.

The process exits with a failure status when any task assertion fails, after writing both reports. A report retains per-case errors and unsuccessful samples. The corpus does not establish universal site compatibility, universal superiority, or performance against every alternative agent framework. Public smoke checks assert nonempty visible content and reject reported HTTP errors and common challenge phrases, while recording titles, text excerpts and final URLs. This does not establish semantic access through every challenge.

External smoke comparisons require URLs supplied explicitly:

```powershell
.\comparebrowser.exe -public -url https://example.com/ -runs 3 -json benchmark-results/public.json -markdown benchmark-results/public.md
```

Public mode performs read-only navigation and snapshot checks. It does not submit forms or perform account actions. There is no built-in list of public websites.

Run deterministic tests with `go test ./pkg/benchmark ./cmd/comparebrowser`. To additionally assert all real Chromium fixtures, set `AI_BROWSER_INTEGRATION=1` and run `go test ./pkg/benchmark -run TestBrowserTaskComparisonIntegration -v`.
