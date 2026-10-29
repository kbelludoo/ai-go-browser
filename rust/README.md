# AI Go Browser — Rust rewrite

This is an incremental Rust implementation track. The Go implementation remains the
behavioral reference until feature-by-feature parity is demonstrated. The Rust
workspace currently contains **portable core contracts**, not a production browser.

## Workspace
- `crates/ai-go-browser-core`: action references and command types, semantic snapshots,
  initial HTML actionable-element extraction with explicit-label and ARIA name resolution, in-memory session and memory primitives,
  URL syntax gate with reserved-address checks, pinned HTTP GET transport with redirect/DNS checks and bounded response reads, and browser adapter trait/output.
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
3. Integrate and harden the safe HTTP transport with integration tests for DNS,
   redirect-to-private-address, timeouts, and oversized response behavior.
4. Improve HTML extraction for visibility, disabled controls, form labels,
   shadow DOM boundaries, and deterministic reference compatibility.
5. Implement concrete browser action execution, session persistence, memory,
   and a JavaScript runtime adapter.
6. Port MCP/API/SDK and compare end-to-end behavior against Go.
7. Benchmark representative workloads and document parity gaps.

## Security status
The URL validator rejects credentials, local hostnames, and several private,
reserved, and special-use IP ranges. `SafeHttpTransport` additionally resolves
DNS, rejects a hostname if any returned address is non-public, pins requests to
validated addresses, disables ambient proxies, revalidates redirects, applies a
timeout, and caps response-body size. This remains an early transport component,
not a full browser security certification: resolver behavior, platform-specific
address ranges, TLS/network edge cases, redirect and body-limit integration tests,
and browser-level controls still need independent verification. Do not expose
this scaffold as a production browser until those tests and controls exist.
