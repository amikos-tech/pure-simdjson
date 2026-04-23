---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
plan: 05
subsystem: benchmarking
tags: [go, python, benchmarks, benchstat, diagnostics, materializer]

requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: Fast materializer wiring into Tier 1 diagnostics with stable Phase 7 row names
provides:
  - Hardened same-host Phase 8 Tier 1 improvement gate with subprocess-backed tests
  - Committed raw diagnostics, benchstat comparison, and machine-gated improvement proof under `testdata/benchmark-results/phase8/`
  - Internal benchmark notes that capture the evidence and defer public positioning to Phase 9
affects: [phase-08, phase-09, tier1-benchmarks, internal-notes]

tech-stack:
  added: []
  patterns:
    - Benchmark gate scripts compare medians only after metadata identity matches exactly
    - Large DOM frame streams must grow native scratch geometrically instead of reserving per container

key-files:
  created:
    - scripts/bench/check_phase8_tier1_improvement.py
    - testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt
    - testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt
    - testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt
    - .planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-BENCHMARK-NOTES.md
  modified:
    - src/native/simdjson_bridge.cpp
    - tests/bench/test_check_phase8_improvement.py

key-decisions:
  - "Phase 8 improvement gating stays same-host only: goos, goarch, pkg, and cpu must match before any PASS line is accepted."
  - "Canada benchmark regressions were caused by per-container frame reserve churn in native scratch growth, not by the fast-materializer API shape."
  - "Phase 8 evidence remains internal; README, published benchmark docs, and release decisions stay deferred to Phase 9."

patterns-established:
  - "Benchmark evidence workflow: Python gate tests -> full Go/Cargo/contract verification -> raw capture -> benchstat -> machine gate -> internal notes."
  - "Frame scratch growth: geometric reserve based on capacity avoids nested-container reallocation storms on large DOM fixtures."

requirements-completed: [D-15, D-16, D-17]

duration: 29 min
completed: 2026-04-23
---

# Phase 08 Plan 05: Benchmark Evidence Summary

**Phase 8 now has committed same-host Tier 1 diagnostic evidence, a hardened improvement gate, and internal closeout notes with public positioning deferred to Phase 9**

## Performance

- **Duration:** 29 min
- **Started:** 2026-04-23T20:57:00Z
- **Completed:** 2026-04-23T21:26:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added `scripts/bench/check_phase8_tier1_improvement.py` and subprocess-backed tests that reject metadata mismatches, missing rows, regressions, and improvements below the `10%` threshold.
- Captured committed Phase 8 raw diagnostics, benchstat output, and machine-gated PASS evidence under `testdata/benchmark-results/phase8/`.
- Wrote internal benchmark notes that record the host identity, measured deltas, passed correctness gates, and the Phase 9 handoff boundary.
- Fixed a large-fixture native scratch-growth bug that initially caused the Canada Tier 1 rows to regress by orders of magnitude during the first 08-05 gate attempt.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing gate tests** - `57d0bfa` (test)
2. **Task 1 blocker fix: geometric frame scratch growth** - `952875e` (fix)
3. **Task 1 GREEN: gate script and benchmark evidence** - `47642e8` (feat)
4. **Task 2: internal benchmark notes** - `b65ff6a` (docs)

**Plan metadata:** committed separately after summary/state/roadmap updates.

## Files Created/Modified

- `tests/bench/test_check_phase8_improvement.py` - subprocess-backed PASS/FAIL coverage for the same-host improvement gate.
- `scripts/bench/check_phase8_tier1_improvement.py` - median-based Phase 8 improvement verifier with explicit PASS/FAIL output.
- `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` - committed raw Tier 1 diagnostics captured on `darwin/arm64` `Apple M3 Max`.
- `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt` - benchstat comparison against the Phase 7 `v0.1.1` baseline.
- `testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt` - machine-gated PASS lines for all six required rows.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-BENCHMARK-NOTES.md` - internal evidence summary and Phase 9 handoff.
- `src/native/simdjson_bridge.cpp` - geometric frame-scratch reserve fix that removed the Canada regression uncovered by the first gate attempt.

## Verification

All plan-level checks passed:

```sh
python3 tests/bench/test_check_phase8_improvement.py
go test ./...
cargo test -- --test-threads=1
make verify-contract
go test ./... -run 'TestFastMaterializer|TestJSONTestSuiteOracle' -count=1
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5 -timeout 1200s > testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt
scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt > testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt
python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt
```

## Decisions Made

- The machine gate compares medians only after `goos`, `goarch`, `pkg`, and `cpu` match exactly between the old and new captures.
- The initial Canada regression came from per-container `reserve(size + 1 + child_hint)` churn in the native frame vector; geometric growth fixes the large-tree case without changing the public or internal ABI.
- Public benchmark positioning is still intentionally deferred; the new evidence is committed for Phase 9 consumption, not for immediate README or release-claim changes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed native frame scratch growth after the initial same-host gate failed on Canada**
- **Found during:** Task 1 benchmark capture and improvement gate
- **Issue:** The first Phase 8 evidence run regressed both `canada_json` rows to roughly `4.2 s/op`, with `37,569` native allocations and more than `232 GB` of cumulative native allocation traffic. The failure was structural, not noise.
- **Fix:** Replaced per-container `reserve(size + 1 + child_hint)` churn with geometric capacity growth in `src/native/simdjson_bridge.cpp`, rebuilt the release artifact, and reran the full Phase 8 evidence capture.
- **Files modified:** `src/native/simdjson_bridge.cpp`
- **Verification:** Focused Canada diagnostics dropped to `6.14 ms` full and `4.43 ms` materialize-only in the final `-count=5` capture, and the machine gate passed all six required rows.
- **Committed in:** `952875e`

---

**Total deviations:** 1 auto-fixed (1 Rule 1)
**Impact on plan:** The fix was required to make the captured Phase 8 evidence truthful and repeatable on the largest Tier 1 fixture.

## Issues Encountered

- The first 08-05 gate attempt correctly failed and left uncommitted evidence because the Canada rows regressed hard on real same-host data.
- After the native reserve fix, the rebuilt release artifact was required for the Go benchmark harness to pick up the corrected frame builder.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 8 execution is complete with committed diagnostic evidence. Phase 9 can now consume `08-BENCHMARK-NOTES.md` plus the raw/benchstat/improvement files to decide how, or whether, to update the public benchmark story and release positioning.

## Self-Check: PASSED

- Confirmed all Phase 8 benchmark evidence files and notes exist on disk.
- Confirmed commits `57d0bfa`, `952875e`, `47642e8`, and `b65ff6a` are reachable in git history.
- Verified the improvement gate now prints six PASS lines and no FAIL lines for the committed Phase 8 capture.

---
*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi*
*Completed: 2026-04-23*
