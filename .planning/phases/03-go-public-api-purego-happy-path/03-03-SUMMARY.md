---
phase: 03-go-public-api-purego-happy-path
plan: "03"
subsystem: api
tags: [go, parserpool, finalizers, testing]
requires:
  - phase: 03-02
    provides: parser/doc happy path, lifecycle tests, closed and busy semantics
provides:
  - sync.Pool-backed ParserPool with deterministic rejection rules
  - Build-tag-specific parser/doc finalizer behavior
  - Semantic proof that pooled parsers are cleaned if GC drops them
affects: [phase-03-04, phase-03-05, purejson]
tech-stack:
  added: []
  patterns: [finalizer counters for leak-proof tests, helper-process warning assertions, pool-safe parser reuse]
key-files:
  created: [pool.go, finalizer_prod.go, finalizer_testbuild.go, pool_test.go]
  modified: [parser.go, doc.go, parser_test.go]
key-decisions:
  - "Kept finalizers armed while parsers sit in sync.Pool so GC eviction still cleans leaked native handles."
  - "Made build tags choose the warning behavior itself rather than attaching a production no-op finalizer body."
patterns-established:
  - "Production finalizers clean silently; purejson_testbuild finalizers log before the same cleanup path."
  - "Pool misuse is surfaced as ErrInvalidHandle, ErrClosed, or ErrParserBusy instead of being auto-repaired."
requirements-completed: [API-10, API-11]
duration: 24min
completed: 2026-04-16
---

# Phase 03: Go Public API + purego Happy Path Summary

**ParserPool reuse plus build-tagged finalizer cleanup that safely handles sync.Pool eviction and test-build leak warnings**

## Performance

- **Duration:** 24 min
- **Started:** 2026-04-16T09:20:00Z
- **Completed:** 2026-04-16T09:44:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added `ParserPool` as the explicit goroutine-per-parser reuse primitive with deterministic rejection of nil, closed, and still-busy parsers.
- Added production and `purejson_testbuild` finalizer variants that share cleanup semantics while only the test-build path emits `purejson leak:` warnings.
- Proved the tricky lifetime cases with semantic tests: cross-goroutine pool reuse, pool eviction cleanup, race-mode pool checks, and helper-process leak-warning assertions.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement `ParserPool` plus build-tag-specific cleanup finalizers** - `dee62bb` (feat)
2. **Task 2: Add tests that prove pool-eviction cleanup and warning split semantically** - `47f94b8` (test)

## Files Created/Modified
- `pool.go` - sync.Pool-backed parser reuse helper with explicit rejection rules.
- `finalizer_prod.go` - Silent cleanup finalizer attachment for production builds.
- `finalizer_testbuild.go` - Warning-emitting cleanup finalizer attachment for test builds.
- `parser.go` - Finalizer counters/hooks plus parser finalizer attachment and cleanup re-arming logic.
- `doc.go` - Document finalizer attachment and cleanup coordination with parser ownership.
- `parser_test.go` - Helper-process leak-warning tests for production and test-build variants.
- `pool_test.go` - Pool round-trip, rejection, and eviction-cleanup tests.

## Decisions Made
- Used package-local atomic counters and helper-process assertions to prove finalizer cleanup happened, instead of relying on timing guesses or fragile stderr-only checks.
- Let the parser finalizer clean a still-live doc handle before freeing the parser so a leaked parser/doc pair cannot strand native state if finalizers run in the wrong order.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The public API now has its concurrency primitive and leak-safety net, which clears the way for documentation and wrapper-smoke verification in Wave 4.
- The helper-process finalizer tests provide a stable harness for later documentation and CI smoke work around leak behavior.

---
*Phase: 03-go-public-api-purego-happy-path*
*Completed: 2026-04-16*
