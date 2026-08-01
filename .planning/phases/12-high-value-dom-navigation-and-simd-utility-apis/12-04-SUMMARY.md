---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 04
subsystem: ffi-utilities
tags: [rust, cpp, simdjson, ffi, minify, utf8, kernel-selection]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plans 12-01 through 12-03's ABI 1.3 foundation, generated-header checks, and shared native implementation-selection state
provides:
  - Capacity-checked minification with exact same-start alias support and conservative partial-overlap rejection
  - Standalone UTF-8 validation through the active simdjson implementation
  - Utility CPU rejection and post-success kernel-selection locking without holding the selection mutex during scans
affects: [12-07-go-utility-api, 12-10-abi-smoke-and-x86-gate, 12-11-go-bindings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Enforce caller-buffer capacity and overlap policy before kernel selection or native writes
    - Scope the process-global selection mutex to CPU validation and lock transition, then scan after releasing it

key-files:
  created:
    - tests/rust_shim_minify.rs
    - tests/rust_shim_utility_lock.rs
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-04-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/lib.rs
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py
    - cbindgen.toml
    - include/pure_simdjson.h

key-decisions:
  - "Treat exact same-start aliasing and fully disjoint buffers as the only valid minify layouts; reject either partial-overlap direction before writing."
  - "Reject implicit fallback before locking utility kernel selection, then release the selection mutex before running the SIMD scan."
  - "Describe the fresh-process test narrowly: it proves Rust pre-FFI rejection leaves native state untouched and a successful native utility call locks selection; it does not force the C++ implicit-fallback branch."

patterns-established:
  - "Standalone SIMD utility flow: Rust forced-fallback gate -> native argument and buffer checks -> scoped C++ CPU/selection gate -> unlocked scan."
  - "Utility ABI checks: every new public export is required by the closed symbol set and pinned to an exact generated signature."

requirements-completed: [UTIL-01, UTIL-02]

# Metrics
duration: 10min
completed: 2026-07-31
---

# Phase 12 Plan 04: Native SIMD Utility Surface Summary

**Capacity-safe minification and standalone UTF-8 validation now cross the Rust/C++ ABI with exact alias rules, CPU rejection, and post-success kernel locking**

## Performance

- **Duration:** 10 min
- **Started:** 2026-07-31T10:45:34Z
- **Completed:** 2026-07-31T10:55:47Z
- **Tasks:** 2
- **Files modified:** 10 task files

## Accomplishments

- Added `pure_simdjson_minify` and `pure_simdjson_validate_utf8` through the public Rust ABI, thin runtime wrappers, and C++ bridge.
- Enforced `dst_cap >= src_len` before kernel selection or any minify write, while allowing exact in-place operation and rejecting both partial-overlap directions without changing storage.
- Applied the standalone utility CPU policy: implicit fallback is rejected before selection locks, successful calls lock selection, and the mutex is released before the scan.
- Added raw-pointer, malformed-input, UTF-8, forced-fallback, fresh-process lock, required-symbol, and exact-signature coverage.

## Task Commits

Each task was committed atomically:

1. **Task 1: Minify/ValidateUTF8 C++ bridge and Rust wiring** - `000416d` (feat)
2. **Task 2: Rust utility tests and focused required-symbol/signature checks** - `ead1840` (test)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.h` - Declares the two private utility bridge functions.
- `src/native/simdjson_bridge.cpp` - Enforces buffer and CPU policy before calling upstream minify and UTF-8 scans.
- `src/runtime/mod.rs` - Adds raw-pointer bridge declarations and result-returning utility wrappers.
- `src/lib.rs` - Exposes panic-contained public utility exports with the Rust forced-fallback gate.
- `tests/rust_shim_minify.rs` - Covers aliasing, disjoint buffers, both overlap directions, capacity, pointer boundaries, malformed input, UTF-8, and forced fallback.
- `tests/rust_shim_utility_lock.rs` - Proves the precise pre-FFI rejection and successful-native-call lock sequence in a fresh test process.
- `tests/abi/check_header.py` - Requires both utility symbols and pins their parameter lists.
- `tests/abi/test_check_header.py` - Adds matching positive and wrong-signature fixtures.
- `cbindgen.toml` - Keeps the private utility bridge declarations out of the public header.
- `include/pure_simdjson.h` - Publishes the generated utility exports and their safety documentation.

## Decisions Made

- Destination overlap is checked against the caller-declared `dst_cap`, not only the bytes that a successful minify would eventually write. This keeps every non-identical layout conservatively disjoint.
- Empty minify and UTF-8 inputs still pass the successful native CPU gate, so they lock selection consistently with non-empty utility calls.
- Minify documentation states that success is not JSON validation: upstream detects unclosed strings but can accept other malformed JSON.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Excluded private utility bridge declarations from generated public headers**
- **Found during:** Task 2 (Rust utility tests and focused required-symbol/signature checks)
- **Issue:** cbindgen discovers private `psimdjson_*` extern declarations unless they are added to the established exclusion list, which would leak bridge-only symbols into the generated public header.
- **Fix:** Added `psimdjson_minify` and `psimdjson_validate_utf8` to `cbindgen.toml` before regenerating the header.
- **Files modified:** `cbindgen.toml`, `include/pure_simdjson.h`
- **Verification:** The focused `no-internal-symbols` rule and the complete `make verify-contract` gate passed.
- **Committed in:** `ead1840`

---

**Total deviations:** 1 auto-fixed (1 blocking issue).
**Impact on plan:** The fix preserves the existing private/public ABI boundary without widening feature scope.

## Issues Encountered

- A repository-wide `cargo fmt` invocation reformatted unrelated existing Rust files. Those incidental changes were reverted, and only the two new test files were formatted directly before commit.

## Verification

- `cargo build` - passed.
- `cargo test --test rust_shim_minify` - passed 11/11 tests.
- `cargo test --test rust_shim_utility_lock -- --test-threads=1` - passed 1/1 test.
- `python3 tests/abi/test_check_header.py` - passed 26/26 tests.
- Focused `required-symbols`, `diag-surface`, and `no-internal-symbols` checks - passed.
- `make verify-contract` - passed `cargo check`, all 122 Rust tests, deterministic header regeneration, every Python header rule, and C layout compilation.

## Threat and Security Impact

- **T-12-BOUNDS mitigated:** the native bridge rejects an undersized destination before kernel selection, upstream entry, or any write; the unchanged-buffer test proves the boundary behavior.
- **T-12-ALIAS mitigated for the implemented contract:** exact same-start aliasing and disjoint buffers succeed, while both partial-overlap directions return invalid argument without changing shared storage. Plan 12-10 still owns the durable x86-64 D-14 execution gate before any five-platform claim.
- **T-12-CPU mitigated:** implementation order rejects implicit fallback before locking and scans after mutex release. The fresh-process test proves only the Rust forced-fallback pre-gate and successful native lock transition; it does not claim to force the C++ implicit-fallback branch.
- **T-12-10 accepted as planned:** generated comments state that minification is not full JSON validation.
- No unmodeled network, authentication, schema, or file-access trust boundary was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12-07 can bind these exports into Go `Minify`, `MinifyInto`, and `ValidateUTF8` APIs.
- Plans 12-10 and 12-11 can consume the exact public symbols for the durable x86-64 gate, native smoke coverage, and required purego bindings.
- No blockers remain; cross-platform alias evidence is intentionally deferred to Plan 12-10.

## Self-Check: PASSED

- Verified the summary and all 10 implementation, test, checker, configuration, and generated-header files exist.
- Verified task commits `000416d` and `ead1840` exist in git history.
- Re-ran every task acceptance criterion and the plan-level/full Wave 5 contract gates successfully.
- Scanned all added lines for placeholder/TODO/FIXME and hardcoded-empty stub patterns; none were found.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
