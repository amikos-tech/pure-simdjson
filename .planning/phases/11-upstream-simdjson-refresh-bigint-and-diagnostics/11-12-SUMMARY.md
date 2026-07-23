---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 12
subsystem: diagnostics
tags: [error-offsets, kernel-selection, ffi, concurrency, race-detection, tdd]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-05 established native raw offset plus an independent known-location flag
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-09 froze ABI 1.2 kernel symbols and append-only status 10
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-11 established validated configured parser and pure-Go pool construction
provides:
  - Truthful Error.HasOffset state that preserves known byte zero independently from unknown
  - Process-global Kernel and SetKernel diagnostics with exact native validation
  - One Go lifecycle mutex spanning setters, parser creation, and pool creation
  - Centralized public ErrKernelLocked identity for Go guards and native status 10
affects: [11-13, 11-14, 16-v0.2-release, diagnostics, parser-lifecycle, concurrency]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Carry native location truth as an explicit boolean instead of inferring it from a numeric offset
    - Freeze Go kernel selection at pool construction while deferring native locking to the first parser miss

key-files:
  created:
    - kernel.go
    - kernel_test.go
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-12-SUMMARY.md
  modified:
    - errors.go
    - errors_test.go
    - parser.go
    - parser_test.go
    - pool.go
    - phase3_followup_test.go

key-decisions:
  - "Treat the native has-offset flag as authoritative; unknown locations normalize to Offset zero plus HasOffset false."
  - "Use one Go mutex to linearize SetKernel with configured parser construction and pure-Go pool construction."
  - "Keep Kernel cache-only and side-effect free while allowing SetKernel to resolve the native library for exact validation."

patterns-established:
  - "Truthful location transport: raw offset and known state travel together; formatting consults only the explicit known state."
  - "Two-level kernel lock: pool construction freezes the Go API immediately, while parser construction activates the native lock."

requirements-completed: [DIAG-01, DIAG-02]

# Metrics
duration: 19min
completed: 2026-07-23
---

# Phase 11 Plan 12: Truthful Error Offsets and Kernel Lifecycle Summary

**Explicit native location truth now preserves byte zero correctly, while process-global kernel selection is validated and permanently linearized with parser and pool creation**

## Performance

- **Duration:** 19 min
- **Started:** 2026-07-23T15:00:37Z
- **Completed:** 2026-07-23T15:19:58Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Added `Error.HasOffset()` and carried the native known-location flag independently from the raw offset, so known byte zero formats as `offset=0` while unknown locations omit offset text.
- Added package-level `Kernel()` and diagnostic-only `SetKernel(name)` with exact native validation, automatic-selection reset, explicit fallback support, and typed invalid/unsupported/locked errors.
- Serialized setters with configured parser construction and pure-Go pool construction; an empty pool locks the Go API without loading native code, and its first miss activates the native lock.
- Mapped native status 10 centrally to `ErrKernelLocked` and proved the real post-construction native return independently from the Go pre-lock guard.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing truthful-offset contract** - `cc31d3a` (test)
2. **Task 1 GREEN: Preserve explicit native offset truth** - `79b2151` (feat)
3. **Task 2 RED: Add failing kernel lifecycle contract** - `7d1af35` (test)
4. **Task 2 GREEN: Add process-global kernel control** - `88cf7f4` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `kernel.go` - Exposes cache-only `Kernel`, validated `SetKernel`, the Go lifecycle mutex, and irreversible selection state.
- `kernel_test.go` - Runs every mutating kernel/parser/pool scenario in a fresh subprocess, including native status 10 and race coverage.
- `errors.go` - Adds explicit known-offset state, `HasOffset`, `ErrKernelLocked`, and centralized status-10 mapping.
- `errors_test.go` - Pins nil/known-zero/known-nonzero/unknown formatting plus kernel-lock sentinel mapping.
- `parser.go` - Holds the kernel lifecycle mutex through native configured construction and locks Go selection after success.
- `pool.go` - Locks Go selection after option validation without touching the native loader.
- `parser_test.go` - Covers the characterized native offset corpus and stale-detail capacity rejection.
- `phase3_followup_test.go` - Marks legacy private formatting fixtures with their explicit known-offset state.

## Decisions Made

