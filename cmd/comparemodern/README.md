# Modern DOM tasks: AI Go and Playwright

`comparemodern` compares the stacks on a separate `local-modern-v1` corpus. It leaves the original seven-task `local-v1` corpus and its reports unchanged. Both engines receive the same Chromium/Chrome/Edge executable, selectors, values and ordered task assertions. All pages and submissions use two local test servers; the command visits no public sites, uses no credentials or LLM, and installs no packages or browser binaries.

| Task | Required evidence |
|---|---|
| Open shadow root and projected slot | Fill the shadow input; click the shadow button and the slotted button. Each click must be trusted exactly once, with zero untrusted clicks, the expected value and the slot in the event's composed path. |
| Contenteditable | Fill Unicode text; require a trusted input event and one trusted send click, zero untrusted clicks and the exact value. |
| Native validation and multipart | First click the required-email form while invalid: one native invalid event, one trusted click and zero submit events. Fill the fields and click again: exactly one valid multipart POST, submitter value preserved, two trusted clicks, one submit event and zero files. |
| Same-origin frame | Discover a named child frame, fill and click inside it, require one valid local POST and observe the result in the child snapshot. |
| Cross-origin frame | Repeat the child task on a second local server using `localhost` while the parent uses `127.0.0.1`; host and port differ. Require child-scoped result and exactly one valid local POST. |

```powershell
go build -o bin/comparemodern.exe ./cmd/comparemodern
.\bin\comparemodern.exe -runs 3 -json benchmark-results/modern-comparison.json -markdown benchmark-results/modern-comparison.md
```

Node must be on `PATH`, or supplied with `-node`. The command reuses the installed dependency probe from `compareplaywright`: pass `-playwright-module` if the package is elsewhere. Missing dependencies fail explicitly without installation. `-browser` selects the same executable for both stacks. [Playwright's custom executable option](https://playwright.dev/docs/api/class-browsertype#browser-type-launch) does not guarantee compatibility with every browser build; the report records the actual versions exercised.

AI Go uses its public `open`, `query`, `fill`, `press`, `frames`, `frame` and `snapshot` commands. Every task explicitly requests rendering and compact snapshots. The adapter discovers frames by name, checks the allowlist and uses returned references. Both multipart clicks include explicit confirmation. It does not bypass the browser's action or submission policies.

AI Go now fills text-family inputs (`text`, `email`, `number`, `password`, `search`, `tel`, `url`) and textareas through native CDP editing, including controls in open shadow roots and selected frames. Filling an ordinary text control with an empty value uses a native Delete key; contenteditable uses native CDP `Input.insertText`. Rich contenteditable preserves Chromium's paragraph model and reports the native rendered outcome, including canceled edits; rendered `innerText` may contain more line breaks than the requested string. This behavior is covered by browser tests, outside this corpus. Select and other special control types still use property setters with synthetic input/change events. `press` uses native mouse input. Trusted input, `beforeinput` cancellation and focus behavior for ordinary text controls are checked by the browser test suite. Historical and current modern fixtures preserve their original assertions: shadow requires trusted clicks, contenteditable requires trusted input, and ordinary form-field input events are not asserted as trusted. Historical measurements describe the implementation used when they were recorded.

Playwright uses normal headless launch defaults and [native locator fill/click](https://playwright.dev/docs/api/class-locator#locator-fill), including contenteditable. [Locators traverse open shadow roots](https://playwright.dev/docs/locators#locate-in-shadow-dom); [FrameLocator](https://playwright.dev/docs/api/class-framelocator) performs child actions. The harness never edits result/counter fields or dispatches events to manufacture a passing assertion.

The AI Go frame oracle records the original main frame ID/URL and the discovered child identity before selection. Selection and every subsequent snapshot must report the selected child's `Page.FrameRef`. After the actions, the adapter reads the frame tree again and requires the original main ID/URL to remain present, with the same child as the only selected frame. It also checks the child's reference, name, URL and ancestry. A main tab navigated to the child's URL therefore cannot pass merely by displaying the expected result.

Navigation waits for load, and visible results are checked every 50 ms without a fixed settling delay. Fixture HTTP counts cover designated submission/receipt endpoints, excluding page/frame/asset loads. They are recorded independently in the local server, so a duplicate or malformed receipt fails even if the page displays the expected text. A zero-receipt check on the shadow/contenteditable tasks only verifies the submission count; their trusted-event and value assertions come from their visible result.

Each repetition runs both stacks in separate fresh Go workers, alternating engine order. Both workers host the same fixtures; Playwright additionally launches Node. Task duration uses the Go controller wall clock for both engines, including Playwright IPC. Engine setup is reported separately and counted once in the corpus total. AI Go starts Chromium lazily during the first task; Playwright starts it during engine setup. Complete worker wall time includes process startup, fixtures, tasks, memory sampling, cleanup and serialization, so it includes both startup paths.

The Playwright snapshot reads the selected document's visible composed tree, visiting open shadow roots and projected slot content once; it does not aggregate child text into the parent. Text is capped at 5000 Unicode code points, headings at 100 and links/buttons/forms at 200. AI Go's compact schema contains different fields. JSON byte counts are measured UTF-8 sizes, not equivalent schemas or model tokens. Playwright `document_bytes` is light-DOM HTML without shadow roots, while AI Go page `Bytes` measures its composed observation with shadow content and reference metadata; those values represent different serializations.

Memory uses the same OS collector for both engines. Go worker RSS, [Node RSS and V8 heap](https://nodejs.org/api/process.html#processmemoryusage), and Chromium descendant working sets remain separate. GoSys is not RSS. Values are post-task samples, not continuous peaks; shared pages can be counted more than once across Chromium processes. Separate maxima must not be added as if they were simultaneous physical-memory consumption. Missing memory stays absent/null. Startup/transport failures retain all five tasks in the planned denominator and label skipped tasks `not_run`.

```powershell
go test ./pkg/benchmark ./cmd/comparemodern -run '^TestModern' -count=1
# Optional integration check; launches both engines sequentially.
$env:AI_BROWSER_MODERN_INTEGRATION = '1'
go test ./pkg/benchmark -run '^TestModernLocalTasksIntegration$' -v -count=1
```

Results cover these five tasks. They do not establish universal website access or superiority. Closed shadow roots, file uploads, nested frames, cross-origin authentication, arbitrary browser builds and custom frameworks are outside this corpus.
