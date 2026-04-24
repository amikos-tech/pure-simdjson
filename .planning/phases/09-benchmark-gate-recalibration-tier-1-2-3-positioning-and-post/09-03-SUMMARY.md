---
phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
plan: 03
subsystem: benchmark-docs
tags: [docs, readme, changelog, benchmarks]

requires:
  - phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
    provides: Passing linux/amd64 v0.1.2 claim summary
provides:
  - Public v0.1.2 benchmark result snapshot
  - README benchmark positioning from summary.json
  - Changelog notes for committed v0.1.2 benchmark evidence
affects: [README, docs, changelog, benchmark-positioning]

key-files:
  created:
    - docs/benchmarks/results-v0.1.2.md
  modified:
    - docs/benchmarks.md
    - README.md
    - CHANGELOG.md
    - phase7_validation_contract_test.go

requirements-completed: [BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05, BENCH-07, DOC-01, DOC-06]

completed: 2026-04-24
---

# Phase 09 Plan 03: Public Benchmark Docs Summary

**Public benchmark documentation now consumes the passing linux/amd64 v0.1.2 claim summary.**

## Accomplishments

- Created `docs/benchmarks/results-v0.1.2.md` with target metadata, durable evidence links, claim gate status, Tier 1/2/3 tables, diagnostics, cold/warm lifecycle rows, comparator notes, and release-boundary language.
- Updated `docs/benchmarks.md` to point at `testdata/benchmark-results/v0.1.2/`, `results-v0.1.2.md`, `-count=10`, `-timeout 1200s`, and the linux/amd64 CI baseline.
- Updated README benchmark wording from `summary.json`: Tier 1 headline is allowed on linux/amd64, with Tier 2 and Tier 3 stdlib-relative ratios and the required platform caveat.
- Updated CHANGELOG Unreleased entries for the linux/amd64 baseline, v0.1.2 evidence, result snapshot, and README benchmark-positioning recalibration.
- Updated the stale Phase 7 release artifact contract test so it validates the current v0.1.2 benchmark docs instead of the old v0.1.1 README pointer.

## Verification

- `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > /tmp/phase9-summary-check.json` - PASS
- `go test ./...` - PASS
- `cargo test -- --test-threads=1` - PASS
- `make verify-contract` - PASS
- `python3 tests/bench/test_check_benchmark_claims.py` - PASS

## Release Boundary

No tag, release publication, bootstrap checksum alignment, or default-install validation was performed. Phase 09.1 still owns bootstrap artifact/default-install alignment before any later release tag.

## Self-Check: PASSED

Docs, README, and CHANGELOG match the committed linux/amd64 evidence and the passing claim summary.

---
*Phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post*
*Completed: 2026-04-24*
