---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 03
subsystem: ffi-navigation
tags: [rust, cpp, simdjson, ffi, dom, wildcard, ownership]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plans 12-01/12-02's ABI 1.3 navigation status mapping and document-tied descendant registration
provides:
  - Ordered wildcard path resolution through a doc-owned native scratch vector copied synchronously into Rust-owned ValueViews
  - Exact pointer/count ownership tracking and a dedicated value-view-array free export
  - Spike 005 semantics, lifetime, free-discipline, concurrency, and exact-signature contract tests
affects: [12-06-go-navigation-api, 12-10-abi-smoke, 12-11-go-bindings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Copy transient native scratch data into owned FFI transport while the existing registry mutex remains held
    - Track struct-array allocations in a dedicated exact pointer/count ledger

key-files:
  created:
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-03-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.cpp
    - src/native/simdjson_bridge.h
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/lib.rs
    - tests/rust_shim_navigation.rs
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py
    - cbindgen.toml
    - include/pure_simdjson.h

key-decisions:
  - "Keep wildcard scratch storage document-owned only for the native call, then copy every match into a Rust-owned ValueView array before unlocking the registry."
  - "Track returned ValueView arrays in a separate pointer-to-element-count map; do not mix struct counts with byte-allocation lengths."
  - "Use the Rust registry mutex for same-document serialization; retain the native wildcard guard only as a re-entrancy backstop."

patterns-established:
  - "Wildcard transport: native ordered json_index scratch -> synchronous registered ValueView copy -> exact tracked-array free."
  - "Empty wildcard results: success with a null pointer and zero count."

requirements-completed: [DOM-03]

# Metrics
duration: 9min
completed: 2026-07-31
---

# Phase 12 Plan 03: Wildcard AtPathAll Transport Summary

**Ordered wildcard navigation now crosses the C++/Rust ABI as document-tied ValueViews with exact free discipline, stable lifetimes, and serialized same-document access**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T10:30:45Z
- **Completed:** 2026-07-31T10:39:42Z
- **Tasks:** 2
- **Files modified:** 10 task files

## Accomplishments

- Added native wildcard path resolution that keeps temporary indices in document-owned scratch storage and copies them before the call releases the registry lock.
- Added a separately tracked Rust-owned ValueView array plus an exact pointer/count free export that rejects mismatches and double frees.
- Pinned ordered, partial, heterogeneous, empty, missing, out-of-range, and multi-wildcard behavior to the Spike 005 truth table.
- Proved copied views outlive the transport array but not their document, and concurrent calls on one document complete without exposing parser-busy behavior.
- Added closed-world public-symbol and exact-signature checks for both new ABI exports.

## Task Commits

Each task was committed atomically:

1. **Task 1: Wildcard scratch-vector bridge and Rust owned-array handoff** - `7a368b6` (feat)
2. **Task 2: Rust free-discipline tests and focused required-symbol/signature checks** - `127671d` (test)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.cpp` - Holds per-document wildcard scratch indices and resolves ordered upstream wildcard matches behind a re-entrancy guard.
- `src/native/simdjson_bridge.h` - Declares the private wildcard-index bridge.
- `src/runtime/mod.rs` - Wraps the native pointer/count result while preserving valid null/zero results.
- `src/runtime/registry.rs` - Copies indices into registered document-tied views and tracks returned arrays separately from byte buffers.
- `src/lib.rs` - Exposes wildcard lookup and ValueView-array free through panic-contained C ABI functions.
- `tests/rust_shim_navigation.rs` - Covers Spike 005 behavior, invalid paths, exact frees, lifetime, and concurrency.
- `tests/abi/check_header.py` - Requires both exports and verifies their exact parameter lists.
- `tests/abi/test_check_header.py` - Keeps synthetic surface fixtures aligned with the ABI checker.
- `cbindgen.toml` - Keeps the private native bridge declaration out of the public header.
- `include/pure_simdjson.h` - Publishes the generated wildcard lookup and free declarations.

## Decisions Made

- A wildcard result array owns only its transport storage; each copied entry remains tied to the document through the existing generation-checked registry.
- A valid zero-match query returns `PURE_SIMDJSON_OK` with null/zero, while missing, out-of-range, and non-container wildcard branches simply contribute no entry.
- Same-document safety comes from the existing Rust registry mutex across native resolution and copying, not from a new lock.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Excluded the private wildcard bridge from generated public headers**
- **Found during:** Task 2 (Rust free-discipline tests and focused required-symbol/signature checks)
- **Issue:** cbindgen discovers private `psimdjson_*` extern declarations unless each is added to the established exclusion list, which would fail the no-internal-symbol contract.
- **Fix:** Added `psimdjson_element_at_path_wildcard_indices` to `cbindgen.toml` before regenerating the public header.
- **Files modified:** `cbindgen.toml`, `include/pure_simdjson.h`
- **Verification:** The focused no-internal-symbol rule and complete `make verify-contract` gate passed.
- **Committed in:** `127671d`

---

**Total deviations:** 1 auto-fixed (1 blocking issue).
**Impact on plan:** The fix preserves the established private/public ABI boundary without expanding feature scope.

## Issues Encountered

- One candidate malformed test input, `.items[*`, is accepted by the vendored parser. The test was corrected before commit to use Spike 005's exact confirmed-invalid `.a[0` fixture.

## Verification

- `cargo build` - passed.
- `cargo test --test rust_shim_navigation` - passed 19/19 tests.
- `python3 tests/abi/test_check_header.py` - passed 25/25 tests.
- Focused `required-symbols`, `diag-surface`, and `no-internal-symbols` checks - passed.
- `make verify-contract` - passed `cargo check`, all 110 Rust tests, deterministic header regeneration, every Python header rule, and C layout compilation.

## Threat and Security Impact

- **T-12-BULK mitigated:** exact pointer/count registration rejects mismatched frees, null-boundary misuse, and double frees before allocation reconstruction.
- **T-12-07 mitigated:** native scratch indices are consumed synchronously under the registry lock and never escape the call.
- **T-12-09 mitigated:** concurrent same-document tests prove registry serialization without relying on `PARSER_BUSY`.
- **T-12-08 accepted as planned:** wildcard result size remains bounded by nodes already admitted under parser capacity and depth limits.
- No unmodeled network, authentication, schema, or file-access trust boundary was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The native DOM-03 surface is ready for Plan 12-06's Go `Element.AtPathAll` method and Plan 12-11's required purego bindings.
- Exact public signatures are ready for Plan 12-10's complete ABI 1.3 smoke gate.
- Plan 12-04 can proceed independently with the SIMD utility exports.
- No blockers remain.

## Self-Check: PASSED

- Verified the summary and all 10 implementation, test, checker, configuration, and generated-header files exist.
- Verified task commits `7a368b6` and `127671d` exist in git history.
- Re-ran every task acceptance criterion and the plan-level `make verify-contract` gate successfully.
- Scanned all added lines for placeholder/TODO/FIXME stub markers; none were found.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
