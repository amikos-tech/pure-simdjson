---
phase: 04-full-typed-accessor-surface
plan: "05"
subsystem: api
tags: [docs, go, examples, fuzzing, race, utf8, numeric-boundaries]
requires:
  - phase: 04-02
    provides: scalar accessors, type classification, and string/bool/null semantics
  - phase: 04-04
    provides: public array/object traversal and field lookup helpers
provides:
  - Package-wide DOC-03 closeout for the final exported purejson surface
  - Executable example matrix covering every exported purejson type
  - Numeric, malformed UTF-8, fuzz, and race verification for the Phase 4 API
affects: [phase-05, phase-07, consumers, purejson]
tech-stack:
  added: []
  patterns: [package-wide example matrix, recursive DOM fuzz validation, boundary-first numeric contract tests]
key-files:
  created: [example_test.go, element_fuzz_test.go]
  modified: [purejson.go, element.go, iterator.go, element_scalar_test.go, iterator_test.go]
key-decisions:
  - "Documented only the shipped v0.1 surface in package comments and examples instead of previewing bootstrap or On-Demand work."
  - "Locked the numeric boundary contract explicitly in tests: max-int64+1 maps to ErrNumberOutOfRange, float-kind 1e20 maps to ErrWrongType, and 9007199254740993 maps to ErrPrecisionLoss."
  - "Shipped a real FuzzParseThenGetString target that recursively walks successful DOM structures and validates copied Go strings."
patterns-established:
  - "Every exported purejson type now has at least one executable example attached through example_test.go."
  - "Malformed UTF-8 is proven at both the parse boundary and the DOM-walk boundary rather than only through scalar unit cases."
requirements-completed: [API-04, API-05, API-06, API-07, API-08, DOC-03]
duration: 11min
completed: 2026-04-17
---

# Phase 04 Plan 05: Full Typed Accessor Surface Summary

**Executable Godoc examples for every exported purejson type plus final numeric, UTF-8, fuzz, and race verification for the v0.1 DOM API**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-17T09:00:11Z
- **Completed:** 2026-04-17T09:10:59Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Re-audited the public `purejson` docs and added an executable example matrix that covers every exported type in the final Phase 4 surface.
- Closed the numeric boundary contract with explicit tests for int64 overflow, float-kind wrong-type behavior, and float64 precision loss.
- Added malformed UTF-8 rejection cases, a real `FuzzParseThenGetString` target, and completed the full `cargo` + release-build + Go race sweep.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Godoc and executable examples for the completed Phase 4 surface** - `c5681a1` (docs)
2. **Task 2: Finish the phase-close verification sweep for numeric, UTF-8, iteration, and race behavior** - `e424705` (test)

## Files Created/Modified
- `example_test.go` - Executable examples covering the full exported `purejson` surface.
- `purejson.go` - Package docs updated to describe the final typed-accessor and traversal surface.
- `element.go` - Exported accessor comments aligned with overflow, precision-loss, iterator, and field-helper behavior.
- `iterator.go` - Iterator docs tightened around document order, copied keys, and terminal error semantics.
- `element_scalar_test.go` - Numeric boundary and malformed UTF-8 scalar parse coverage.
- `iterator_test.go` - UTF-8 validation and malformed object-parse coverage alongside the existing traversal contract.
- `element_fuzz_test.go` - Recursive fuzz target for parse, traversal, and copied-string safety.

## Decisions Made
- Kept the documentation scoped to shipped behavior so Godoc does not promise Phase 5 bootstrap or later On-Demand features.
- Treated the `1e20` `GetInt64()` case as a float-kind `ErrWrongType` assertion, matching the finalized Phase 4 contract instead of coercing it into an overflow case.
- Used recursive DOM walking in the fuzz target so successful object and array paths validate copied Go strings without adding new public helpers.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A transient `.git/index.lock` blocked the first Task 1 commit attempt. The lock was already gone when inspected, and the retry succeeded without manual cleanup.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 4 now closes with package-wide DOC-03 coverage, examples for the full public API, and explicit verification of numeric, UTF-8, fuzz, and race behavior.
- Phase 5 can assume the accessor and traversal surface is both documented and semantically locked for downstream bootstrap and distribution work.

## Self-Check

PASSED

---
*Phase: 04-full-typed-accessor-surface*
*Completed: 2026-04-17*
