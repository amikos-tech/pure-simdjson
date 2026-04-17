---
phase: 04-full-typed-accessor-surface
plan: "04"
subsystem: api
tags: [go, purego, simdjson, iterators, object-lookup]
requires:
  - phase: 04-02
    provides: public scalar/type/string/bool/null accessors and error semantics used by iterator values and field helpers
  - phase: 04-03
    provides: hidden iterator transport wrappers and native object field lookup
provides:
  - public scanner-style ArrayIter and ObjectIter wrappers
  - public Array.Iter, Object.Iter, Object.GetField, and Object.GetStringField methods
  - race-checked Go semantic tests for iteration order, done behavior, missing-vs-null, and closed-doc iteration failures
affects: [04-05, API-07, API-08, DOC-03]
tech-stack:
  added: []
  patterns:
    - scanner-style iterators cache the current key/value on successful Next and surface terminal failures through Err
    - ObjectIter keys are copied into Go strings through the existing ElementGetString ownership path
    - GetStringField preserves primitive semantics by composing GetField with GetString
key-files:
  created: [iterator.go, iterator_test.go]
  modified: [element.go]
key-decisions:
  - "ObjectIter.Next decodes the key view immediately through ElementGetString so Key never exposes borrowed native memory."
  - "Object.GetStringField remains explicit Go composition over GetField plus GetString to preserve missing/null/wrong-type behavior without adding ABI."
patterns-established:
  - "Array.Iter and Object.Iter return non-nil scanner wrappers whose Err reports constructor-time failures like ErrClosed or ErrWrongType."
  - "Closed documents stop public iteration only when Next is attempted, producing scanner-style false plus ErrClosed."
requirements-completed: [API-05, API-07, API-08]
duration: 8min
completed: 2026-04-17
---

# Phase 04 Plan 04: Public Traversal Surface Summary

**Scanner-style public array/object traversal with copied object keys and direct field helpers for the Go DOM API**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-17T11:50:00+03:00
- **Completed:** 2026-04-17T11:58:29+03:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added public `ArrayIter` and `ObjectIter` wrappers that expose the locked scanner-style `Next`/`Value`/`Key`/`Err` traversal API over the hidden iterator transport.
- Exposed `Array.Iter()`, `Object.Iter()`, `Object.GetField()`, and `Object.GetStringField()` with the required copied-key and missing-vs-null behavior.
- Added race-enabled semantic coverage for array/object order, empty iterators, repeated `Next()` after completion, null-field propagation, and iterator failure after `Doc.Close()`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement public array/object traversal plus direct field helpers** - `eb48e86` (feat)
2. **Task 2: Add semantic tests for iteration order, copied keys, and scanner edge cases** - `1429b72` (test)

## Files Created/Modified

- `iterator.go` - defines the public iterator types, scanner methods, and closed-doc error normalization for traversal.
- `element.go` - exposes the public iteration entrypoints plus direct field lookup and composed string-field helper methods.
- `iterator_test.go` - locks iteration order, copied key behavior, empty/done semantics, missing-vs-null lookup, and close-after-iteration behavior.

## Decisions Made

- Decoded object keys during `ObjectIter.Next()` through the existing `ElementGetString` binding so the public API only returns copied Go strings.
- Kept `Object.GetStringField(name)` as explicit Go composition over `GetField(name)` and `GetString()` to preserve the primitive error semantics without ABI changes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `GetStringField` return wiring**
- **Found during:** Task 1 (Implement public array/object traversal plus direct field helpers)
- **Issue:** The first `GetStringField` implementation wrapped `field.GetString()` as if it returned a single value, which broke `go build`.
- **Fix:** Returned `field.GetString()` directly so the composed helper preserves the underlying `(string, error)` signature.
- **Files modified:** `element.go`
- **Verification:** `cargo build --release && go build ./... && go vet ./...`
- **Committed in:** `eb48e86` (part of task commit)

**2. [Rule 3 - Blocking] Reused the shared parse helper in iterator tests**
- **Found during:** Task 2 (Add semantic tests for iteration order, copied keys, and scanner edge cases)
- **Issue:** `iterator_test.go` initially introduced a local `mustParseDoc` helper that duplicated the existing helper in `element_scalar_test.go`, causing the test build to fail with a redeclaration error.
- **Fix:** Removed the duplicate helper and rewired the new tests to the shared `mustParseDoc(t, json) (*Parser, *Doc)` helper already used by the scalar accessor tests.
- **Files modified:** `iterator_test.go`
- **Verification:** `cargo build --release && go test ./... -race -run 'Test(ArrayIterOrder|ArrayIterEmpty|ObjectIterOrder|ObjectIterEmpty|ObjectGetFieldMissingVsNull|GetStringField|GetStringFieldNullValue|IteratorNextAfterDone|IteratorAfterDocClose)'`
- **Committed in:** `1429b72` (part of task commit)

**3. [Rule 1 - Bug] Synchronized stale human-readable state and roadmap progress**
- **Found during:** Plan finalization
- **Issue:** The GSD state/roadmap tools advanced the machine-readable counters, but the visible progress lines in `STATE.md` and the `04-04-PLAN.md` checkbox in `ROADMAP.md` still showed the plan as incomplete.
- **Fix:** Updated the human-readable progress lines in `STATE.md` and marked `04-04-PLAN.md` complete in `ROADMAP.md` so the planning docs match the recorded counters.
- **Files modified:** `.planning/STATE.md`, `.planning/ROADMAP.md`
- **Verification:** Verified `STATE.md` shows `Plan: 5 of 5`, `Shipping: Phase 04 Plan 04 verified locally`, and `Progress: [█████████░] 15/16 plans (94%)`; verified `ROADMAP.md` marks `04-04-PLAN.md` as complete.
- **Committed in:** final metadata commit

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking issue)
**Impact on plan:** All fixes were local corrections discovered by verification or finalization gates. No API, ABI, or scope changes were introduced.

## Issues Encountered

- Initial test helper naming conflicted with an existing shared helper; resolved by reusing the established helper signature instead of adding another local variant.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The public iterator and field-helper surface is now implemented and race-checked, so Phase `04-05` can focus on DOC-03, examples, malformed-UTF-8 coverage, and full phase-close verification.
- Public traversal now matches the locked scanner model and direct-field semantics, which gives the doc/example plan a stable surface to describe.

## Self-Check: PASSED

- Confirmed `.planning/phases/04-full-typed-accessor-surface/04-04-SUMMARY.md` exists.
- Confirmed task commits `eb48e86` and `1429b72` exist in git history.

---
*Phase: 04-full-typed-accessor-surface*
*Completed: 2026-04-17*
