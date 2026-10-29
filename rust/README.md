# AI Go Browser — Rust rewrite

This is an incremental Rust implementation track. The Go implementation remains the
behavioral reference until feature-by-feature parity is demonstrated. The Rust
workspace currently contains **portable core contracts**, not a production browser.

## Workspace
- `crates/ai-go-browser-core`: action references, semantic snapshots, initial HTML
  actionable-element extraction, in-memory session and memory primitives, URL
  syntax gate, and browser trait.
- Workspace forbids unsafe Rust.
- Network transport, JS engine, persistent storage, MCP server, and SDK are not
  wired into this initial core.

## Development
From this directory:
```sh
cargo fmt --all
cargo test --workspace
cargo clippy --workspace --all-targets -- -D warnings
```

## Migration order
1. Freeze public behavior in Go fixtures and tests.
2. Compare semantic snapshot/action-reference output against Go fixtures.
3. Add transport with DNS/IP pinning, redirect checks, timeouts, size limits,
   and SSRF controls.
4. Improve HTML extraction for visibility, disabled controls, form labels,
   shadow DOM boundaries, and deterministic reference compatibility.
5. Add browser actions, session persistence, memory, and JS execution adapter.
6. Port MCP/API/SDK and compare end-to-end behavior against Go.
7. Benchmark representative workloads and document parity gaps.

## Security status
The URL validator is a syntactic first gate only. It cannot defend against DNS
rebinding or redirect-based SSRF. A production transport must resolve and validate
all addresses, connect to the checked address, revalidate each redirect, and
reject private/reserved destinations. Do not expose this scaffold as a usable
headless browser until those controls and tests exist.
