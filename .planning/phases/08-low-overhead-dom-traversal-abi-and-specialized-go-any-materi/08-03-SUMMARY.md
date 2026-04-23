---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
plan: 03
subsystem: api
tags: [go, purego, ffi, materializer, dom, testing]

requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: Internal native frame stream plus Rust/C++ validation for root and subtree traversal
provides:
  - Go mirror of psdj_internal_frame_t with exact 64-bit layout coverage
  - Internal purego binding for psdj_internal_materialize_build
  - Unexported fastMaterializeElement over one borrowed frame span
  - Active root, subtree, lifetime, duplicate-key, oversized-literal, and busy/closed fast-materializer tests
affects: [phase-08-plan-04, phase-08-plan-05, phase-09, tier1-benchmarks]

tech-stack:
  added: []
  patterns:
    - Borrowed frame spans are consumed under doc.mu and copied into Go-owned values only at the escape boundary
    - Full preorder-frame consumption is enforced in Go so desynchronized native streams fail with ErrInternal

key-files:
  created:
    - internal/ffi/types_test.go
    - materializer_fastpath.go
  modified:
    - internal/ffi/types.go
    - internal/ffi/bindings.go
    - materializer_fastpath_test.go
    - doc.go

key-decisions:
  - "fastMaterializeElement remains unexported and returns only Go-owned any values built from one internal frame-stream handoff."
  - "Fast materializer string and object-key bytes are copied at the Go value boundary while doc.mu stays held across borrowed-frame consumption."
  - "Doc.isClosed now avoids blocking on a contended doc.mu so the fast-path busy guard returns ErrParserBusy instead of deadlocking."

patterns-established:
  - "Internal frame binding: register psdj_internal_materialize_build in internal/ffi and expose a borrowed []InternalFrame helper without copying."
  - "Fast materializer correctness: reject leftover or under-consumed frame spans with ErrInternal and preserve int64/uint64/float64 distinctions."

requirements-completed: [D-01, D-04, D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13, D-14, D-15]

duration: 9 min
completed: 2026-04-23
---

# Phase 08 Plan 03: Go Fast Materializer Summary

**Internal Go fast materializer over borrowed native frame spans with copied strings/keys, subtree support, and deterministic busy/closed semantics**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-23T17:36:00Z
- **Completed:** 2026-04-23T17:45:14Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added the Go `InternalFrame` mirror plus exact 64-bit layout tests and a borrowed-frame `InternalMaterializeBuild` binding for `psdj_internal_materialize_build`.
- Implemented unexported `fastMaterializeElement` and `buildAnyFromFrames` so root and subtree `Element` values materialize through one internal frame-stream handoff while preserving exact numeric kinds, duplicate-key last-wins semantics, and Go-owned strings.
- Activated the Phase 8 fast-materializer tests for parity, subtree materialization, string lifetime after `Doc.Close`, parse-time oversized-literal rejection, full-frame consumption, and deterministic busy/closed behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing internal frame binding tests** - `63dfe3b` (test)
2. **Task 1 GREEN: Go internal frame mirror and binding** - `f83909d` (feat)
3. **Task 2 RED: active fast-materializer behavior tests** - `a89b3cd` (test)
4. **Task 2 GREEN: fast Go any materializer implementation** - `085d6d3` (feat)

**Plan metadata:** committed separately after summary/state/roadmap updates.

## Files Created/Modified

- `internal/ffi/types.go` - adds the Go `InternalFrame` mirror for the private native frame ABI.
- `internal/ffi/types_test.go` - pins the 72-byte layout and borrowed-frame helper semantics on supported 64-bit targets.
- `internal/ffi/bindings.go` - registers `psdj_internal_materialize_build` and exposes `InternalMaterializeBuild`.
- `materializer_fastpath.go` - implements `fastMaterializeElement`, `buildAnyFromFrames`, and copied string/key helpers over borrowed frames.
- `materializer_fastpath_test.go` - activates fast-materializer parity/lifetime/subtree/busy/error tests and removes the Wave 0 skip scaffold.
- `doc.go` - makes `Doc.isClosed()` non-blocking under doc-mutex contention so the new busy guard can return `ErrParserBusy` instead of hanging.

## Verification

All plan-level checks passed:

```sh
go test ./internal/ffi -run 'TestInternalFrameLayout' -count=1
go test ./... -run 'TestFastMaterializer' -count=1
go test ./...
cargo test -- --test-threads=1
make verify-contract
```

Acceptance criteria were also checked directly with `rg` against the expected `InternalFrame` declaration, layout assertions, internal symbol registration, `InternalMaterializeBuild` helper, `fastMaterializeElement` entrypoint, `TryLock`/`KeepAlive` placement, full-frame-consumption invariant, container preallocation, zero-length string guard, review-driven test names, absence of any public `Interface()` API, and removal of the old skip scaffold.

## Decisions Made

- The fast materializer stays internal and benchmark-facing for Phase 8; no `Element.Interface()`, `Doc.Interface()`, or other public convenience API was added.
- `fastMaterializeElement` holds `doc.mu` while consuming the borrowed frame slice and defers `runtime.KeepAlive(doc)` immediately after a successful frame fetch so borrowed bytes cannot outlive the owning `Doc`.
- `buildAnyFromFrames` treats empty streams, under-consumed containers, and trailing frames as `ErrInternal` to fail loudly on native/Go frame desynchronization.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Made `Doc.isClosed()` non-blocking under doc-mutex contention**
- **Found during:** Task 2 (Implement the unexported fast materializer and activate tests)
- **Issue:** `fastMaterializeElement` correctly used `doc.mu.TryLock()`, but `element.usableDoc()` called `Doc.isClosed()` with a blocking mutex lock. Holding `doc.mu` in `TestFastMaterializerConcurrentCloseGuard` therefore deadlocked before the busy-path check could return `ErrParserBusy`.
- **Fix:** Changed `Doc.isClosed()` to use `TryLock()` and return `false` when the doc mutex is already held, allowing `fastMaterializeElement` to reach its own `TryLock` guard and surface `ErrParserBusy` deterministically.
- **Files modified:** `doc.go`
- **Verification:** `go test . -run 'TestFastMaterializer' -count=1 -v -timeout=60s`, `go test ./...`, `cargo test -- --test-threads=1`, and `make verify-contract`
- **Committed in:** `085d6d3`

---

**Total deviations:** 1 auto-fixed (1 Rule 1)
**Impact on plan:** The fix was required for the planned concurrent-close semantics and stayed local to the doc-lifetime check used by the new fast materializer.

## Issues Encountered

- Root-package tests initially loaded an older `target/release` library that did not export `psdj_internal_materialize_build`, so the first fast-materializer run failed at symbol bind time. Rebuilding the native release artifact with `cargo build --release` resolved the mismatch with no source changes.
- The first all-tests fast-materializer run appeared hung because the busy-guard path deadlocked in `Doc.isClosed()`. Fixing the non-blocking closed check resolved the hang and made the new concurrency test deterministic.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan `08-04` can wire Tier 1 benchmark materialization through `fastMaterializeElement` without reopening correctness, subtree support, or string-lifetime behavior. The internal frame binding, borrowed-frame lifetime guard, and active fast-materializer test suite are now in place for the benchmark-wiring wave.

## Self-Check: PASSED

- Confirmed `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-03-SUMMARY.md` and all key implementation files exist on disk.
- Confirmed task commits `63dfe3b`, `f83909d`, `a89b3cd`, and `085d6d3` are reachable in git history.
- Stub scan across the 08-03 files found no TODO/FIXME/placeholder markers that would block the plan goal.

---
*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi*
*Completed: 2026-04-23*
