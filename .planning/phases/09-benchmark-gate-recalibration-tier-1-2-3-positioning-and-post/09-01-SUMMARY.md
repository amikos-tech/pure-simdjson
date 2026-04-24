---
phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
plan: 01
subsystem: benchmarking
tags: [benchstat, github-actions, python, benchmarks, evidence]

requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: post-ABI Tier 1 diagnostic improvement context
provides:
  - Deterministic benchmark claim gate for Phase 9 public positioning
  - Same-snapshot stdlib benchstat input normalizer
  - Dispatchable linux/amd64 benchmark capture workflow
  - v0.1.2 release-scoped benchmark evidence directory
affects: [benchmark-docs, readme-positioning, release-evidence]

tech-stack:
  added: []
  patterns:
    - Python standard-library CLI gates with unittest subprocess coverage
    - Atomic benchmark snapshot staging before final evidence directory promotion
    - Least-privilege workflow_dispatch benchmark artifact transport

key-files:
  created:
    - scripts/bench/check_benchmark_claims.py
    - tests/bench/test_check_benchmark_claims.py
    - scripts/bench/prepare_stdlib_benchstat_inputs.py
    - tests/bench/test_prepare_stdlib_benchstat_inputs.py
    - scripts/bench/capture_release_snapshot.sh
    - .github/workflows/benchmark-capture.yml
    - testdata/benchmark-results/v0.1.2/.gitkeep
  modified: []

key-decisions:
  - "Claim allowances are generated from committed benchmark evidence through a deterministic JSON gate."
  - "Same-snapshot stdlib benchstat comparisons are normalized by a tested helper instead of shell rewriting."
  - "GitHub Actions benchmark artifacts remain temporary transport; committed testdata is the durable source."

patterns-established:
  - "Claim gate output uses exact top-level keys: snapshot, target, thresholds, claims, fixtures, errors."
  - "Capture script stages a complete snapshot in a sibling temporary directory before promoting it."

requirements-completed: [BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05, BENCH-07]

duration: 7m
completed: 2026-04-24
---

# Phase 09 Plan 01: Benchmark Claim Gate and Capture Scaffolding Summary

**Deterministic Phase 9 benchmark claim gating plus linux/amd64 capture scaffolding for v0.1.2 evidence.**

## Performance

- **Duration:** 7m
- **Started:** 2026-04-24T11:54:21Z
- **Completed:** 2026-04-24T12:01:36Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added `scripts/bench/check_benchmark_claims.py`, which parses raw benchmark files, benchstat outputs, metadata, and baseline evidence to emit deterministic summary JSON and fail closed on malformed or wrong-target evidence.
- Added CLI tests covering headline, conservative, missing-file, target-mismatch, malformed-row, and Tier 2/Tier 3 regression cases.
- Added `scripts/bench/prepare_stdlib_benchstat_inputs.py` to normalize same-snapshot stdlib-vs-pure rows without ad-hoc shell rewriting.
- Added `scripts/bench/capture_release_snapshot.sh` and `.github/workflows/benchmark-capture.yml` for dispatchable linux/amd64 capture and temporary artifact upload.
- Created `testdata/benchmark-results/v0.1.2/.gitkeep` as the durable release-scoped evidence directory anchor.

## Task Commits

1. **Task 1 RED: Claim gate tests** - `df6f987` (test)
2. **Task 1 GREEN: Claim gate implementation** - `36f1d1a` (feat)
3. **Task 2: Capture scaffolding and normalizer** - `01a1cb5` (feat)

## Files Created/Modified

- `scripts/bench/check_benchmark_claims.py` - Phase 9 evidence parser and claim allowance JSON generator.
- `tests/bench/test_check_benchmark_claims.py` - Synthetic CLI coverage for claim gate modes and fail-closed evidence handling.
- `scripts/bench/prepare_stdlib_benchstat_inputs.py` - Normalizes stdlib and pure-simdjson rows to shared benchmark names for same-snapshot benchstat.
- `tests/bench/test_prepare_stdlib_benchstat_inputs.py` - CLI coverage for row normalization, metadata preservation, and missing fixture rejection.
- `scripts/bench/capture_release_snapshot.sh` - Atomic v0.1.2 benchmark capture, benchstat generation, metadata capture, and summary generation entrypoint.
- `.github/workflows/benchmark-capture.yml` - Manual linux/amd64 capture workflow with read-only token permissions and 30-day artifact upload.
- `testdata/benchmark-results/v0.1.2/.gitkeep` - Release-scoped evidence directory marker.

## Decisions Made

- Used Python standard-library CLIs and subprocess tests, matching the existing benchmark gate style.
- Kept claim-gate nonzero exits for malformed, missing, wrong-target, and Tier 2/Tier 3 regression evidence, while allowing conservative publishable modes when evidence is complete but noisy.
- Kept workflow permissions to `contents: read` and `actions: read`; no Pages, tag, push, or release publication behavior was added.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first implementation mapped noisy Tier 1 benchstat evidence to the Tier 2/Tier 3 headline mode. The claim-mode selection was tightened so median Tier 1 wins with non-significant benchstat evidence use `conservative_current_strengths`.
- The normalizer test initially matched `pure-simdjson` in the module path metadata. The assertion was narrowed to benchmark comparator suffixes.

## User Setup Required

None - no external service configuration required.

## Verification

- `python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py` - PASS
- `python3 tests/bench/test_check_benchmark_claims.py` - PASS
- `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` - PASS
- Task acceptance grep checks for required CLI flags, metadata keys, benchmark families, benchstat files, workflow permissions, and forbidden publication patterns - PASS

## Known Stubs

None.

## Next Phase Readiness

Plan 09-02 can run the dispatchable capture path or import its artifact bundle, commit real linux/amd64 `v0.1.2` benchmark evidence, and generate `summary.json` with the claim gate added here.

## Self-Check: PASSED

All created files exist on disk, and task commits `df6f987`, `36f1d1a`, and `01a1cb5` are present in git history.

---
*Phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post*
*Completed: 2026-04-24*
