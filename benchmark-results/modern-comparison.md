# AI Go and Playwright modern DOM comparison

Generated: 2026-10-10T17:11:20Z. Fixture corpus: local-modern-v1. Repetitions per engine: 3.

Same Chromium executable: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`. Environment: windows/amd64, go1.27.0.

Playwright: 1.62.1; Node: v24.19.0; Chromium: 155.0.4283.45.

| Engine | Assertions passed | Mean task corpus incl. engine setup (ms) | Mean complete worker wall time (ms) | Mean engine setup (ms) | Median compact JSON bytes |
|---|---:|---:|---:|---:|---:|
| ai_go | 15/15 | 2399.59 | 3212.99 | 0.19 | 652 |
| playwright_chromium | 15/15 | 2737.14 | 3208.82 | 1307.39 | 385 |

## Separate post-task memory samples

The Go worker includes the fixture server. The Playwright worker additionally owns Node. Values are maximum sampled readings, not continuous peaks; do not sum separate maxima. Chromium sums can include shared pages more than once.

| Engine | Go worker heap (MiB) | Go worker RSS (MiB) | Node V8 heap (MiB) | Node RSS (MiB) | Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|
| ai_go | 2.69 | 17.06 | unavailable | unavailable | 581.60 |
| playwright_chromium | 1.25 | 11.24 | 57.72 | 122.73 | 576.93 |

## Task results

Cold is the first rendered task after engine construction. AI Go launches Chromium lazily inside that task; Playwright launches it during engine setup. Complete worker wall time includes both startup paths.

| Engine | Run | Task | Phase | Passed | Not run | Task ms | Engine setup ms | JSON bytes |
|---|---:|---|---|---|---|---:|---:|---:|
| ai_go | 1 | open_shadow_slot | cold | true | false | 1310.07 | 0.00 | 733 |
| ai_go | 1 | trusted_contenteditable | warm | true | false | 168.56 | 0.00 | 643 |
| ai_go | 1 | native_validation_multipart | warm | true | false | 474.99 | 0.00 | 485 |
| ai_go | 1 | frame_same_origin | warm | true | false | 229.64 | 0.00 | 652 |
| ai_go | 1 | frame_cross_origin | warm | true | false | 262.02 | 0.00 | 656 |
| playwright_chromium | 1 | open_shadow_slot | cold | true | false | 450.96 | 1300.24 | 471 |
| playwright_chromium | 1 | trusted_contenteditable | warm | true | false | 126.71 | 0.00 | 385 |
| playwright_chromium | 1 | native_validation_multipart | warm | true | false | 350.32 | 0.00 | 368 |
| playwright_chromium | 1 | frame_same_origin | warm | true | false | 211.00 | 0.00 | 383 |
| playwright_chromium | 1 | frame_cross_origin | warm | true | false | 259.78 | 0.00 | 386 |
| playwright_chromium | 2 | open_shadow_slot | cold | true | false | 440.60 | 1310.34 | 471 |
| playwright_chromium | 2 | trusted_contenteditable | warm | true | false | 120.46 | 0.00 | 385 |
| playwright_chromium | 2 | native_validation_multipart | warm | true | false | 386.11 | 0.00 | 368 |
| playwright_chromium | 2 | frame_same_origin | warm | true | false | 207.64 | 0.00 | 383 |
| playwright_chromium | 2 | frame_cross_origin | warm | true | false | 252.47 | 0.00 | 386 |
| ai_go | 2 | open_shadow_slot | cold | true | false | 1244.95 | 0.00 | 733 |
| ai_go | 2 | trusted_contenteditable | warm | true | false | 161.30 | 0.00 | 643 |
| ai_go | 2 | native_validation_multipart | warm | true | false | 435.88 | 0.00 | 485 |
| ai_go | 2 | frame_same_origin | warm | true | false | 218.92 | 0.00 | 652 |
| ai_go | 2 | frame_cross_origin | warm | true | false | 250.75 | 0.00 | 656 |
| ai_go | 3 | open_shadow_slot | cold | true | false | 1280.27 | 0.57 | 733 |
| ai_go | 3 | trusted_contenteditable | warm | true | false | 184.55 | 0.00 | 643 |
| ai_go | 3 | native_validation_multipart | warm | true | false | 481.42 | 0.00 | 485 |
| ai_go | 3 | frame_same_origin | warm | true | false | 225.33 | 0.00 | 652 |
| ai_go | 3 | frame_cross_origin | warm | true | false | 269.55 | 0.00 | 656 |
| playwright_chromium | 3 | open_shadow_slot | cold | true | false | 476.70 | 1311.60 | 471 |
| playwright_chromium | 3 | trusted_contenteditable | warm | true | false | 130.51 | 0.00 | 385 |
| playwright_chromium | 3 | native_validation_multipart | warm | true | false | 373.18 | 0.00 | 368 |
| playwright_chromium | 3 | frame_same_origin | warm | true | false | 209.83 | 0.00 | 383 |
| playwright_chromium | 3 | frame_cross_origin | warm | true | false | 292.97 | 0.00 | 386 |

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