- `ParserLastErrorHasOffset` is authoritative. A raw zero is retained only when that flag succeeds and is true; every untrusted or unavailable location becomes zero plus false.
- `Kernel()` never calls the resolver or loader. It snapshots only `cachedLibrary`, then reads the current implementation from the already-bound native API.
- `SetKernel`, `newParserWithConfig`, and `NewParserPool` share one Go mutex. Pool construction changes only Go state; native state remains unlocked until a parser is actually constructed.
- Invalid native kernel names are translated to an error wrapping `ErrInvalidOption`; CPU-unsupported and native-lock statuses keep their centralized public sentinel identities.

## TDD Gate Compliance

- **Task 1 RED:** `cc31d3a` failed to compile because `Error` lacked both the explicit known field and public `HasOffset` method.
- **Task 1 GREEN:** `79b2151` made the known-zero, unknown, corpus, stale-detail, sentinel, and full root-package suites pass.
- **Task 2 RED:** `7d1af35` failed to compile because `Kernel`, `SetKernel`, and public `ErrKernelLocked` did not exist.
- **Task 2 GREEN:** `88cf7f4` made native-status, subprocess lifecycle, parser/pool race, fallback, and centralized sentinel tests pass.
- No refactor commits were needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated legacy private error-format fixtures for explicit known state**

- **Found during:** Task 1 GREEN
- **Issue:** Two Phase 3 formatting fixtures constructed private `Error` values with a nonzero offset but no independent known flag, so the full root suite correctly omitted their now-unproven locations.
- **Fix:** Marked only the two known-offset fixtures with `hasOffset: true`; unknown and sentinel fixtures remain false.
- **Files modified:** `phase3_followup_test.go`
- **Verification:** `go test . -count=1`
- **Committed in:** `79b2151`

**Total deviations:** 1 auto-fixed blocking issue.
**Impact on plan:** The fixture migration was required by the explicit-state contract and added no production scope.

## Issues Encountered

None.

## Verification

- `cargo test --locked --test rust_shim_diagnostics --test rust_shim_kernel -- --test-threads=1` - passed 7 diagnostic tests and the process-isolated native kernel scenario test.
- `cargo build --release --locked` - rebuilt the current ABI 1.2 native library successfully.
- `go test . -run '^TestError(HasOffset|OffsetKnownZero|OffsetUnknown|OffsetCorpus|OffsetStaleDetails)$' -count=1` - passed known-zero, unknown, corpus, and stale-detail behavior.
- `go test . -run '^TestKernelNativeLockedStatusMapsToSentinel$' -count=1` - observed raw native status 10 and preserved both `ErrKernelLocked` and `Error.Code()==10`.
- `go test . -run '^TestKernel' -race -count=1` - passed unloaded, exact-name, unsupported, fallback, parser-lock, native-lock, and setter/parser race scenarios.
- `go test . -run '^TestParserPool.*(KernelLock|LazyNativeLock)' -race -count=1` - passed pure-Go pool locking, first-miss native locking, and setter/pool race scenarios.
- `go test ./... -race -count=1` - passed all four Go packages under the race detector.
- `make verify-contract` - passed the full Rust, generated-header, ABI policy, layout, and required-symbol contract.
- `make verify-docs` - passed repository documentation checks.

## Threat and Security Impact

- **T-11-05 mitigated:** public diagnostics never infer location truth from a numeric value; known zero and unknown are independently executable.
- **T-11-02 mitigated:** exact native implementation validation rejects invalid names and maps runtime-unsupported kernels without changing the active selection.
- **T-11-06 mitigated:** one Go mutex and the existing native state-machine lock preserve a single lifecycle order across setters, parsers, and pools; raw native status 10 remains observable and typed.
- **T-11-SC preserved:** no dependency, package, network endpoint, unlock API, per-parser kernel field, or alternate native path was introduced.

## Known Stubs

None. Offset truth, kernel reads, setter validation, parser locking, and pool locking are wired to real native state.

## User Setup Required

None - diagnostics and kernel control require no external service configuration.

## Next Phase Readiness

- Plan 11-13 can rely on immutable parser/pool limits and one stable process-global native implementation during diagnostic or benchmark work.
- Plan 11-14 can run the complete public smoke matrix against truthful offsets and the finalized ABI 1.2 kernel lifecycle.
- No blockers remain for the dependent Phase 11 plans.

## Self-Check: PASSED

- All eight implementation/test files and this summary exist.
- Task commits `cc31d3a`, `79b2151`, `7d1af35`, and `88cf7f4` are present in repository history.
- The implementation/stub scan is clean, and the unrelated dirty planning paths remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
