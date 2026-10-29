# AI Go Browser — Rust rewrite

This is an incremental Rust implementation track. The Go implementation remains the
behavioral reference until feature-by-feature parity is demonstrated. The Rust
workspace currently contains **portable core contracts**, not a production browser.

## Workspace
- `crates/ai-go-browser-core`: action references, semantic snapshots, session and
  memory primitives, URL syntax gate, and browser trait.
- Workspace forbids unsafe Rust.
- Network, JS engine, browser transport, persistence, MCP server, and SDK are not
  wired into this initial core.

## Development
From this directory:
```sh
cargo fmt --all -- --check
cargo test --workspace
cargo clippy --workspace --all-targets -- -D warnings
```

## Migration order
1. Freeze public behavior in Go fixtures and tests.
2. Define semantic snapshot/action-reference compatibility.
3. Add transport with DNS/IP pinning, redirect checks, timeouts, size limits,
   and SSRF controls.
4. Implement HTML parsing and deterministic semantic extraction.
5. Add browser actions, session persistence, memory, and JS execution adapter.
6. Port MCP/API/SDK and compare end-to-end behavior against Go.
7. Benchmark representative workloads and document parity gaps.

## Security status
The URL validator is a syntactic first gate only. It cannot defend against DNS
rebinding or redirect-based SSRF. A production transport must resolve and validate
all addresses, connect to the checked address, revalidate each redirect, and
reject private/reserved destinations. Do not expose this scaffold as a usable
headless browser until those controls and tests exist.
