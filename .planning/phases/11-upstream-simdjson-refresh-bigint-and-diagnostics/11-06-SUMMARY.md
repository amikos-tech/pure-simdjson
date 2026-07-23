---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 06
subsystem: native-abi
tags: [simdjson, kernel-selection, cpu-dispatch, concurrency, ffi, rust, cpp]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-04 configured parser construction and Plan 11-05 native/Rust diagnostic transport
provides:
  - Process-global exact-name implementation selection with runtime CPU-support validation
  - One native synchronization boundary for selection, explicit locking, and parser construction
  - Append-only kernel-locked status 10 and public Rust setter/lock exports
  - Subprocess-isolated auto, valid, invalid, unsupported, fallback, post-lock, and race proof
affects: [11-07, 11-08, 11-09, 11-12, public-abi, kernel-diagnostics]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Validate a compiled implementation and its runtime CPU support before assigning the process-global active pointer
    - Test irreversible process-global state in one fresh subprocess per scenario

key-files:
  created:
    - tests/rust_shim_kernel.rs
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-06-SUMMARY.md
  modified:
    - build.rs
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/lib.rs

key-decisions:
  - "Use one native mutex/state for setter, explicit lock, and both parser constructors; a valid construction attempt sets locked before allocation."
  - "Keep the native state authoritative for production fallback policy while retaining the existing Rust forced-fallback seam only for isolated tests."
  - "Compile the upstream fallback implementation on every supported architecture so explicit diagnostic fallback remains available on arm64 as well as x86-64."

patterns-established:
  - "Irreversible kernel lifecycle: unlocked selection -> explicit lock or first valid parser construction -> permanently locked."
  - "Mutation safety: missing or CPU-unsupported names return before assignment and leave both the active name and explicit-selection state unchanged."

requirements-completed: [DIAG-01]

# Metrics
duration: 11min
completed: 2026-07-23
---

# Phase 11 Plan 06: Process-Global Kernel Control Summary

**Exact-name simdjson kernel selection now validates host CPU support, shares one irreversible lock with parser construction, and distinguishes explicit diagnostic fallback from silent automatic fallback**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-23T13:19:13Z
- **Completed:** 2026-07-23T13:30:10Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added a native process-global state machine whose mutex serializes exact-name selection, explicit locking, legacy parser construction, and configured parser construction.
- Validated compiled implementation names case-sensitively and called `supported_by_runtime_system()` before assignment; invalid and unsupported attempts preserve the active implementation.
- Locked implementation selection before parser allocation, returning append-only status `PURE_SIMDJSON_ERR_KERNEL_LOCKED = 10` for every later mutation.
- Preserved the project fallback policy: explicit `fallback` selection is accepted for diagnostics, while silent/test-forced automatic fallback still returns CPU unsupported.
- Proved ten irreversible scenarios in separate child processes, including both parser constructors and a setter-versus-constructor race.

## Task Commits

Each task and TDD gate was committed atomically:

1. **Task 1: Implement one synchronized native implementation-selection state machine** - `efdcefa` (feat)
2. **Task 2 RED: Add the failing kernel lifecycle contract** - `9d27f20` (test)
3. **Task 2 GREEN: Expose kernel setter/lock and status 10** - `7673498` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `build.rs` - Forces the upstream fallback implementation into every supported target build so explicit diagnostics remain cross-platform.
- `src/native/simdjson_bridge.h` - Declares the internal native setter and explicit lock bridge functions.
- `src/native/simdjson_bridge.cpp` - Implements exact lookup/support validation, explicit-selection tracking, fallback policy, and the shared irreversible mutex.
- `src/runtime/mod.rs` - Wraps native setter/lock calls and confines the forced-fallback precheck to the existing test seam.
- `src/lib.rs` - Appends status 10 and exports `pure_simdjson_set_implementation` plus `pure_simdjson_lock_implementation_selection` through `ffi_wrap`.
- `tests/rust_shim_kernel.rs` - Runs auto, valid, invalid, unsupported-when-observable, fallback, post-lock, configured-constructor, and race cases in fresh subprocesses.

## Decisions Made

