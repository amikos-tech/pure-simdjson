# Benchmark Results v0.1.1

This snapshot records the benchmark evidence gathered for Plan `07-05` before
any public README claim was allowed to ship.

## Gate Status

BENCH-07 >=3x: FAIL
BENCH-07 within-2x-minio-x86_64: FAIL

## Targets and Raw Evidence

- `darwin/arm64` on `Apple M3 Max`
  - Go toolchain: `go1.26.2`
  - Rust toolchain: `rustc 1.89.0`
  - OS: `macOS 26.4.1 (25E253)`
  - Tier 1/2/3 evidence: `testdata/benchmark-results/v0.1.1/phase7.darwin-arm64.actual.bench.txt`
  - Cold/warm evidence: `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`
- `darwin/amd64` via Rosetta (`cpu: VirtualApple @ 2.50GHz`)
  - Go toolchain: `go1.26.2`
  - Native shim built with `cargo build --release --target x86_64-apple-darwin`
  - Tier 1/2/3 evidence: `testdata/benchmark-results/v0.1.1/phase7.darwin-amd64-rosetta.bench.txt`
- Exact plan-command artifact:
  - `testdata/benchmark-results/v0.1.1/phase7.bench.txt`
  - The plan-required regex `Benchmark(Tier1|Tier2|Tier3)_` does not match the
    shipped benchmark names (`BenchmarkTier1FullParse_*`,
    `BenchmarkTier2Typed_*`, `BenchmarkTier3SelectivePlaceholder_*`), so this
    file contains package PASS lines only and does not back the gate
    calculations.

## >=3x vs encoding/json + any

Median `ns/op` from the committed `darwin/arm64` Tier 1 evidence:

| Fixture | pure-simdjson | encoding/json + any | Speedup |
| --- | ---: | ---: | ---: |
| `twitter.json` | `20,298,350` | `3,703,585` | `0.18x` |
| `citm_catalog.json` | `56,884,496` | `9,253,814` | `0.16x` |
| `canada.json` | `145,661,482` | `18,708,701` | `0.13x` |

Files satisfying `>=3x vs encoding/json + any`: none.

Backing raw file: `testdata/benchmark-results/v0.1.1/phase7.darwin-arm64.actual.bench.txt`

## within 2x of minio/simdjson-go on x86_64

The Rosetta `darwin/amd64` run reached the Tier 1 matrix, but every
`minio-simdjson-go` row failed instead of producing a comparable `ns/op`
measurement:

- `twitter.json`: `Host CPU does not meet target specs`
- `citm_catalog.json`: `Host CPU does not meet target specs`
- `canada.json`: `Host CPU does not meet target specs`

Because the best available local `x86_64` evidence did not yield a valid
`minio/simdjson-go` baseline, the `within 2x of minio/simdjson-go on x86_64`
claim is not supported and the gate remains `FAIL`.

Backing raw file: `testdata/benchmark-results/v0.1.1/phase7.darwin-amd64-rosetta.bench.txt`

## Cold vs Warm Snapshot

The separate cold/warm capture completed successfully on `darwin/arm64` and is
available at `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`. It is not
used to decide either BENCH-07 gate line.
