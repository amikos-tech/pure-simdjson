---
status: testing
phase: 07-benchmarks-v0.1-release
source:
  - 07-01-SUMMARY.md
  - 07-02-SUMMARY.md
  - 07-03-SUMMARY.md
  - 07-04-SUMMARY.md
  - 07-05-SUMMARY.md
  - 07-06-SUMMARY.md
started: 2026-04-23T08:15:38Z
updated: 2026-04-23T08:17:18Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

number: 2
name: Run Phase 7 Benchmark Entry Points
expected: |
  The Phase 7 benchmark commands in the Makefile and `scripts/bench/run_benchstat.sh` should be usable from the repo root. Running the benchmark targets or helper should produce Tier 1 full-materialization and cold/warm benchmark rows with stable names for benchstat comparison.
awaiting: user response

## Tests

### 1. Run Correctness Oracle and Examples
expected: Run `go test ./... -run '^Example|TestJSONTestSuiteOracle$' -count=1`. The command should pass, including the committed JSONTestSuite oracle that validates `expectations.tsv` against the vendored case files before parsing.
result: pass
evidence: `go test ./... -run '^Example|TestJSONTestSuiteOracle$' -count=1` passed for root, `cmd/pure-simdjson-bootstrap`, `internal/bootstrap`, and `internal/ffi`.

### 2. Run Phase 7 Benchmark Entry Points
expected: The Phase 7 benchmark commands in the Makefile and `scripts/bench/run_benchstat.sh` should be usable from the repo root. Running the benchmark targets or helper should produce Tier 1 full-materialization and cold/warm benchmark rows with stable names for benchstat comparison.
result: [pending]

### 3. Verify Native Allocation Metrics
expected: Tier 1 and cold/warm benchmark output should include `native-bytes/op`, `native-allocs/op`, and `native-live-bytes` alongside Go benchmem fields, proving the reset/snapshot allocator telemetry is visible in published benchmark rows.
result: [pending]

### 4. Run Tier 2 and Tier 3 Benchmark Families
expected: Running the Tier 2 and Tier 3 benchmark families with `-benchtime=1x` should succeed. Tier 2 should compare shared-schema typed extraction across supported comparators, and Tier 3 should be clearly named as a DOM-era selective placeholder rather than a shipped On-Demand API claim.
result: [pending]

### 5. Review Public Benchmark Evidence
expected: `README.md`, `docs/benchmarks.md`, `docs/benchmarks/results-v0.1.1.md`, and `testdata/benchmark-results/v0.1.1/` should present committed evidence. They should state that Tier 1 full `any` materialization is not the current headline and that Tier 2 typed extraction plus Tier 3 selective traversal are the current strengths.
result: [pending]

### 6. Review Release and Legal Artifacts
expected: `CHANGELOG.md`, `LICENSE`, and `NOTICE` should exist and reflect the Phase 7 closeout. No new release tag should have been created during Phase 7; the existing `v0.1.0` tag remains unchanged.
result: [pending]

### 7. Review Phase Handoff
expected: `.planning/ROADMAP.md`, `.planning/STATE.md`, and `07-06-SUMMARY.md` should mark Phase 7 complete as a benchmark/docs/legal baseline, route low-overhead traversal/materialization work to Phase 8, and route benchmark recalibration plus any later release decision to Phase 9.
result: [pending]

## Summary

total: 7
passed: 1
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps

[]