- Valid argument and limit checks occur before the lifecycle lock; once a valid parser construction attempt reaches native creation, selection locks before any allocation and stays locked even if allocation later fails.
- Empty selection assigns upstream `detect_best_supported()` and clears explicit-selection state; only a successful nonempty selection sets the explicit flag.
- The native constructor is authoritative for production automatic-fallback rejection. Rust checks only the pre-existing forced-fallback test override, preventing it from rejecting a legitimately explicit native fallback.
- Repeated explicit lock calls are idempotent successes; no unlock or reset-after-lock production API exists.

## TDD Gate Compliance

- **RED:** `9d27f20` failed to compile only because status 10 and the setter/lock exports were absent.
- **GREEN:** `7673498` added the public exports and fallback portability support; all ten isolated child scenarios passed.
- No refactor commit was needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Compiled diagnostic fallback on arm64**
- **Found during:** Task 2 GREEN verification
- **Issue:** Upstream disables its fallback implementation by default when arm64 is guaranteed available, so exact lookup correctly returned invalid argument and the required explicit-fallback diagnostic path could not exist on two supported release targets.
- **Fix:** Defined `SIMDJSON_IMPLEMENTATION_FALLBACK=1` for the existing native build, without adding a source tree, dependency, or alternate selection path.
- **Files modified:** `build.rs`
- **Verification:** The explicit-fallback child creates and frees a parser successfully on arm64; all Rust tests and the fresh release-library Go race suite pass.
- **Committed in:** `7673498`

---

**Total deviations:** 1 auto-fixed missing critical portability requirement.
**Impact on plan:** The fix makes the planned explicit-fallback behavior consistent across supported architectures without widening the public API or dependency surface.

## Issues Encountered

- The vendored v4.6.4 `atomic_ptr` exposes implicit pointer conversion rather than `.load()`; native compilation identified the mismatch and the bridge now uses the documented conversion operator.
- Repository-wide `cargo fmt --check` still reports the pre-existing unrelated drift recorded in `deferred-items.md`; all Rust lines changed by this plan were formatted locally.

## Verification

- `cargo check --locked && cargo test --locked --lib -- --test-threads=1` — passed; all 16 library tests green.
- `cargo test --locked --test rust_shim_kernel -- --test-threads=1` — passed; all ten subprocess scenarios green.
- Kernel subprocess/race test repeated 10 consecutive times — passed without crash, partial state, or unexpected status.
- `cargo test --locked --test rust_shim_fallback_gate -- --test-threads=1` — both legacy forced-fallback policy tests passed.
- `cargo test --locked -- --test-threads=1` — all 83 Rust unit/integration tests passed.
- `cargo build --release --locked` — fresh optimized library built from locked inputs.
- `PURE_SIMDJSON_LIB_PATH=<fresh target/release dylib> go test ./... -race` — all four Go packages passed.
- Release-library symbol inspection found both `pure_simdjson_set_implementation` and `pure_simdjson_lock_implementation_selection`.
- `make verify-contract` remains intentionally deferred to Plan 11-09; generated header and ABI-audit synchronization are the only planned interim mismatch.

## Threat and Security Impact

- **T-11-02 mitigated:** exact compiled-name lookup and `supported_by_runtime_system()` both succeed before assignment; missing and unsupported attempts preserve the prior active implementation.
- **T-11-06 mitigated:** one mutex serializes mutation, explicit lock, and both constructors; the state becomes locked before allocation, and repeated race testing admits only setter success/locked plus constructor success.
- **T-11-SC preserved:** no package install, dependency, lockfile, secondary source tree, network endpoint, authentication path, schema, or file-access trust boundary changed.
- No unplanned security-relevant surface was introduced; both new FFI exports and the process-global state are covered by the plan threat model.

## Known Stubs

None. Generated public-header and cross-ABI mirror synchronization is dependency-ordered Plan 11-09 work, not a placeholder in this implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-07 can append status 10 to the coordinated ABI 1.2 Rust/Go mirrors.
- Plans 11-08 and 11-09 can bind and generate the now-proven setter/lock exports.
- Plan 11-12 can add the package-level Go `Kernel`/`SetKernel` lifecycle on this native state machine.
- No blockers remain for subsequent Phase 11 plans.

## Self-Check: PASSED

- All six implementation/test artifacts and this summary exist.
- Task commits `efdcefa`, `9d27f20`, and `7673498` are present in repository history.
- All task acceptance criteria and plan-level verification commands passed.
- The only unrelated dirty paths remain the pre-existing config and Phase 10 learnings files.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
