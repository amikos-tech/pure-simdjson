---
phase: 07-benchmarks-v0.1-release
plan: "01"
subsystem: testing
tags: [benchmarks, testdata, json, oracle, go]
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: released runtime baseline and the final v0.1 parser surface that Phase 7 benchmarks exercise
provides:
  - vendored benchmark corpus under testdata/bench with pinned provenance and checksums
  - vendored JSONTestSuite snapshot plus committed expectations manifest
  - cached fixture loader and manifest-driven BENCH-06 oracle entrypoint for later benchmark plans
affects: [07-02, 07-04, 07-05, benchmark harness]
tech-stack:
  added: [repo-local benchmark corpora, repo-local JSONTestSuite oracle snapshot]
  patterns: [sync.Once-backed testdata caches, manifest-driven oracle verification]
key-files:
  created: [testdata/bench/README.md, testdata/jsontestsuite/README.md, testdata/jsontestsuite/expectations.tsv, benchmark_fixtures_test.go, benchmark_oracle_test.go]
  modified: [.planning/ROADMAP.md, .planning/REQUIREMENTS.md, .planning/STATE.md]
key-decisions:
  - "Benchmark fixtures must be loaded only by exact filename from testdata/bench so later plans cannot drift back to third_party or network inputs."
  - "The JSONTestSuite oracle uses expectations.tsv as the only runtime source of truth and fails on both missing and extra vendored case files."
patterns-established:
  - "Fixture caching: benchmark and oracle helpers memoize committed testdata with sync.Once so later benchmark plans reuse one loader surface."
  - "Oracle drift checks: validate manifest-to-filesystem symmetry before parsing any case."
requirements-completed: [BENCH-02, BENCH-06]
duration: 12 min
completed: 2026-04-22
---

# Phase 07 Plan 01: Corpus And Oracle Foundation Summary

**Vendored benchmark corpora plus a manifest-driven JSONTestSuite oracle and cached fixture-loader helpers for Phase 7 benchmarks**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-22T18:50:18Z
- **Completed:** 2026-04-22T19:00:44Z
- **Tasks:** 2
- **Files modified:** 328

## Accomplishments
- Vendored all five BENCH-02 benchmark fixtures into `testdata/bench/` with pinned simdjson provenance and checksum inventory.
- Vendored a 318-file JSONTestSuite snapshot plus a committed `expectations.tsv` manifest for deterministic BENCH-06 execution.
- Added the shared fixture/oracle helper surface and `TestJSONTestSuiteOracle` as the canonical committed correctness entrypoint.

## Task Commits

Each task was committed atomically:

1. **Task 1: Vendor the benchmark corpus and record deterministic provenance** - `082ccfe` (feat)
2. **Task 2: Add fixture helpers and the committed correctness oracle** - `711a17a` (feat)

## Files Created/Modified
- `testdata/bench/README.md` and `testdata/bench/*.json` - pinned benchmark fixtures and provenance table for BENCH-02.
- `testdata/jsontestsuite/README.md`, `testdata/jsontestsuite/expectations.tsv`, and `testdata/jsontestsuite/cases/` - pinned JSONTestSuite snapshot and manifest with 318 vendored parse cases.
- `benchmark_fixtures_test.go` - canonical cached loader surface for benchmark fixtures and the oracle manifest.
- `benchmark_oracle_test.go` - manifest-driven oracle test that enforces corpus floor and manifest/filesystem drift in both directions.

## Verification
- `rg 'func loadBenchmarkFixture\\(tb testing.TB, name string\\) \\[\\]byte|func loadOracleManifest\\(tb testing.TB\\)' benchmark_fixtures_test.go` — PASS
- `rg 'TestJSONTestSuiteOracle|expectations.tsv|testdata/jsontestsuite/cases|minOracleCaseCount|300' benchmark_oracle_test.go` — PASS
- `go test ./... -run TestJSONTestSuiteOracle -count=1` — PASS

## Decisions Made
- `loadBenchmarkFixture` requires exact filenames and reads only from `testdata/bench`, which locks later benchmark plans to the committed corpus.
- `TestJSONTestSuiteOracle` validates manifest/file symmetry before parsing so BENCH-06 fails on both missing manifest targets and stray vendored files.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `benchmark_fixtures_test.go` now exposes the canonical loader surface that Plan `07-02` can reuse for Tier 1 and cold/warm benchmarks.
- `TestJSONTestSuiteOracle` provides a committed BENCH-06 guardrail for every later Phase 7 plan.
- No blockers.

## Self-Check: PASSED

- Verified `.planning/phases/07-benchmarks-v0.1-release/07-01-SUMMARY.md`, `benchmark_fixtures_test.go`, `benchmark_oracle_test.go`, `testdata/bench/README.md`, and `testdata/jsontestsuite/expectations.tsv` exist on disk.
- Verified task commits `082ccfe` and `711a17a` exist in `git log --oneline --all`.
