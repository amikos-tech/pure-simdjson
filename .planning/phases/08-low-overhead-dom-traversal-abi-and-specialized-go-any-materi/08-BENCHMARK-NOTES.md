# Phase 8 Benchmark Notes

## Evidence Captured

- Raw diagnostics: `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt`
- Benchstat comparison: `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt`
- Machine gate output: `testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt`

## Capture Host Identity

- `goos`: `darwin`
- `goarch`: `arm64`
- `pkg`: `github.com/amikos-tech/pure-simdjson`
- `cpu`: `Apple M3 Max`

These values match `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`.

## Phase 7 Comparison

Phase 8 kept the Phase 7 diagnostic row names and compared the new medians against the committed `v0.1.1` baseline on the same host identity.

- `twitter_json`
  - `pure-simdjson-full`: `27.96 ms` -> `0.86 ms` (`96.92%` faster)
  - `pure-simdjson-materialize-only`: `31.10 ms` -> `0.68 ms` (`97.82%` faster)
- `citm_catalog_json`
  - `pure-simdjson-full`: `75.92 ms` -> `2.35 ms` (`96.90%` faster)
  - `pure-simdjson-materialize-only`: `66.54 ms` -> `1.92 ms` (`97.11%` faster)
- `canada_json`
  - `pure-simdjson-full`: `149.00 ms` -> `6.14 ms` (`95.88%` faster)
  - `pure-simdjson-materialize-only`: `150.76 ms` -> `4.43 ms` (`97.06%` faster)

Benchstat confirms the same direction for the Tier 1 pure-simdjson full and materialize-only rows. This note intentionally stays internal and does not claim a public headline or release result.

## Correctness Gates

These commands passed before the final benchmark capture:

- `python3 tests/bench/test_check_phase8_improvement.py`
- `go test ./...`
- `cargo test -- --test-threads=1`
- `make verify-contract`
- `go test ./... -run 'TestFastMaterializer|TestJSONTestSuiteOracle' -count=1`

The final evidence capture and gate also passed:

- `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5 -timeout 1200s > testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt`
- `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt > testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt`
- `python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt > testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt`

## Phase 9 Handoff

Phase 9 owns public benchmark repositioning, README/result updates, and any release decision based on this evidence.
