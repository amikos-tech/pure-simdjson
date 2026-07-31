---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 06
subsystem: dom-navigation
tags: [go, purego, simdjson, rfc-6901, json-pointer]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-11's mandatory ABI 1.3 navigation bindings and ownership-safe wildcard transport
provides:
  - RFC 6901 Element.AtPointer with typed traversal errors
  - Documented simdjson dot/index Element.AtPath subset
  - Ordered, document-tied Element.AtPathAll wildcard results with skipped branch failures
  - Public navigation and utility capacity error sentinels
affects: [12-07-public-utilities, 12-08-container-helpers, phase-13-on-demand-extraction]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Delegate pointer and path grammar to the pinned native simdjson implementation
    - Normalize wildcard branch misses to non-nil empty Go slices

key-files:
  created:
    - element_pointer_test.go
  modified:
    - errors.go
    - element.go

key-decisions:
  - "Require a literal wildcard before AtPathAll crosses the FFI boundary; wildcard-free traversal belongs to AtPath."
  - "Preserve upstream path semantics, including bracket-quote non-awareness and trailing empty-key segments, instead of adding a second Go parser."

patterns-established:
  - "Navigation wrappers: usableDoc guard, required binding call, runtime.KeepAlive, then shared status mapping."
  - "Wildcard results: preserve document order and lifetime while treating missing, out-of-range, and non-container branches as no match."

requirements-completed: [DOM-01, DOM-02, DOM-03, UTIL-01]

# Metrics
duration: 7min
completed: 2026-07-31
---

# Phase 12 Plan 06: Public DOM Navigation Summary

**RFC 6901 pointers, simdjson dot/index paths, and ordered wildcard selection now share one typed, document-lifetime-safe Go navigation surface**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T11:44:08Z
- **Completed:** 2026-07-31T11:51:31Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Added `Element.AtPointer` with RFC 6901 escapes, empty-pointer roots, and distinct invalid-path, missing-field, wrong-type, and out-of-range errors.
- Added `Element.AtPath` with honest documentation and tests for the required leading separator and upstream bracket-quote behavior.
- Added `Element.AtPathAll` with a pre-FFI wildcard requirement, document-order results, non-nil empty matches, branch skipping, and explicit non-support for full RFC 9535 JSONPath.
- Mapped navigation statuses and caller-recoverable destination capacity failures to dedicated exported sentinels.

## Task Commits

Each task was committed atomically:

1. **Task 1: Map navigation and utility status sentinels** - `14f5d8a` (feat)
2. **Task 2 RED: Define the navigation behavior contract** - `e625b34` (test)
3. **Task 2 GREEN: Add AtPointer, AtPath, and AtPathAll** - `1abcfe0` (feat)
4. **Task 3: Expand navigation taxonomy and pitfall coverage** - `fff6b20` (test)

Plan summary metadata is committed separately from the task commits.

## Files Created/Modified

- `errors.go` - Exposes and maps invalid-path, out-of-range, and buffer-capacity sentinels.
- `element.go` - Implements the three public navigation methods and documents their exact subset, errors, and lifetimes.
- `element_pointer_test.go` - Proves RFC pointer escapes, path asymmetries, wildcard ordering/skipping, non-nil empty results, and post-close invalidation.

## Decisions Made

- Kept all pointer and path parsing in upstream simdjson; Go adds only the literal-wildcard entry check required by the public `AtPathAll` contract.
- Preserved upstream's surprising but established behavior in documentation and tests rather than translating it into a broader JSONPath dialect.
- Reused `ErrElementNotFound` and `ErrWrongType` for traversal failures while adding only the two planned navigation sentinels.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go vet .` emitted the previously recorded `materializer_fastpath.go:217:37: possible misuse of unsafe.Pointer` warning while exiting successfully. The unrelated source remains unchanged and is already listed in this phase's `deferred-items.md`.

## Verification

- `go build .` - passed.
- `go vet .` - exited 0 with the pre-existing deferred warning above.
- `go test . -run 'TestElement_(AtPointer|AtPath|AtPathAll|NavigationAfterClose)$' -count=1 -v` - passed all focused navigation and lifetime cases.
- `go test ./... -count=1` - passed the complete Go test tree.
- TDD gate - `e625b34` failed for the expected missing methods before `1abcfe0` made the same focused contract pass.

## Threat and Security Impact

- **T-12-14 mitigated:** untrusted pointer/path grammar stays in the native typed-error path; Go adds only the amended literal-wildcard presence check and branch-result normalization.
- **T-12-15 accepted and made explicit:** bracket-quoted keys retain upstream's quote characters, with the behavior documented and tested so callers do not infer full JSONPath semantics.
- No new network, authentication, file-access, schema, or unmodeled trust boundary was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12-07 can use the new `ErrBufferTooSmall` mapping for public minify utilities.
- Plan 12-08 can reuse `ErrIndexOutOfRange` for indexed array access.
- No Plan 12-06 blocker remains.

## Self-Check: PASSED

- Verified `errors.go`, `element.go`, `element_pointer_test.go`, and this summary exist.
- Verified task commits `14f5d8a`, `e625b34`, `1abcfe0`, and `fff6b20` exist in repository history.
- Re-ran every task acceptance criterion, the focused plan verification, and the complete Go test tree successfully.
- Confirmed no tracked file was deleted, no goal-blocking stub exists, and no unmodeled threat surface was introduced.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
