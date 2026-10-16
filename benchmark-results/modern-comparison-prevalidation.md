# AI Go and Playwright modern DOM comparison

Generated: 2026-10-10T16:59:15Z. Fixture corpus: local-modern-v1. Repetitions per engine: 3.

Same Chromium executable: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`. Environment: windows/amd64, go1.27.0.

Playwright: 1.62.1; Node: v24.19.0; Chromium: 155.0.4283.45.

| Engine | Assertions passed | Mean task corpus incl. engine setup (ms) | Mean complete worker wall time (ms) | Mean engine setup (ms) | Median compact JSON bytes |
|---|---:|---:|---:|---:|---:|
| ai_go | 15/15 | 2085.27 | 2706.02 | 0.00 | 652 |
| playwright_chromium | 15/15 | 2201.52 | 2572.77 | 970.54 | 385 |

## Separate post-task memory samples

The Go worker includes the fixture server. The Playwright worker additionally owns Node. Values are maximum sampled readings, not continuous peaks; do not sum separate maxima. Chromium sums can include shared pages more than once.

| Engine | Go worker heap (MiB) | Go worker RSS (MiB) | Node V8 heap (MiB) | Node RSS (MiB) | Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|
| ai_go | 2.59 | 17.05 | unavailable | unavailable | 586.00 |
| playwright_chromium | 1.26 | 11.58 | 57.60 | 122.18 | 580.84 |

## Task results

Cold is the first rendered task after engine construction. AI Go launches Chromium lazily inside that task; Playwright launches it during engine setup. Complete worker wall time includes both startup paths.

| Engine | Run | Task | Phase | Passed | Not run | Task ms | Engine setup ms | JSON bytes |
|---|---:|---|---|---|---|---:|---:|---:|
| ai_go | 1 | open_shadow_slot | cold | true | false | 1010.47 | 0.00 | 733 |
| ai_go | 1 | trusted_contenteditable | warm | true | false | 158.99 | 0.00 | 643 |
| ai_go | 1 | native_validation_multipart | warm | true | false | 410.68 | 0.00 | 485 |
| ai_go | 1 | frame_same_origin | warm | true | false | 196.93 | 0.00 | 652 |
| ai_go | 1 | frame_cross_origin | warm | true | false | 299.47 | 0.00 | 656 |
| playwright_chromium | 1 | open_shadow_slot | cold | true | false | 356.57 | 955.48 | 471 |
| playwright_chromium | 1 | trusted_contenteditable | warm | true | false | 103.54 | 0.00 | 385 |
| playwright_chromium | 1 | native_validation_multipart | warm | true | false | 319.08 | 0.00 | 368 |
| playwright_chromium | 1 | frame_same_origin | warm | true | false | 245.80 | 0.00 | 383 |
| playwright_chromium | 1 | frame_cross_origin | warm | true | false | 212.92 | 0.00 | 386 |
| playwright_chromium | 2 | open_shadow_slot | cold | true | false | 352.98 | 963.14 | 471 |
| playwright_chromium | 2 | trusted_contenteditable | warm | true | false | 104.38 | 0.00 | 385 |
| playwright_chromium | 2 | native_validation_multipart | warm | true | false | 319.46 | 0.00 | 368 |
| playwright_chromium | 2 | frame_same_origin | warm | true | false | 174.11 | 0.00 | 383 |
| playwright_chromium | 2 | frame_cross_origin | warm | true | false | 280.89 | 0.00 | 386 |
| ai_go | 2 | open_shadow_slot | cold | true | false | 983.65 | 0.00 | 733 |
| ai_go | 2 | trusted_contenteditable | warm | true | false | 149.01 | 0.00 | 643 |
| ai_go | 2 | native_validation_multipart | warm | true | false | 480.76 | 0.00 | 485 |
| ai_go | 2 | frame_same_origin | warm | true | false | 208.51 | 0.00 | 652 |
| ai_go | 2 | frame_cross_origin | warm | true | false | 245.01 | 0.00 | 656 |
| ai_go | 3 | open_shadow_slot | cold | true | false | 1001.42 | 0.00 | 733 |
| ai_go | 3 | trusted_contenteditable | warm | true | false | 152.88 | 0.00 | 643 |
| ai_go | 3 | native_validation_multipart | warm | true | false | 436.66 | 0.00 | 485 |
| ai_go | 3 | frame_same_origin | warm | true | false | 267.88 | 0.00 | 652 |
| ai_go | 3 | frame_cross_origin | warm | true | false | 253.49 | 0.00 | 656 |
| playwright_chromium | 3 | open_shadow_slot | cold | true | false | 374.06 | 993.01 | 471 |
| playwright_chromium | 3 | trusted_contenteditable | warm | true | false | 115.00 | 0.00 | 385 |
| playwright_chromium | 3 | native_validation_multipart | warm | true | false | 350.00 | 0.00 | 368 |
| playwright_chromium | 3 | frame_same_origin | warm | true | false | 177.64 | 0.00 | 383 |
| playwright_chromium | 3 | frame_cross_origin | warm | true | false | 206.50 | 0.00 | 386 |

## Independent fixture HTTP evidence

Counts cover designated submission/receipt endpoints, excluding page/frame/asset loads. A passed HTTP count supplements the visible event/value assertion. Tasks not attempted have no evidence row.

| Engine | Run | Task | Expected receipts | Observed receipts | Valid receipts | HTTP count passed |
|---|---:|---|---:|---:|---:|---|
| ai_go | 1 | open_shadow_slot | 0 | 0 | 0 | true |
| ai_go | 1 | trusted_contenteditable | 0 | 0 | 0 | true |
| ai_go | 1 | native_validation_multipart | 1 | 1 | 1 | true |
| ai_go | 1 | frame_same_origin | 1 | 1 | 1 | true |
| ai_go | 1 | frame_cross_origin | 1 | 1 | 1 | true |
| playwright_chromium | 1 | open_shadow_slot | 0 | 0 | 0 | true |
| playwright_chromium | 1 | trusted_contenteditable | 0 | 0 | 0 | true |
| playwright_chromium | 1 | native_validation_multipart | 1 | 1 | 1 | true |
| playwright_chromium | 1 | frame_same_origin | 1 | 1 | 1 | true |
| playwright_chromium | 1 | frame_cross_origin | 1 | 1 | 1 | true |
| playwright_chromium | 2 | open_shadow_slot | 0 | 0 | 0 | true |
| playwright_chromium | 2 | trusted_contenteditable | 0 | 0 | 0 | true |
| playwright_chromium | 2 | native_validation_multipart | 1 | 1 | 1 | true |
| playwright_chromium | 2 | frame_same_origin | 1 | 1 | 1 | true |
| playwright_chromium | 2 | frame_cross_origin | 1 | 1 | 1 | true |
| ai_go | 2 | open_shadow_slot | 0 | 0 | 0 | true |
| ai_go | 2 | trusted_contenteditable | 0 | 0 | 0 | true |
| ai_go | 2 | native_validation_multipart | 1 | 1 | 1 | true |
| ai_go | 2 | frame_same_origin | 1 | 1 | 1 | true |
| ai_go | 2 | frame_cross_origin | 1 | 1 | 1 | true |
| ai_go | 3 | open_shadow_slot | 0 | 0 | 0 | true |
| ai_go | 3 | trusted_contenteditable | 0 | 0 | 0 | true |
| ai_go | 3 | native_validation_multipart | 1 | 1 | 1 | true |
| ai_go | 3 | frame_same_origin | 1 | 1 | 1 | true |
| ai_go | 3 | frame_cross_origin | 1 | 1 | 1 | true |
| playwright_chromium | 3 | open_shadow_slot | 0 | 0 | 0 | true |
| playwright_chromium | 3 | trusted_contenteditable | 0 | 0 | 0 | true |
| playwright_chromium | 3 | native_validation_multipart | 1 | 1 | 1 | true |
| playwright_chromium | 3 | frame_same_origin | 1 | 1 | 1 | true |
| playwright_chromium | 3 | frame_cross_origin | 1 | 1 | 1 | true |

## Interpretation limits

- Results describe five local modern DOM tasks, not universal website compatibility or superiority. The original local-v1 corpus and its reports are separate and unchanged. No public websites, credentials, LLMs or authenticated APIs are used.
- Both engines receive the same Chromium executable, selectors, input values, ordered actions and result assertions. AI Go uses its stealth launch configuration; Playwright uses normal headless launch defaults. The flags differ between stacks. Playwright uses native locator fill/click, waits for navigation load, and polls results every 50 ms. AI Go uses query references, explicit RenderJS and a 50 ms snapshot poll. Its input/textarea/select fill uses property setters with synthetic input/change events; contenteditable uses native CDP Input.insertText and press uses native mouse input. The shadow task requires trusted clicks and the contenteditable task requires trusted input; ordinary form-field input events are not asserted as trusted.
- The shadow task covers an open shadow root, a projected slot and trusted clicks. Contenteditable requires a trusted input event and a trusted click. The multipart task first observes native invalid/zero-submit evidence, then requires exactly one valid multipart POST with the submitter value and no files. Uploading files, closed shadow roots, nested frames and arbitrary custom element frameworks are outside this corpus.
- Frame tasks discover the frame by name and interact in its selected document. The cross-origin task uses localhost on a second local server while the main document uses 127.0.0.1, making host and port distinct. The snapshot/result must be from the child document, not text aggregated into the main page.
- Visible result text contains exact event counts and values. Independently counted local HTTP submission receipts must match the expected valid request count; a duplicate or malformed submission fails the task even if the displayed result matches. Counts cover designated receipt endpoints, excluding page/frame/asset loads. HTTP evidence is additional to the visible result assertion, not evidence that every task generated a submission.
- Playwright's custom executable option may not be compatible with every Chromium build. Actual dependency and browser versions are recorded. This command installs no packages and downloads no browser binaries.
- Every repetition starts each engine in a fresh Go worker; engine order alternates. Both workers host the same two local fixture servers. The Playwright worker additionally owns Node. Complete worker wall time includes process startup, fixtures, engine setup, tasks, memory sampling, cleanup and serialization. Task time uses the Go controller wall clock for both engines, including Playwright IPC, and excludes post-task memory sampling and cleanup.
- Cold is the first task. Playwright engine setup launches Node and Chromium. AI Go engine setup constructs Browser.New; Chromium starts lazily during the first task because all five tasks explicitly request rendering. Complete worker wall time is the separate cold end-to-end comparison. Corpus totals include engine setup once per repetition.
- Playwright's compact observation traverses visible composed content, open shadow roots and projected slots once, within the selected document. Text is capped at 5000 Unicode code points, headings at 100, and links/buttons/forms at 200. AI Go uses its compact page schema, which contains different fields. UTF-8 JSON byte counts are observed sizes, not equivalent schemas or model tokens. Playwright document_bytes is serialized light-DOM HTML, excluding shadow roots; AI Go page Bytes measures its serialized composed observation, including shadow content and reference metadata. These document-byte metrics describe different representations.
- Go heap and GoSys include fixture/controller overhead; GoSys is runtime memory reservation, not RSS. Node heap is V8 heapUsed and Node RSS is process.memoryUsage().rss. Go worker, Node and Chromium tree RSS are separate. Values are maxima of post-task samples, not continuously measured peaks. Chromium working-set sums can double-count shared pages; do not add separate maxima and present them as simultaneous physical-memory consumption. Unavailable readings remain absent/null.
- Startup/transport failures keep all five planned tasks in the denominator; skipped tasks are explicitly marked not_run. HTTP evidence exists only for tasks attempted in a live worker. Very short durations can round to zero; hardware, caches, browser build and concurrent workload affect results.

## Official API references

- [API reference](https://playwright.dev/docs/api/class-browsertype#browser-type-launch)
- [API reference](https://playwright.dev/docs/api/class-page#page-goto)
- [API reference](https://playwright.dev/docs/api/class-locator#locator-fill)
- [API reference](https://playwright.dev/docs/api/class-browser#browser-new-browser-cdp-session)
- [API reference](https://nodejs.org/api/process.html#processmemoryusage)
- [API reference](https://playwright.dev/docs/locators#locate-in-shadow-dom)
- [API reference](https://playwright.dev/docs/api/class-framelocator)
