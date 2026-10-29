# AI Go Browser — Rust rewrite

This is an incremental Rust implementation track. The Go implementation remains the
behavioral reference until feature-by-feature parity is demonstrated. This is not yet
a production browser.

## Workspace
- `crates/ai-go-browser-core`: action references and command types, semantic snapshots,
  HTML actionable-element extraction with accessible-name handling, in-memory session
  and memory primitives, URL policy checks, bounded HTTP transport with DNS validation
  and redirect revalidation, and an asynchronous `HttpSnapshotBrowser` adapter.
- Workspace forbids unsafe Rust.
- `HttpSnapshotBrowser` fetches and parses static HTML, retains a navigation history,
  and supports back/forward/reload by refetching stored URLs, text extraction, and checking
  text already present in the fetched page. It intentionally rejects click, fill, select,
  and submit because no live DOM or JavaScript engine is connected yet.

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
3. Add network integration tests for DNS handling, redirects to private destinations,
   timeout behavior, and oversized responses.
4. Integrate a real browser engine and implement action execution against a live DOM.
5. Improve semantic extraction for visibility, disabled controls, forms, and deterministic
   reference compatibility.
6. Add durable session/memory storage, then port MCP/API/SDK.
7. Compare end-to-end behavior against Go fixtures and benchmark representative workloads.

## Security status
The URL validator rejects credentials, local hostnames, and several private, reserved,
and special-use IP ranges. `SafeHttpTransport` resolves DNS, rejects a hostname if any
returned address is non-public, pins requests to validated addresses, disables ambient
proxies, revalidates redirects, applies a timeout, and caps response-body size. This is
an early security-sensitive component, not a security certification. Address-range
classification and network behavior need independent review and integration testing.
Do not expose this scaffold as a production browser until those tests and browser-level
controls exist.
