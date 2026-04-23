---
phase: 07-benchmarks-v0.1-release
status: passed
verified_at: 2026-04-23
verified_by: codex
requirements:
  - BENCH-01
  - BENCH-02
  - BENCH-03
  - BENCH-04
  - BENCH-05
  - BENCH-06
  - BENCH-07
  - DOC-01
  - DOC-06
  - DOC-07
---

# Phase 7 Verification

## Verdict

Status: passed

Phase 7 delivers the benchmark/docs/legal baseline promised in `.planning/ROADMAP.md`: the repo contains a three-tier benchmark harness, vendored benchmark and oracle corpora, comparator baselines with target-aware omissions, cold/warm benchmark families, native allocator metrics, committed benchmark evidence, public README/docs/changelog/license artifacts, and a closeout handoff to Phases 8 and 9.

## Evidence

Fresh verification was run on 2026-04-23 from `gsd/phase-07-benchmarks-v0.1-release`.

- `go test ./... -count=1 -timeout 180s` passed.
- `cargo test --release -- --test-threads=1` passed.
- `python3 tests/abi/check_header.py include/pure_simdjson.h` passed.
- `bash -n scripts/bench/run_benchstat.sh` passed.
- `go test ./... -run 'Test(Phase7BenchmarkFixtureContract|Phase7BenchmarkComparatorContract|Phase7ReleaseArtifactContract|JSONTestSuiteOracle)$|^Example' -count=1` passed.
- `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder|ColdStart|Warm|Tier1Diagnostics)_' -benchtime=1x -count=1` passed.

## Requirement Coverage

| Requirement | Status | Evidence |
| --- | --- | --- |
| BENCH-01 | passed | Tier 1, Tier 2, and Tier 3 benchmark families exist and benchmark smoke passed. |
| BENCH-02 | passed | `testdata/bench/` contains the five committed simdjson corpus files with provenance. |
| BENCH-03 | passed | Comparator registry covers the required baselines with target-aware omission paths. |
| BENCH-04 | passed | Cold-start and warm benchmark families exist, and `scripts/bench/run_benchstat.sh` syntax checks. |
| BENCH-05 | passed | Native allocator reset/snapshot ABI, Go helpers, and benchmark custom metrics are covered by tests and header audit. |
| BENCH-06 | passed | `TestJSONTestSuiteOracle` passed against the vendored JSONTestSuite manifest. |
| BENCH-07 | passed | `docs/benchmarks/results-v0.1.1.md` records truthful positioning and links committed raw evidence. |
| DOC-01 | passed | `README.md` contains installation, quick start, platform matrix, and benchmark snapshot. |
| DOC-06 | passed | `CHANGELOG.md` follows Keep-a-Changelog format and records Phase 7 work under `Unreleased`. |
| DOC-07 | passed | Root `LICENSE` and `NOTICE` are present and covered by the release artifact contract test. |

## Residual Notes

`07-UAT.md` still records optional manual review checkpoints for wording quality and project-level handoff judgment. Those do not block the automated phase verification because the public evidence, artifact contracts, tests, and closeout state all passed.
