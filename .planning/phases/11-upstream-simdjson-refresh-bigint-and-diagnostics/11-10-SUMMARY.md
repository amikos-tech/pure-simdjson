---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 10
subsystem: go-dom
tags: [bigint, ffi, dom, materializer, fuzz, ownership]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-02 established native kind-9 classification and exact borrowed frame text
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-08 established the mandatory copied BigInt FFI accessor
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-09 froze the complete ABI 1.2 BigInt ownership contract
provides:
  - Strict public GetBigInt returning exact Go-owned decimal text
  - Exact kind-9 propagation through frame and accessor materializers
  - BigInt-aware root, descendant, iterator, lifetime, adversarial-frame, and fuzz coverage
affects: [11-12, 11-14, 16-v0.2-release, go-dom, materialization]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Public BigInt copy-out mirrors the existing copied-string ownership path
    - Internal any materializers encode BigInt as exact Go string without a wrapper type

key-files:
  created:
    - bigint_test.go
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-10-SUMMARY.md
  modified:
    - element.go
    - materializer_fastpath.go
    - materializer_fastpath_test.go
    - element_fuzz_test.go

key-decisions:
  - "GetBigInt mirrors GetString ownership: native bytes are copied and freed before the Go-owned string is returned."
  - "The unexported any materializers represent BigInt as exact Go string, intentionally indistinguishable from a JSON string."

patterns-established:
  - "Strict copied scalar: validate the document view, call the copied FFI accessor, keep the owner alive, then map native status."
  - "Kind-9 frame consumption: reuse StringPtr/StringLen and copy before the borrowed frame lifetime ends."

requirements-completed: [NUM-01, NUM-02]

# Metrics
duration: 11min
completed: 2026-07-23
---

# Phase 11 Plan 10: Exact Go BigInt API and Materialization Summary

**Strict copied `GetBigInt` and both internal materializers now preserve kind-9 decimal text exactly across document and frame lifetimes**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-23T14:27:50Z
- **Completed:** 2026-07-23T14:39:22Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added strict `Element.GetBigInt() (string, error)` using the existing copied byte/free binding and `runtime.KeepAlive`, with exact text remaining valid after `Doc.Close()` and GC.
- Propagated BigInt through root, lookup, array/object iteration, the direct accessor fallback, and the borrowed-frame fast materializer without changing `InternalFrame`.
- Replaced parse-only oversized-literal coverage with exact root/nested materialization, ownership, adversarial span, fallback parity, and accessor-capable fuzz checks.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing BigInt API contract** - `248540e` (test)
2. **Task 1 GREEN: Expose strict copied BigInt text** - `e80da08` (feat)
3. **Task 2 RED: Add failing BigInt materializer contract** - `cd697f3` (test)
4. **Task 2 GREEN: Preserve BigInt in materializers** - `31e0dcf` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `bigint_test.go` - Pins numeric value 9, exact boundaries, strict getter matrices, copied lifetime, and root/descendant/iterator traversal.
- `element.go` - Adds the strict copied public `GetBigInt` method.
- `materializer_fastpath.go` - Routes accessor kind 9 through `GetBigInt` and copies kind-9 frame spans into Go strings.
- `materializer_fastpath_test.go` - Covers exact root/nested values, direct fallback parity, close/GC ownership, unchanged frame size, and adversarial spans.
- `element_fuzz_test.go` - Makes successful BigInt traversal call `GetBigInt`, require non-empty text, and retain exact positive/negative seed assertions.

## Decisions Made

- `GetBigInt` stays parallel to `GetString`: the binding owns copy/free mechanics, while the public method owns document validation, lifetime preservation, and status mapping.
- Internal materialization deliberately uses a plain exact Go `string`; the unexported `any` tree cannot distinguish that value from a JSON string, and no public wrapper or automatic `math/big` conversion was added.
- Kind 9 reuses the existing frame string span and copier, preserving the 72-byte ABI layout and the existing frame-guard lifetime.

## TDD Gate Compliance

- **Task 1 RED:** `248540e` failed at compile time because the boundary, strictness, lifetime, and traversal contract called the absent `Element.GetBigInt`.
- **Task 1 GREEN:** `e80da08` added only the copied public accessor and made every focused API test pass.
- **Task 2 RED:** `cd697f3` failed because both frame and accessor materializers rejected kind 9 as internal/unknown.
- **Task 2 GREEN:** `31e0dcf` added the two minimal kind-9 switch arms and made materializer, fuzz, ownership, and adversarial-span coverage pass.
- No refactor commits were needed.

## Deviations from Plan

None - plan execution stayed within the specified public accessor, materializer, test, and fuzz surfaces.

## Issues Encountered

- `go list -deps .` already includes the standard-library `math/big` transitively through the repository's existing crypto/X.509 dependency chain. Plan 11-10 introduced no direct `math/big` import, no manifest change, and no new dependency edge; the package's BigInt API remains exact text only.

## Verification

- `cargo build --release --locked` - rebuilt the ABI 1.2 native library successfully.
- `go test . -run '^TestBigInt(Classification|Boundaries|Getter|WrongType|CopyLifetime|Traversal)$' -count=1` - passed.
- `go test . -run '^Test(BigIntMaterializer.*|FastMaterializer.*BigInt.*|AccessorMaterializerBigIntParity)$' -count=1` - passed.
- `go test . -run '^FuzzParseThenGetString/' -count=1` - all seed and cached corpus cases passed.
- `go test ./internal/ffi -run '^TestInternalFrameLayout$' -count=1` - passed with the unchanged 72-byte frame.
- `go test ./... -race -count=1` - all four Go packages passed.
- Direct import and manifest checks confirmed no `math/big`, `go.mod`, or `go.sum` addition.

## Threat and Security Impact

- **T-11-04 mitigated:** public BigInt text is copied through the existing native allocation/free boundary, and frame text is copied while the document and borrowed frame span remain live.
- **T-11-SC preserved:** no package install, dependency change, alternate allocator, layout growth, borrowed public view, or network/file-access surface was introduced.

## Known Stubs

None. Every new public and internal BigInt path is wired to native data and covered by exact behavior tests.

## User Setup Required

None - no external service configuration is required.

## Next Phase Readiness

- Plan 11-11 can build immutable parser capacity/depth options on the complete ABI 1.2 binding without touching BigInt propagation.
- Later public smoke and release plans can rely on strict copied BigInt access plus exact frame/fallback materialization.
- No blockers remain for the dependent Phase 11 plans.

## Self-Check: PASSED

- All five task files and this summary exist.
- Task commits `248540e`, `e80da08`, `cd697f3`, and `31e0dcf` are present in repository history.
- The implementation/stub scan is clean, and the unrelated dirty planning paths remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
