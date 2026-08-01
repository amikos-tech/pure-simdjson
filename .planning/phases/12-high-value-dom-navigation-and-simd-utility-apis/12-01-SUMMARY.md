---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 01
subsystem: ffi-navigation
tags: [rust, cpp, simdjson, json-pointer, json-path, cbindgen]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: ABI 1.3 numeric foundation with INVALID_PATH=11 and INDEX_OUT_OF_RANGE=12 from Plan 12-09
provides:
  - RFC 6901 pointer resolution through vendored simdjson and document-bound descendant views
  - simdjson dot/index path resolution through the same validated native stack
  - Exact generated-header symbol and signature checks for both navigation exports
affects: [12-06-go-navigation, 12-10-abi-smoke, 12-11-go-bindings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Resolve navigation in C++ and register the returned tape index as a lifetime-tracked Rust descendant
    - Exclude private Rust-to-C++ bridge declarations explicitly from cbindgen output

key-files:
  created:
    - tests/rust_shim_navigation.rs
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-01-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.cpp
    - src/native/simdjson_bridge.h
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/lib.rs
    - cbindgen.toml
    - include/pure_simdjson.h
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py

key-decisions:
  - "Delegate both pointer and dot/index path grammar to vendored simdjson instead of reimplementing either parser."
  - "Reuse with_resolved_view and encode_descendant_view_locked without a container-kind pre-check so empty pointers retain upstream scalar behavior."
  - "Keep the two private psimdjson navigation declarations out of the generated public header through the existing cbindgen exclusion mechanism."

patterns-established:
  - "Single-result navigation: validate the input view, resolve one native json_index, then register and return a descendant view."
  - "Navigation ABI checks: every public export is pinned by required-symbol name and exact parameter order, constness, size type, and output indirection."

requirements-completed: [DOM-01, DOM-02]

# Metrics
duration: 7min
completed: 2026-07-31
---

# Phase 12 Plan 01: Native AtPointer and AtPath Summary

**RFC 6901 pointer and simdjson dot/index path navigation now resolve through the full C++/Rust stack with lifetime-safe descendant views and distinct path errors**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-31T10:06:11Z
- **Completed:** 2026-07-31T10:13:21Z
- **Tasks:** 2
- **Files modified:** 10 task files

## Accomplishments

- Added `AtPointer` and `AtPath` bridge functions that pass bounded byte spans directly to vendored simdjson and return resolved tape indices.
- Reused the registry's generation and descendant-membership checks before native calls, then registered successful results as normal document-bound value views.
- Mapped malformed paths and out-of-range indices to the ABI 1.3 statuses fixed by Plan 12-09.
- Added five Rust integration tests plus exact generated-header symbol and signature enforcement.

## Task Commits

Each task was committed atomically:

1. **Task 1: AtPointer/AtPath C++ bridge and Rust wiring** - `5e37ae2` (feat)
2. **Task 2: Rust integration tests and focused required-symbol/signature checks** - `f1e7429` (test)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.cpp` - Resolves pointer/path queries and maps both navigation-specific native errors.
- `src/native/simdjson_bridge.h` - Declares the two private native bridge entry points.
- `src/runtime/mod.rs` - Declares and wraps the C++ navigation functions for Rust.
- `src/runtime/registry.rs` - Validates input views and registers resolved descendants.
- `src/lib.rs` - Exposes the two panic-contained public C ABI functions.
- `tests/rust_shim_navigation.rs` - Covers nested success, malformed syntax, out-of-range indexing, dot-path success, and bare-path rejection.
- `tests/abi/check_header.py` - Requires both symbols and pins their exact signatures.
- `tests/abi/test_check_header.py` - Keeps the independent synthetic-header surface in lockstep.
- `cbindgen.toml` - Excludes the private native bridge declarations from the public generated header.
- `include/pure_simdjson.h` - Publishes the generated `pure_simdjson_element_at_pointer` and `pure_simdjson_element_at_path` declarations.

## Decisions Made

- Kept all path parsing and traversal inside vendored simdjson; the project code only validates handles, transports bounded bytes, translates errors, and tracks result lifetimes.
- Applied no object/array kind pre-check because upstream navigation dispatches by element kind and an empty JSON Pointer can validly return the receiver itself.
- Preserved the unpublished ABI-wave boundary: these symbols are branch-local implementation state until later Phase 12 contract, binding, and bootstrap plans close the ABI 1.3 surface.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Excluded private navigation bridge declarations from cbindgen**
- **Found during:** Task 2 (Rust integration tests and focused required-symbol/signature checks)
- **Issue:** The first generated-header unit run exposed `psimdjson_element_at_pointer_index` and `psimdjson_element_at_path_index` as public prototypes because the existing cbindgen exclusion list did not yet know the new private declarations.
- **Fix:** Added both names to the established `cbindgen.toml` exclusion list and regenerated the public header.
- **Files modified:** `cbindgen.toml`, `include/pure_simdjson.h`
- **Verification:** `python3 tests/abi/test_check_header.py`, focused `required-symbols`/`diag-surface` checks, and `make verify-contract` all passed with no internal symbols in the public header.
- **Committed in:** `f1e7429`

---

**Total deviations:** 1 auto-fixed (1 blocking issue).
**Impact on plan:** The fix preserves the existing public/private ABI boundary and adds no feature scope.

## Issues Encountered

None beyond the auto-fixed cbindgen exclusion described above.

## Verification

- `cargo build` - passed.
- `cargo test --test rust_shim_navigation` - passed 5/5 navigation tests.
- `python3 tests/abi/test_check_header.py` - passed 25/25 header-contract tests.
- `python3 tests/abi/check_header.py --rule required-symbols --rule diag-surface include/pure_simdjson.h` - passed.
- `make verify-contract` - passed `cargo check`, 96 Rust tests, deterministic header regeneration, all Python header rules, and C layout compilation.

## Threat and Security Impact

- **T-12-01 mitigated:** pointer/path inputs stay bounded `(ptr,len)` byte spans and use upstream typed syntax errors rather than NUL-terminated reads.
- **T-12-02 mitigated:** `with_resolved_view` validates generation, tag, range, and descendant membership before either native navigation call.
- **T-12-ABI-WAVE preserved:** no tag, publication, release artifact, dependency, or loader policy changed.
- No unmodeled network, authentication, schema, file-access, or trust-boundary surface was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The native pointer/path exports and exact ABI signatures are ready for the later Go binding and public API plans.
- Plan 12-02 can build the indexed/container helper layer on the same resolve-then-register pattern.
- No blockers remain.

## Self-Check: PASSED

- Verified every key implementation/test file exists.
- Verified task commits `5e37ae2` and `f1e7429` exist in git history.
- Re-ran every task acceptance criterion and the plan-level contract gate successfully.
- Scanned all added lines for placeholder/TODO/FIXME stub patterns; none were found.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
