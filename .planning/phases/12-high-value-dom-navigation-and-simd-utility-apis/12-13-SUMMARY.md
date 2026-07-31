---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 13
subsystem: ffi
tags: [purego, abi-loader, symbol-binding, allocator-telemetry, gap-closure]

# Dependency graph
requires:
  - phase: 12-11
    provides: allocator-telemetry ABI surface (pure_simdjson_native_alloc_stats_reset/_snapshot) and the original (incomplete) optional-registration wiring this plan fixes
provides:
  - Mandatory Go-side binding of both allocator-telemetry exports for ABI 1.3
  - Per-symbol missing-binding regressions at the Bind() layer
  - Per-symbol fail-closed-before-cache regressions at the loader layer
affects: [12-VERIFICATION.md Gap 2, any future ABI 1.4+ loader work]

# Tech tracking
tech-stack:
  added: []
  patterns: ["mandatory purego symbol table entries fail Bind() before any Bindings value is returned; test fixtures must mirror the real mandatory-symbol order to be faithful regressions"]

key-files:
  created: []
  modified:
    - internal/ffi/bindings.go
    - internal/ffi/bindings_test.go
    - library_loading_test.go

key-decisions:
  - "Moved pure_simdjson_native_alloc_stats_reset/_snapshot from registerOptionalFuncWithRegistrar into the mandatory symbols slice in bindWithRegistrar, immediately after pure_simdjson_copy_implementation_name — matching the existing Bindings struct field order."
  - "Replaced the resetRegistered && snapshotRegistered guard with an unconditional b.hasNativeAllocStats = true after the mandatory loop succeeds, since a missing symbol already aborts Bind() before that point."
  - "Left the psdj_internal_materialize_build optional-registration block untouched — only the two public allocator-telemetry symbols became mandatory, per CR-02's internal-symbols-stay-optional guidance."

requirements-completed: [DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02]

# Metrics
duration: 7min
completed: 2026-07-31
---

# Phase 12 Plan 13: Allocator-Telemetry Mandatory Binding Gap Closure Summary

**Moved pure_simdjson_native_alloc_stats_reset/_snapshot out of optional purego registration into the mandatory ABI 1.3 symbol table, closing the gap where an artifact missing either export still bound and got cached.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T17:21:59+03:00 (after 12-12 completion)
- **Completed:** 2026-07-31T17:28:25+03:00
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments
- `internal/ffi/bindings.go`'s mandatory `symbols` slice now includes both allocator-telemetry exports; a missing one fails `Bind()`/`bindWithRegistrar` with a nil `*Bindings` before any lookup for later symbols even happens (fail-fast order preserved).
- Added `TestBindRequiresNativeAllocStatsSymbols` (per-symbol subtests) proving each symbol's absence individually fails binding, mirroring `TestBindRequiresEveryPhase12Symbol`'s structure.
- Added two loader-layer regressions, `TestABI13IncompleteMissingAllocStatsResetFailsClosed` and `TestABI13IncompleteMissingAllocStatsSnapshotFailsClosed`, each proving `cachedLibrary` stays nil and the `"implementation-name"` read never happens when one allocator-telemetry symbol is missing from an otherwise-ABI-1.3-complete fixture.
- Both `abi13RequiredSymbols` (internal/ffi) and `abi13MandatoryFixtureSymbols` (root package) now include both symbol names at the same index position `bindings.go` registers them, so the existing positional `reflect.DeepEqual` "complete surface" tests (`TestBindLooksUpCompleteABI13Surface`, `TestABI13CompleteBindsAndCaches`) model the true mandatory surface instead of the previously-incomplete one.

## Task Commits

Each task was committed atomically:

1. **Task 1: Make allocator-telemetry symbols mandatory in the binding table** - `910d09a` (fix)
2. **Task 2: Prove the loader fails closed before cache installation for each missing symbol** - `8865609` (test)

**Plan metadata:** (this commit) `docs(12-13): complete allocator-telemetry mandatory binding gap closure plan`

## Files Created/Modified
- `internal/ffi/bindings.go` - Two allocator-telemetry symbols moved from optional registration into the mandatory `symbols` slice in `bindWithRegistrar`; `hasNativeAllocStats` is now unconditionally `true` post-loop
- `internal/ffi/bindings_test.go` - `abi13RequiredSymbols` includes both names at the correct position; new `TestBindRequiresNativeAllocStatsSymbols`
- `library_loading_test.go` - `abi13MandatoryFixtureSymbols` includes both names at the correct position; two new fail-closed-before-cache regressions

## Decisions Made
- Kept the fix minimal and Go-only, exactly matching the plan's `files_modified` list and the gap-closure scope (no docs, no Rust/C/C++ changes needed — `docs/ffi-contract.md` already documented these as mandatory).
- Preserved `psdj_internal_materialize_build`'s optional registration untouched since it is an internal (non-public) symbol and out of scope for this fix.

## Deviations from Plan

None functionally — the code and test changes match the plan's `<action>` blocks exactly. One documentation-only note:

**Plan acceptance-criteria literal command imprecision (not a deviation from intent):**
The plan's Task 1 acceptance criteria states `grep -c registerOptionalFuncWithRegistrar internal/ffi/bindings.go` should return exactly `1`. In practice it returns `3` — one for the function definition, one for the `registerOptionalFunc` wrapper that calls it, and one for the remaining `psdj_internal_materialize_build` call site. The plan's intent ("only the `psdj_internal_materialize_build` call remains" as the sole *feature-registration* call site, down from three before this fix) is satisfied; the literal grep count in the plan text did not account for the pre-existing function definition and thin wrapper. No code change was needed — this is purely a note on an imprecise verification command in the plan, not a functional gap.

## Issues Encountered

None. Both RED-phase verifications were performed as instructed:
- Reverted `internal/ffi/bindings.go` to its pre-fix state via `git checkout -- internal/ffi/bindings.go` (a file this task itself modified, then restored via `git apply` from a saved patch) and confirmed `TestBindRequiresNativeAllocStatsSymbols` failed with "Bind() error = nil with pure_simdjson_native_alloc_stats_{reset,snapshot} missing" for both subtests.
- Reverted only the `abi13MandatoryFixtureSymbols` fixture-list hunk in `library_loading_test.go` (keeping the two new test functions) and confirmed both `TestABI13IncompleteMissingAllocStats{Reset,Snapshot}FailsClosed` failed with "activeLibrary() error = nil, want incomplete ABI 1.3 failure" — proving the fixture, not just the new test code, was necessary to catch the gap.
- Both reverts were restored before committing; final state matches the plan's intended diff exactly (confirmed via `git diff`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 12-VERIFICATION.md's Gap 2 (failed 12-11 T5) is closed: the ABI 1.3 loader now rejects an artifact missing either allocator-telemetry export before that artifact is cached process-wide.
- `TestABILaterAdditiveMinorBindsAndCaches` (ABI 1.4 forward compatibility) confirmed still green — no future additive ABI is rejected by this tightening.
- Zero Rust/C/C++ files touched; `make verify-contract` passes clean, confirming no fallout from the concurrently-landed 12-12 plan on this same branch.
- Phase 12 gap closure appears complete pending any further verifier pass.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
