---
phase: 04-full-typed-accessor-surface
plan: "02"
subsystem: api
tags: [go, purego, simdjson, ffi, testing]
requires:
  - phase: 04-01
    provides: hidden scalar/string/bool/null bindings plus descendant-view validation
provides:
  - public ElementType enum and Element.Type()
  - public uint64/float64/string/bool/null accessors on Element
  - semantic Go coverage for scalar classification, tampered views, and numeric edges
affects: [04-03, 04-04, 04-05, DOC-03]
tech-stack:
  added: []
  patterns:
    - public accessors stay as thin Element methods over hidden ffi bindings
    - error-free inspectors collapse invalid or closed views to sentinel values
key-files:
  created: [element_scalar_test.go]
  modified: [element.go]
key-decisions:
  - "Public ElementType numerically mirrors ffi.ValueKind so Type() preserves the exact int64/uint64/float64 split."
  - "GetFloat64 rejects lossy integral conversions in the Go wrapper because native get_double rounds large int64/uint64 values silently."
  - "Integers larger than uint64 max are locked as parse-time ErrInvalidJSON cases because simdjson rejects them before GetUint64 can run."
patterns-established:
  - "Type() and IsNull() remain total: closed or invalid views map to TypeInvalid or false instead of surfacing an error."
  - "Scalar accessor tests directly mutate in-package Element.view state to lock reserved-bit and descendant-tag invalidation."
requirements-completed: [API-04, API-05, API-06]
duration: 8 min
completed: 2026-04-17
---

# Phase 04 Plan 02: Public Scalar Accessor Surface Summary

**Public `ElementType`, scalar/string/bool/null accessors, and semantic Go coverage for invalid-view and numeric-edge behavior**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-17T11:20:09+03:00
- **Completed:** 2026-04-17T11:27:21+03:00
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added the public Phase 4 scalar API on `Element`: `Type()`, `GetUint64()`, `GetFloat64()`, `GetString()`, `GetBool()`, and `IsNull()`.
- Exposed `ElementType` with the exact concrete numeric split required by Phase 4 (`Int64`, `Uint64`, `Float64`).
- Added focused Go tests covering classification, tampered-view invalidation, closed-doc semantics, copied strings, bool/null behavior, and numeric edge cases.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement the public scalar/type/string/bool/null surface on `Element`** - `e95aa2f` (feat)
2. **Task 2: Add semantic Go tests for classification, numeric edges, and string/bool/null behavior** - `e912c73` (fix)

## Files Created/Modified

- `element.go` - public `ElementType` plus scalar/string/bool/null accessors and precision-loss guard for integral-to-float64 conversion
- `element_scalar_test.go` - semantic coverage for type classification, tampered views, numeric edges, copied strings, and bool/null behavior

## Decisions Made

- Kept `ElementType` numerically aligned with `ffi.ValueKind` instead of introducing a second classifier layer.
- Enforced `ErrPrecisionLoss` in `GetFloat64()` for large integral values in the public wrapper because the native `get_double()` path does not reject those conversions itself.
- Locked oversized integer literals above `uint64` max as parse-time `ErrInvalidJSON` behavior in tests because simdjson rejects them before accessor dispatch.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added an explicit precision-loss guard for `GetFloat64()`**
- **Found during:** Task 2 (semantic Go tests)
- **Issue:** `GetFloat64()` returned a rounded float for `9007199254740993` instead of surfacing `ErrPrecisionLoss`.
- **Fix:** Added a `KindHint`-based integral fast path in `element.go` that rejects `int64`/`uint64` values outside the exact float64 integer range with `ErrPrecisionLoss`.
- **Files modified:** `element.go`, `element_scalar_test.go`
- **Verification:** `cargo build --release && go build ./... && go vet ./...` and `cargo build --release && go test ./... -run 'Test(ElementTypeClassification|TypeInvalidOnTamperedView|GetUint64|GetFloat64|GetString|GetBool|IsNull)'`
- **Committed in:** `e912c73` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** The fix was required to match the locked Phase 4 precision-loss contract. No architecture change or scope expansion.

## Issues Encountered

- The original oversized-`uint64` test input (`18446744073709551616`) fails at `Parse(...)` with `ErrInvalidJSON` because simdjson rejects integer literals beyond `uint64` max. The semantic coverage was updated to lock that parser behavior instead of expecting `GetUint64()` to receive an impossible value.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The public scalar surface is now stable and test-locked for the exact semantics Phase 4 needs before iterator work.
- Plans `04-03` and `04-04` can build on this API without reopening numeric classification or invalid-view behavior.

## Self-Check: PASSED

- Confirmed `.planning/phases/04-full-typed-accessor-surface/04-02-SUMMARY.md` exists.
- Confirmed task commits `e95aa2f` and `e912c73` exist in git history.

---
*Phase: 04-full-typed-accessor-surface*
*Completed: 2026-04-17*
