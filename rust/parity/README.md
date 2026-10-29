# Parity harness

Differential testing between the Go reference implementation and the Rust port.
See `../PARITY.md` for the phase plan and the current baseline.

## Layout

- `fixtures/` — HTML documents exercised by both implementations.
- `golden/` — committed output of the Go reference for each fixture.
- `gen/` — Go generator that produces `golden/` from `fixtures/`.

## Regenerate the goldens

From the repository root (the Go module root):

```sh
go run ./rust/parity/gen
```

## Compare Rust against the goldens

```sh
cargo run -p ai-go-browser-core --example parity_report
```

The command prints one line per fixture and exits non-zero while any fixture
differs, so it can gate a manual comparison job. It is an example rather than a
test because the gap is expected to be non-zero until Phase 1 lands.

## Adding a fixture

1. Add `fixtures/<name>.html`.
2. Run the generator; `golden/<name>.json` is written.
3. Run the comparison and fix the Rust side, not the golden.

Goldens must never be edited by hand: they are the Go behavior under test.
