# 07-05 Summary

## Outcome

Plan `07-05` was rerun under the relaxed Phase 7 contract and completed successfully.

The plan captured fresh benchmark evidence in:

- `testdata/benchmark-results/v0.1.1/phase7.bench.txt`
- `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`
- `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`

It also created or updated the public-facing artifacts:

- `README.md`
- `docs/benchmarks.md`
- `docs/benchmarks/results-v0.1.1.md`
- `CHANGELOG.md`
- `LICENSE`
- `NOTICE`

## Key Findings

- Tier 1 full `any` materialization on `darwin/arm64` remains slower than `encoding/json + any` in the current DOM ABI: `0.21x` on `twitter.json`, `0.20x` on `citm_catalog.json`, and `0.17x` on `canada.json`.
- Tier 1 diagnostics show the remaining cost is dominated by materialization, not parse: `parse-only` is small relative to `pure-simdjson-full` and `materialize-only` across all three Tier 1 corpora.
- Tier 2 typed extraction is the current strength story: the published medians are `14.52x`, `11.45x`, and `10.08x` faster than `encoding/json` struct decoding on `twitter.json`, `citm_catalog.json`, and `canada.json`.
- Tier 3 selective placeholder rows are also strong on the current DOM API: `15.19x` faster than `encoding/json` struct on `twitter.json` and `20.05x` faster on `citm_catalog.json`.
- Local x86_64 parity with `minio/simdjson-go` remains unavailable because the Rosetta-backed amd64 run aborts with `Host CPU does not meet target specs`.

## Verification

- `go test ./... -run '^Example|TestJSONTestSuiteOracle$' -count=1`
- `cargo test --release -- --test-threads=1`
- README/docs/legal grep validation from the `07-05` verify block
