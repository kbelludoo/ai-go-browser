# AI Go Browser — Rust Rewrite

Incremental Rust implementation. Go on main remains the reference until parity and benchmarks are established.

## First migration slice
1. Port action/reference contracts and tests.
2. Port semantic extraction and HTML parsing with fixture parity.
3. Port URL/network policy and SSRF protections before transport integration.
4. Port session lifecycle, memory/SQLite, MCP/API surfaces, and SDK compatibility.
5. Compare Go and Rust outputs using existing fixtures and benchmarks.

## Validate
Run from this directory: cargo test --workspace; cargo clippy --workspace --all-targets -- -D warnings

Do not claim parity until the Go test corpus has been ported and run against Rust.
