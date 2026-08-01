---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 07
subsystem: public-api
tags: [go, purego, simdjson, minify, utf8, kernel-selection]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-04's native minify and UTF-8 utility gates
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-11's mandatory ABI 1.3 utility bindings
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-06's public ErrBufferTooSmall mapping
provides:
  - Public allocating Minify and buffer-reuse MinifyInto functions
  - Public standalone ValidateUTF8 with invalid-data false/nil semantics
  - Exact same-start alias, disjoint-buffer, and pre-FFI overlap safety
  - Go/native kernel-selection lock synchronization for utility calls
affects: [12-08-container-helpers, phase-13-on-demand-extraction, phase-16-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Validate caller buffers before library loading or native calls
    - Mirror native utility CPU-gate state while holding the process-wide kernel mutex

key-files:
  created:
    - minify.go
    - utf8.go
    - minify_test.go
    - utf8_test.go
  modified:
    - kernel.go

key-decisions:
  - "Allow exact same-start minify aliasing and disjoint buffers; reject either partial-overlap direction before library loading or writes."
  - "Mark Go kernel selection locked after every native utility result except ErrCPUUnsupported, including malformed-content results."
  - "Return invalid UTF-8 as false with nil error while preserving operational failures as typed errors."

patterns-established:
  - "Standalone utilities: prevalidate, hold kernelMu, resolve activeLibrary, call the binding, mirror gate state, then wrap status."
  - "Overlap checks compare ordered start-address differences without unchecked end-address addition."

requirements-completed: [UTIL-01, UTIL-02]

# Metrics
duration: 9min
completed: 2026-07-31
---

# Phase 12 Plan 07: Public SIMD Utility API Summary

**SIMD minification and UTF-8 validation now expose allocation and buffer-reuse paths with exact overlap safety and native-consistent kernel locking**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T11:55:46Z
- **Completed:** 2026-07-31T12:04:21Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added `Minify` as a copied-result convenience and `MinifyInto` for caller-owned destination reuse, including supported exact in-place aliasing.
- Rejected short destinations and both partial-overlap directions before library loading, native calls, or buffer writes.
- Added `ValidateUTF8` with `(false, nil)` for invalid bytes and typed errors for loader, ABI, CPU, and native operational failures.
- Synchronized utility calls with process-wide kernel selection so CPU rejection stays unlocked while successful gates and later content errors lock selection.
- Covered ordinary, empty, disjoint, exact-alias, malformed, invalid-UTF-8, parser-regression, and subprocess-isolated kernel scenarios.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Define standalone utility behavior** - `8247972` (test)
2. **Task 1 GREEN: Expose minify and UTF-8 utilities** - `565f603` (feat)
3. **Task 2: Cover buffer safety and kernel-state outcomes** - `8b6b420` (test)

Plan summary metadata is committed separately from the task commits.

## Files Created/Modified

- `minify.go` - Implements allocating and caller-buffer minification with exact alias/disjoint validation.
- `utf8.go` - Implements standalone SIMD UTF-8 validation with typed operational errors.
- `kernel.go` - Documents utility-triggered locking and mirrors native gate outcomes.
- `minify_test.go` - Covers minify output, immutability, overlap, capacity, malformed input, and kernel state.
- `utf8_test.go` - Covers valid/invalid/empty UTF-8, CPU gating, locking, and parser validation regression.

## Decisions Made

- Used a small local overlap helper based on ordered address subtraction; this avoids unchecked pointer-end arithmetic while admitting exact same-start aliasing.
- Mirrored the native ordering exactly: preflight and loading failures do not lock, `ErrCPUUnsupported` does not lock, and every other native result does.
- Kept minification documentation explicit that it is a whitespace scanner, not general JSON validation.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go vet .` reports the pre-existing `materializer_fastpath.go:217:37: possible misuse of unsafe.Pointer` warning and exited non-zero in this run. The unrelated file remains unchanged and the issue was already recorded in this phase's `deferred-items.md`; `go vet -unsafeptr=false .` passed for the remaining analyzers.

## Verification

- `go build .` - passed.
- `go vet -unsafeptr=false .` - passed; default vet remains limited by the unrelated deferred warning above.
- `go test . -run 'Test(Minify|ValidateUTF8|ParseRejectsInvalidUTF8)' -count=1 -v` - passed every focused utility, overlap, CPU-gate, lock-state, and parser-regression case.
- `go test . -race -run 'Test(Minify|ValidateUTF8|ParseRejectsInvalidUTF8)' -count=1` - passed.
- `go test ./... -count=1` - passed the complete Go test tree.
- TDD gate - `8247972` failed for the expected undefined public functions before `565f603` made the same contract pass.

## Threat and Security Impact

- **T-12-16 mitigated:** short and partially-overlapping destinations fail before library loading, native calls, or writes, and tests prove storage remains unchanged.
- **T-12-17 mitigated:** one mutex-guarded state helper keeps Go selection state aligned with rejected, successful, invalid-UTF-8, and malformed-minify native outcomes.
- **T-12-18 accepted and documented:** minification success is not presented as proof of valid JSON; only the detectable unterminated-string case is tested as malformed input.
- No new network, authentication, file-access, schema, dependency, or unmodeled trust boundary was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12-08 can complete the remaining public container helpers independently.
- Phase 16 can validate the utility APIs through the already-expanded ABI 1.3 cross-platform smoke path.
- No Plan 12-07 blocker remains.

## Self-Check: PASSED

- Verified all five task files and this summary exist.
- Verified task commits `8247972`, `565f603`, and `8b6b420` exist in repository history.
- Re-ran every task acceptance criterion, the focused plan verification, the race-enabled utility suite, and the complete Go test tree successfully apart from the unrelated deferred vet analyzer noted above.
- Confirmed no tracked file was deleted, no goal-blocking stub exists, and no unmodeled threat surface was introduced.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
