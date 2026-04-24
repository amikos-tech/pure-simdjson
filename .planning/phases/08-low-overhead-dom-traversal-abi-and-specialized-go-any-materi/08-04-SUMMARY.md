---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
plan: 04
subsystem: testing
tags: [go, benchmarks, materializer, dom, performance, testing]

requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: Internal fast materializer and borrowed-frame correctness/lifetime coverage for root and subtree materialization
provides:
  - Tier 1 pure-simdjson full and materialize-only benchmark helpers delegated to the internal fast materializer
  - Stable literal diagnostic row labels for Phase 7 vs Phase 8 benchstat comparability
  - Explicit no-cache benchmark guidance that rebuilds Go trees on every materialize-only iteration
affects: [phase-08-plan-05, phase-09, tier1-benchmarks]

tech-stack:
  added: []
  patterns:
    - Tier 1 benchmark helpers call the unexported fast materializer directly instead of benchmarking public per-node accessor recursion
    - Diagnostic row names are pinned as literal constants when future plans need grep-stable evidence anchors

key-files:
  created: []
  modified:
    - benchmark_comparators_test.go
    - benchmark_diagnostics_test.go

key-decisions:
  - "Tier 1 pure-simdjson full and materialize-only benchmark paths now delegate to fastMaterializeElement without re-entering public per-node accessors."
  - "Diagnostic Tier 1 row names remain benchstat-compatible with the Phase 7 v0.1.1 evidence, and materialize-only explicitly rebuilds Go-owned trees on every iteration."

patterns-established:
  - "Benchmark wiring: keep comparator registry keys unchanged while swapping only the pure-simdjson materializer implementation."
  - "Diagnostic semantics: pin row labels and document no-cache behavior next to the benchmark loop that materializes each result."

requirements-completed: [D-01, D-05, D-12, D-14, D-15, D-16]

duration: 6 min
completed: 2026-04-23
---

# Phase 08 Plan 04: Benchmark Wiring Summary

**Tier 1 pure-simdjson benchmark paths now use the internal fast materializer while keeping Phase 7 diagnostic row names stable and forbidding cached Go tree reuse**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-23T17:47:00Z
- **Completed:** 2026-04-23T17:53:08Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Routed the pure-simdjson Tier 1 benchmark materializer helper directly to `fastMaterializeElement`, removing the recursive accessor-based benchmark path.
- Preserved stable benchmark registry and Tier 1 diagnostic row names so Phase 8 results stay comparable against `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`.
- Documented and preserved the materialize-only benchmark contract that Go maps, slices, and strings are rebuilt on every iteration while only safe native frame scratch may be reused.

## Task Commits

Each task was committed atomically:

1. **Task 1: Route pure-simdjson Tier 1 materialization through the fast path** - `de0bad8` (feat)
2. **Task 2: Preserve diagnostic row stability and no-cache benchmark semantics** - `c78ebca` (feat)

**Plan metadata:** committed separately after summary/state/roadmap updates.

## Files Created/Modified

- `benchmark_comparators_test.go` - switches `benchmarkMaterializePureElement` to direct `fastMaterializeElement` delegation and keeps the stable pure-simdjson comparator key explicit.
- `benchmark_diagnostics_test.go` - pins literal Tier 1 diagnostic labels and adds the no-cache materialize-only loop comment required for Phase 8 evidence hygiene.

## Verification

All plan-level checks passed:

```sh
go test ./... -run 'TestFastMaterializer|TestJSONTestSuiteOracle' -count=1
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_twitter_json/(pure-simdjson-full|pure-simdjson-materialize-only)$' -benchmem -benchtime=1x -count=1
git diff --name-only -- README.md docs/benchmarks.md docs/benchmarks/results-v0.1.1.md
```

Acceptance criteria were also checked directly with `rg` and file reads for the fast-materializer delegation, absence of any `benchmarkMaterializePureElementViaAccessors` helper, the stable `benchmarkComparatorPureSimdjson = "pure-simdjson"` declaration, the required Phase 8 no-cache comment, the in-loop `benchmarkMaterializePureElement(root)` call, and the stable literal Tier 1 diagnostic row labels.

## Decisions Made

- Kept the comparator registry keys unchanged and changed only the pure-simdjson benchmark materialization implementation, preserving benchstat continuity against the Phase 7 baseline.
- Made the Tier 1 diagnostic row labels explicit string constants in the diagnostics file so the stability contract is visible in source and directly grep-verifiable.
- Left all public benchmark docs and result-positioning files untouched; Phase 9 still owns any public benchmark-story change.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan `08-05` can now capture Phase 8 diagnostic output, run benchstat against the Phase 7 baseline, and produce internal benchmark notes without reopening the benchmark wiring or row-name stability work. Public README and published results remain unchanged, preserving the explicit Phase 9 boundary.

## Self-Check: PASSED

- Confirmed `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-04-SUMMARY.md` exists on disk.
- Confirmed task commits `de0bad8` and `c78ebca` are reachable in git history.
- Stub scan across `benchmark_comparators_test.go` and `benchmark_diagnostics_test.go` found no TODO/FIXME/placeholder markers that would block the plan goal.
- Threat surface scan found no new public API, header, network, or schema surface beyond the intended internal benchmark-routing change.

---
*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi*
*Completed: 2026-04-23*
