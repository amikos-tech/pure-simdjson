# Benchmark Methodology

This project publishes benchmark results from committed `go test -bench` output under [testdata/benchmark-results/v0.1.1](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.1). The current public snapshot is [results-v0.1.1.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.1.md).

## Tier Definitions

- Tier 1: `BenchmarkTier1FullParse_*` measures full parse plus full Go `any` materialization. For `pure-simdjson`, this means parse the document, walk the DOM, and recursively build `map[string]any`, `[]any`, and scalar Go values. This is the strict full-materialization parity benchmark and the current worst-case workload for the DOM API.
- Tier 2: `BenchmarkTier2Typed_*` measures schema-shaped typed extraction using the current public API. It reflects the intended `[]byte -> Doc -> typed accessors` path much better than Tier 1.
- Tier 3: `BenchmarkTier3SelectivePlaceholder_*` measures selective reads on the current DOM API only. It is a DOM-era placeholder benchmark, not a shipped On-Demand or path-query API.

## Comparator Rules

- Comparator tables only include libraries that actually run on that exact target/toolchain combination.
- Unsupported comparators are omitted from that target table instead of being rendered as `N/A`.
- The canonical comparator set remains `encoding/json` + `any`, `encoding/json` + struct, `minio/simdjson-go`, `bytedance/sonic`, and `goccy/go-json`, subject to the omission rule above.

## Cold Start and Native Allocation Metrics

- Cold start is defined as the first Parse after NewParser inside an already loaded process. It does not include bootstrap/download time.
- Warm benchmarks reuse an already-created parser to isolate steady-state behavior.
- The native allocator metrics `native-bytes/op`, `native-allocs/op`, and `native-live-bytes` are reported alongside Go `B/op` and `allocs/op` because simdjson-native allocations do not appear in Go’s heap metrics.

## Tier 1 Diagnostics

The diagnostic family `BenchmarkTier1Diagnostics_*` splits the steady-state Tier 1 path into `pure-simdjson-full`, `parse-only`, `materialize-only`, and two input-staging models. The important conclusion from the current snapshot is that materialization dominates parse on the current DOM ABI. Parse-only is measured in hundreds of microseconds to low single-digit milliseconds, while full and materialize-only remain in the tens to hundreds of milliseconds on the large Tier 1 corpora.

These diagnostic rows are intentionally not additive accounting. `materialize-only` keeps one parsed document open across the loop so the DOM walk and string extraction path can be measured without parse/setup noise.

## Rerun Commands

Capture the main benchmark snapshot:

```sh
go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=5 > testdata/benchmark-results/v0.1.1/phase7.bench.txt
go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=5 > testdata/benchmark-results/v0.1.1/coldwarm.bench.txt
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=1 > testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt
```

Compare two benchmark captures with `benchstat`:

```sh
./scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/phase7.bench.txt --new /path/to/new-phase7.bench.txt
```

## Interpretation Notes

- Tier 1 answers: “How fast is full parse plus full generic Go tree construction?”
- Tier 2 answers: “How fast is the current typed extraction path users are expected to write?”
- Tier 3 answers: “How fast is selective traversal on the current DOM API before On-Demand exists?”

For the current Phase 7 snapshot, the honest summary is:

- Tier 1 full `any` materialization is not the current strength of the DOM ABI.
- Tier 2 and Tier 3 are the current performance strengths.
- x86_64 parity with `minio/simdjson-go` requires a real x86_64 host; Rosetta-backed local runs are not used as a parity claim.
