# Benchmark Results v0.1.1

This snapshot records the current Phase 7 benchmark/docs/legal baseline after the steady-state harness fixes, parser input-buffer reuse, and Tier 1 diagnostic split were added. It is intentionally a truthful evidence snapshot, not a forced release gate.

## Status

BENCH-07 truthful-positioning: PASS
Tier 1 headline on current DOM ABI: NOT SUPPORTED
Tier 2/Tier 3 headline on current DOM ABI: SUPPORTED
x86_64 minio parity on this snapshot: UNAVAILABLE

## Target and Raw Evidence

- `darwin/arm64` on `Apple M3 Max`
  - Go toolchain: `go1.26.2`
  - Rust toolchain: `rustc 1.89.0 (29483883e 2025-08-04)`
  - OS: `macOS 26.4.1 (25E253)`
  - Tier 1/2/3 evidence: `testdata/benchmark-results/v0.1.1/phase7.bench.txt`
  - Cold/warm evidence: `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`
  - Tier 1 diagnostics: `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`
- Local `darwin/amd64` Rosetta-only attempt retained for context:
  - `testdata/benchmark-results/v0.1.1/phase7.darwin-amd64-rosetta.bench.txt`
  - not used as a valid x86_64 parity claim because `minio/simdjson-go` aborts there with `Host CPU does not meet target specs`

## Tier 1: Full Parse + Full `any` Materialization

Median `ns/op` from `testdata/benchmark-results/v0.1.1/phase7.bench.txt`:

| Fixture | pure-simdjson | encoding/json + any | Relative to stdlib |
| --- | ---: | ---: | ---: |
| `twitter.json` | `18,028,216` | `3,838,594` | `0.21x` |
| `citm_catalog.json` | `47,426,735` | `9,463,443` | `0.20x` |
| `canada.json` | `109,460,912` | `18,747,883` | `0.17x` |

On the current DOM ABI, Tier 1 is still dominated by building a full generic Go tree rather than by raw parse throughput. This is why Tier 1 remains slower than `encoding/json + any` even after the harness and warm-parser fixes.

## Tier 1 Diagnostics

Single-sample diagnostic rows from `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`:

| Fixture | pure-simdjson full | parse-only | materialize-only | encoding/json + any full |
| --- | ---: | ---: | ---: | ---: |
| `twitter.json` | `27,962,032` | `289,270` | `31,103,942` | `6,150,937` |
| `citm_catalog.json` | `75,919,203` | `543,059` | `66,541,446` | `11,908,422` |
| `canada.json` | `148,995,750` | `2,059,823` | `150,756,839` | `24,259,647` |

These diagnostic cuts are not additive accounting. They exist to show shape, not to define the published medians. The shape is consistent across all three corpora: parse-only is small relative to the full Tier 1 path, while full and materialize-only stay in the same order of magnitude. That is the Phase 7 reason Tier 1 is not the current public headline.

## Tier 2: Typed Extraction

Median `ns/op` from `testdata/benchmark-results/v0.1.1/phase7.bench.txt`:

| Fixture | pure-simdjson | encoding/json + struct | bytedance/sonic | Speedup vs stdlib |
| --- | ---: | ---: | ---: | ---: |
| `twitter.json` | `299,108` | `4,343,663` | `638,380` | `14.52x` |
| `citm_catalog.json` | `983,510` | `11,258,191` | `1,462,488` | `11.45x` |
| `canada.json` | `2,882,704` | `29,063,723` | `10,475,639` | `10.08x` |

`pure-simdjson` also beat the current `goccy/go-json` medians on all three Tier 2 fixtures in this snapshot.

## Tier 3: Selective Placeholder on the Current DOM API

Median `ns/op` from `testdata/benchmark-results/v0.1.1/phase7.bench.txt`:

| Fixture | pure-simdjson | encoding/json + struct | bytedance/sonic | Speedup vs stdlib |
| --- | ---: | ---: | ---: | ---: |
| `twitter.json` | `191,788` | `2,912,582` | `439,581` | `15.19x` |
| `citm_catalog.json` | `587,661` | `11,781,268` | `1,550,381` | `20.05x` |

Tier 3 currently covers the DOM-era placeholder workloads only. It is not an On-Demand API claim, but it does show the current selective-traversal strength story clearly.

## Cold vs Warm Parser Lifecycle

Median `ns/op` and native allocation profile from `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`:

| Fixture | cold-start median | warm median | cold native allocs/op | warm native allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `twitter.json` | `298,775` | `259,950` | `8` | `3` |
| `citm_catalog.json` | `654,227` | `718,453` | `8` | `3` |
| `canada.json` | `2,390,704` | `2,647,359` | `8` | `3` |

The important split is lifecycle shape, not claiming warm is faster on every noisy run. Cold-start rows still pay the extra parser lifecycle cost and report `8 native-allocs/op`, while steady-state rows report `3 native-allocs/op`.

## x86_64 `minio/simdjson-go` Parity

This snapshot does not claim x86_64 parity with `minio/simdjson-go`.

The best available local amd64 artifact in the repo is the Rosetta-backed file `testdata/benchmark-results/v0.1.1/phase7.darwin-amd64-rosetta.bench.txt`, and every relevant `minio/simdjson-go` row there failed with:

- `twitter.json`: `Host CPU does not meet target specs`
- `citm_catalog.json`: `Host CPU does not meet target specs`
- `canada.json`: `Host CPU does not meet target specs`

That file explains why the parity claim is unavailable locally, but a real x86_64 host is still required for any future parity statement.
