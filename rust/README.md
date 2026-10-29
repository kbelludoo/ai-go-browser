# AI Go Browser — Rust rewrite

This is an incremental Rust implementation track. The Go implementation remains the
behavioral reference until feature-by-feature parity is demonstrated. This is not yet
a production browser.

## Workspace
- `crates/ai-go-browser-core`: action references and command types, semantic snapshots,
  HTML actionable-element extraction with accessible-name handling, in-memory session
  and memory primitives, URL policy checks, bounded HTTP transport with DNS validation
  and redirect revalidation, and an asynchronous `HttpSnapshotBrowser` adapter.
- The transport exposes two seams: `DestinationPolicy` gates every hop URL and every
  resolved address (`PublicInternet` is the production default), and `HostResolver`
  performs the single DNS lookup whose answers are then pinned for the connection.
- `tests/transport_network.rs` drives the real transport against a loopback fixture
  server and asserts, per rejection, that the fixture accepted zero connections.
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
3. ~~Add network integration tests for DNS handling, redirects to private destinations,
   timeout behavior, and oversized responses.~~ Done: `tests/transport_network.rs`
   covers those cases and address pinning through the resolver seam.
4. Integrate a real browser engine and implement action execution against a live DOM.
5. Improve semantic extraction for visibility, disabled controls, forms, and deterministic
   reference compatibility.
6. Add durable session/memory storage, then port MCP/API/SDK.
7. Compare end-to-end behavior against Go fixtures and benchmark representative workloads.

## Security status
The URL validator rejects credentials, local hostnames, and several private, reserved,
and special-use IP ranges. `SafeHttpTransport` resolves DNS once per hop, rejects a
hostname if any returned address is non-public, pins the connection to the validated
answers (`resolve_to_addrs`), disables ambient proxies, revalidates every redirect target
before the next lookup, applies a timeout, and caps the streamed response body.

Loopback integration tests now demonstrate the end-to-end chain:

- a private literal URL and a hostname whose answer is private both fail before a TCP
  connection reaches the fixture server;
- a redirect from an allowed origin to `127.0.0.1` or `169.254.169.254` is rejected and
  the second fixture server records zero connections;
- DNS answers may contain private addresses; the hop fails closed if any answer is
  rejected;
- one lookup per hop reaches the resolver, and a resolver that would answer differently
  on a second call cannot change the connection (rebinding window closed by pinning);
- a silent server trips the request timeout, and bodies over the cap are rejected both by
  `Content-Length` and mid-stream, while a body exactly at the cap is accepted.

Still not proven, and out of scope for these tests: TLS interception or hostile
certificates, proxy behaviour (proxies are disabled by design), platform resolver
compromise above `HostResolver`, redirects that require cookies or authentication, and
live-DOM behaviour. This remains a security-sensitive component, not a security
certification; do not expose it as a production browser until an independent review and
browser-level controls exist.
