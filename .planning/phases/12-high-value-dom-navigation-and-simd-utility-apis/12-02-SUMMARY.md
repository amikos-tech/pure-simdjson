---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 02
subsystem: ffi-navigation
tags: [rust, cpp, simdjson, ffi, dom, cbindgen]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-01's ABI 1.3 status mapping and resolve-then-register navigation pattern
provides:
  - Indexed array lookup through an O(n) upstream tape scan and lifetime-tracked descendant view
  - Constant-time array and object direct-child counts with valid zero-size handling
  - Exact generated-header symbol and signature checks for all three DOM-04 exports
affects: [12-08-go-indexed-container-api, 12-10-abi-smoke, 12-11-go-bindings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Validate the declared container kind before calling array- or object-specific native helpers
    - Return scalar container counts directly without descendant registration or a nonzero sentinel

key-files:
  created:
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-02-SUMMARY.md
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
  - "Preserve upstream array::at behavior: indexed lookup is a bounds-checked linear scan, and INDEX_OUT_OF_BOUNDS maps through the existing ABI status 12."
  - "Treat zero as a valid array/object count while retaining the nonzero invariant only for resolved descendant tape indices."
  - "Fail compilation if size_t is not 64-bit instead of allowing a future target to narrow the uint64_t array index."

patterns-established:
  - "Indexed descendant access: kind-check the input view, resolve one native tape index, then register the returned descendant."
  - "Container size access: kind-check the input view and return the native scalar count without allocating or registering a view."

requirements-completed: [DOM-04]

# Metrics
duration: 6min
completed: 2026-07-31
---

# Phase 12 Plan 02: Native Indexed and Container Size Helpers Summary

**Array indexing and constant-time container counts now cross the full C++/Rust ABI with bounds-safe errors, correct empty-container results, and exact generated-header contracts**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-31T10:18:37Z
- **Completed:** 2026-07-31T10:24:53Z
- **Tasks:** 2
- **Files modified:** 10 task files

## Accomplishments

- Added `pure_simdjson_array_at`, backed by upstream's bounds-checked `array::at` linear scan and the existing document-lifetime descendant registry.
- Added `pure_simdjson_array_len` and `pure_simdjson_object_size` as allocation-free scalar reads that preserve valid zero counts for empty containers.
- Enforced the supported 64-bit `uint64_t`-to-`size_t` index assumption at compile time.
- Added nine Rust integration tests plus closed-world symbol and exact-signature checks for all three exports.

## Task Commits

Each task was committed atomically:

1. **Task 1: Array.At/Len and Object.Size C++ bridge and Rust wiring** - `28ff749` (feat)
2. **Task 2: Rust integration tests and focused required-symbol/signature checks** - `cfc7319` (test)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.cpp` - Calls upstream indexed lookup and constant-time container size APIs, with a compile-time 64-bit target guard.
- `src/native/simdjson_bridge.h` - Declares the three private Rust-to-C++ bridge functions.
- `src/runtime/mod.rs` - Declares and wraps the native helpers while preserving zero as a valid scalar size.
- `src/runtime/registry.rs` - Validates array/object kinds, registers indexed descendants, and returns scalar counts directly.
- `src/lib.rs` - Exposes the three panic-contained public C ABI functions.
- `tests/rust_shim_navigation.rs` - Covers indexed success, bounds errors, wrong kinds, direct counts, and empty containers.
- `tests/abi/check_header.py` - Requires the three symbols and pins their exact public C signatures.
- `tests/abi/test_check_header.py` - Keeps the synthetic contract surface in lockstep with the generated header.
- `cbindgen.toml` - Excludes the three private bridge declarations from public header generation.
- `include/pure_simdjson.h` - Publishes the generated Array.At/Len and Object.Size declarations.

## Decisions Made

- Kept `Array.At` as a thin wrapper over upstream's O(n) `array::at`; no random-access index or duplicate bounds logic was introduced.
- Applied container-kind checks in the Rust registry before each helper so wrong-kind calls return `PURE_SIMDJSON_ERR_WRONG_TYPE` consistently.
- Kept empty counts valid by avoiding the nonzero sentinel used only for resolved descendant tape indices.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Excluded private indexed/size bridge declarations from cbindgen**
- **Found during:** Task 2 (Rust integration tests and focused required-symbol/signature checks)
- **Issue:** New private `psimdjson_*` declarations must be excluded explicitly or cbindgen exposes them in the public header and the no-internal-symbol contract fails.
- **Fix:** Added `psimdjson_array_at_index`, `psimdjson_array_size`, and `psimdjson_object_size` to the established cbindgen exclusion list before regenerating the header.
- **Files modified:** `cbindgen.toml`, `include/pure_simdjson.h`
- **Verification:** Focused header rules and the complete `make verify-contract` gate passed with no private symbols in the generated header.
- **Committed in:** `cfc7319`

---

**Total deviations:** 1 auto-fixed (1 blocking issue).
**Impact on plan:** The fix preserves the existing public/private ABI boundary without expanding feature scope.

## Issues Encountered

None beyond the cbindgen exclusion handled above.

## Verification

- `cargo build` - passed.
- `cargo test --test rust_shim_navigation` - passed 14/14 navigation tests.
- `python3 tests/abi/test_check_header.py` - passed 25/25 header-checker tests.
- `python3 tests/abi/check_header.py --rule required-symbols --rule diag-surface include/pure_simdjson.h` - passed.
- `make verify-contract` - passed `cargo check`, all 105 Rust unit/integration tests, deterministic header regeneration, every Python header rule, and C layout compilation.

## Threat and Security Impact

- **T-12-06 mitigated:** untrusted array indices use upstream's bounds-checked `array::at`; overrun maps to `PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE` without unchecked pointer arithmetic.
- **T-12-04 accepted:** indexed access remains O(n), matching upstream; the later Go API plan owns the caller-facing performance warning.
- **T-12-05 accepted:** size reads preserve upstream's 24-bit saturation behavior; the later Go API plan owns the exact `0xFFFFFF` documentation.
- No unmodeled network, authentication, schema, file-access, allocation-ownership, or trust-boundary surface was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The complete native DOM-04 surface is ready for Plan 12-08's Go methods and Plan 12-11's required purego bindings.
- Plan 12-03 can add wildcard bulk navigation without changing the indexed/container helper contracts.
- No blockers remain.

## Self-Check: PASSED

- Verified every implementation, test, checker, configuration, and generated-header file exists.
- Verified task commits `28ff749` and `cfc7319` exist in git history.
- Re-ran every task acceptance criterion and the plan-level `make verify-contract` gate successfully.
- Scanned all added lines for placeholder/TODO/FIXME stub markers; none were found.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
