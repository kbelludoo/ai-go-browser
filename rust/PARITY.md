# Rust parity with the Go reference

The goal of this track is behavioral parity with the Go implementation on `main`,
followed by a measured comparison. Parity is not asserted from reading the code:
every phase must be proven by replaying Go output through the Rust code and
comparing the results.

## Method

1. **Goldens come from Go.** `rust/parity/gen` runs the Go reference over
   `rust/parity/fixtures` and writes `rust/parity/golden/*.json` with the
   agent-facing fields (title, headings, links, buttons, inputs, later: forms,
   markdown, metadata).
2. **The Rust side is compared.** `cargo run -p ai-go-browser-core --example
   parity_report` replays the same fixtures through the Rust parser and prints
   every disagreement, then exits non-zero.
3. **A phase is done only when the report is green**, the mutation checks for
   that phase fail when the feature is removed, and `cargo clippy --workspace
   --all-targets -- -D warnings` stays clean.
4. **Comparison is reported, not assumed.** The final phase repeats the Go
   benchmarks (RAM, latency, token reduction) with the same corpus and publishes
   a side-by-side table.

Golden files are committed so a Go change that alters behavior shows up as a
parity diff instead of silently changing the target.

## Baseline (measured, current branch)

`go run ./rust/parity/gen` → 6 fixtures; `cargo run --example parity_report` →
**0/6 fixtures match**. The gap classes are:

| Gap | Example from the report |
| --- | --- |
| Relative URLs are not resolved against the page URL | `"/catalog"` vs `"https://example.test/catalog"` |
| Headings are not extracted | Go reports `h1:Widget Store, h2:Featured` |
| Visibility rules are not applied | Rust lists `display:none` and `aria-hidden` links; Go excludes them |
| Forms are not modeled | forms.html: Rust 5 loose fields, Go 0 inputs + 1 form |
| Reference prefixes and naming differ | Go uses `i1` for inputs, `f1` for forms; `title`-only and `image` inputs get `(button)`/no label |

## Phases

Legend: `todo` · `partial` · `done`

### Phase 1 — Static parsing parity (`todo`)
Go: `browser/dom.go` (1083), `browser/parser.go` (514), `browser/forms.go` (293),
`browser/encoding.go` (128), `browser/promotion.go` (169).
Rust target: `src/parser.rs`, `src/semantic.rs`, new `src/dom.rs`, `src/forms.rs`.
Covers: ref numbering (`l*`/`b*`/`i*`/`f*`), accessible names in Go's precedence
order, visibility (`display:none`, `aria-hidden`, `hidden` quirks), form
ownership and submit metadata, headings, description/metadata, charset
detection, URL resolution.
Acceptance: 6/6 fixtures match; new fixtures cover tables, nested forms,
select/optgroup, contenteditable, SVG links, and duplicate ids.

### Phase 2 — HTTP transport parity (`partial`)
Go: `browser/transport.go` (464), `browser/resources.go` (124), `browser/ticketrelay.go` (534).
Rust target: `src/transport.rs`.
Covers: POST and form submission, cookies, redirect semantics, `Accept`/`Accept-Language`,
compression, charset, download/resource limits, TLS fingerprint strategy (see Phase 5).
Acceptance: recorded-response fixtures replayed through both stacks produce
identical status/headers/body for the same URL set; the existing SSRF tests stay green.

### Phase 3 — Actions and navigation parity (`partial`)
Go: `browser/actions.go` (683), `browser/browser.go` (672), `browser/navigation.go` (67),
`browser/frames.go` (299), `browser/diagnostics.go` (199).
Rust target: `src/actions.rs`, `src/browser.rs`.
Covers: `open/render/discover/inspect/run/snapshot/click/press/fill/check/query/frames/frame/submit/back/wait/release/markdown/text`,
history semantics, frame handling, diagnostics.
Acceptance: command-level golden runs (Go `Execute` vs Rust) match for every
command, including error strings.

### Phase 4 — JavaScript and hydration (`todo`)
Go: `browser/javascript.go` (407), `browser/script_engine.go` (314),
`browser/hydration.go` (211), `browser/bytecode.go` (171).
Rust target: new `src/js/`, `src/hydration.rs`.
Covers: in-process ES VM, `__NEXT_DATA__`, JSON-LD, CSR/SSR detection, stack metadata.
Acceptance: hydration/stack goldens match on an SPA corpus; JS engine runs the
Go script fixtures with identical observable output.

### Phase 5 — Live engine and stealth (`todo`)
Go: `browser/live.go` (1230), `browser/live_frames.go` (850), `browser/cdp.go` (660),
`browser/cdp_session.go` (176), `browser/live_dom.go` (198),
`browser/live_frame_coordinates.go` (203), `browser/live_native_fill.go` (105),
`browser/headless.go` (173), `browser/cso.go` (305), `browser/stealth.go` (162),
`browser/launch_args.go` (69), `browser/warp.go` (135), `browser/process_memory*.go` (97).
Rust target: new `src/live/`, feature-gated.
Covers: CDP transport, live DOM snapshots, frames, native fill, stealth profile,
uTLS-class TLS fingerprint, process memory accounting.
Acceptance: live engine produces the same snapshot/action output for the same
pages; stealth profile verified against the Go JA4 baseline; feature stays
optional so the static path has no runtime dependency on a browser.

### Phase 6 — Storage, search, MCP and service parity (`todo`)
Go: `pkg/memory` (765), `pkg/mcp` (722), `pkg/session`, `pkg/search`, `pkg/resolver`,
`pkg/deepseek` (1475), `pkg/arena` (1120), `pkg/codembvm`.
Rust target: new workspace crates (`core`, `memory`, `mcp`, `sdk`).
Covers: SQLite FTS5 + BM25 recall, durable sessions, MCP server tool surface,
resolver/search adapters, model and arena clients.
Acceptance: identical MCP tool schemas and responses, identical recall results
for the same corpus, migration test from a Go-produced database.

### Phase 7 — CLI, benchmarks and published comparison (`todo`)
Go: `cmd/*` (~50 commands), `pkg/benchmark` (2294).
Rust target: new `cli` crate + `bench/`.
Covers: `aigobrowser` CLI and interactive commands, benchmark harness parity.
Acceptance: every Go command has a Rust equivalent or a documented, approved
exception; the benchmark table (RSS, startup, latency, token reduction) is
reproduced with the same corpus and published next to the Go numbers.

## Guardrails

- `unsafe` stays forbidden workspace-wide.
- Production defaults stay fail-closed: `PublicInternet` policy and the pinned
  resolver are not relaxed for parity fixtures (fixtures use the test policy).
- Go quirks are matched deliberately, not "fixed" silently. Current examples:
  the `hidden` attribute still yields a button in Go, `input type=image` becomes
  a button, and `title`-only controls fall back to `(button)`. Each one is either
  reproduced or changed upstream in Go first, so the comparison stays truthful.
- A phase cannot be marked done from unit tests alone; the differential report
  must be green.

## Current Rust surface

`src`: `action`, `actions`, `browser`, `destination`, `memory`, `parser`,
`semantic`, `session`, `transport`, `url_policy` (~1.2k LOC). That is roughly one
tenth of the Go surface, so the matrix above is the actual backlog.
