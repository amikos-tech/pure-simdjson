---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 05
subsystem: native-abi
tags: [simdjson, diagnostics, ondemand, error-offsets, ffi, rust, cpp]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-04 immutable effective parser capacity/depth and Spike 001 v4.6.4 replay evidence
provides:
  - Truthful upstream-proven diagnostic offsets with an independent known bit
  - At most two fresh failure-only On-Demand replay passes under exact parser limits
  - Checked integer-domain pointer proof and explicit recursive depth enforcement
  - Golden v4.6.4 malformed-input offsets plus success/capacity stale-state coverage
affects: [11-06, 11-09, diagnostics, public-abi, generated-header]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Treat a diagnostic location as known only after a checked in-range non-end pointer proof
    - Bound diagnostic replay with the caller's immutable capacity/depth and a fixed two-parser topology

key-files:
  created:
    - tests/rust_shim_diagnostics.rs
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-05-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/lib.rs
    - tests/rust_shim_minimal.rs

key-decisions:
  - "Replay only ordinary syntax failures: raw_json/at_end first, recursive public-accessor consumption second only when the first pass is fully valid."
  - "Represent unknown as UINT64_MAX plus false, and expose the bool independently so a proven byte-zero location remains distinguishable."
  - "Construct and allocate both fresh replay parsers with the exact stored effective capacity/depth; resource or limit failure ends replay immediately."

patterns-established:
  - "Proven offset: convert addresses to uintptr_t, reject addition overflow, prove start <= location < end, then subtract."
  - "Bounded failure diagnostics: successful input pays no replay; an eligible failure pays at most two caller-bounded linear scans."

requirements-completed: [DIAG-02]

# Metrics
duration: 26min
completed: 2026-07-23
---

# Phase 11 Plan 05: Truthful Native Diagnostic Offsets Summary

**Bounded v4.6.4 On-Demand replay now reports only upstream-proven byte offsets, preserving known byte zero independently from explicit unknown**

## Performance

- **Duration:** 26 min
- **Started:** 2026-07-23T12:44:13Z
- **Completed:** 2026-07-23T13:10:22Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added a failure-only two-parser On-Demand replay path with no retry loop, no parser/document reuse after error, and exact Plan 11-04 capacity/depth on both passes.
- Proved diagnostic pointers in checked `uintptr_t` space before subtraction; exact-end, below-start, out-of-range, overflow, iterate, resource, and internal failures stay `(UINT64_MAX, false)`.
- Added an additive public known-offset getter while leaving the existing numeric offset export unchanged, including a natural `(0, true)` proof for malformed root token `x`.
- Locked capacity N-1/N, depth N-1/N, no-amplification, stale-success, stale-capacity, and the full pinned v4.6.4 location corpus in Rust integration tests.

## v4.6.4 Characterization

The initial characterization run produced the same mapping as Spike 001. Replay pass `1` is
raw JSON/`at_end`, pass `2` is recursive public-accessor consumption; pointer relation `0`
means no usable pointer was queried and `1` means the pointer was proven in bounds.

| Case | Primary status | Replay pass | Replay status | Location status | Pointer relation | Offset | Known |
|---|---:|---:|---:|---:|---:|---:|---|
| empty | 13 | 1 | 13 | 12 | 0 | `UINT64_MAX` | false |
| invalid UTF-8 | 11 | 1 | 11 | 12 | 0 | `UINT64_MAX` | false |
| unclosed string | 15 | 1 | 15 | 12 | 0 | `UINT64_MAX` | false |
| array trailing comma | 3 | 2 | 3 | 0 | 1 | 3 | true |
| trailing content | 3 | 1 | 31 | 0 | 1 | 8 | true |
| missing object key | 3 | 2 | 3 | 0 | 1 | 16 | true |
| unexpected root token `x` | 3 | 2 | 3 | 0 | 1 | 0 | true |
| extra closing bracket | 3 | 1 | 31 | 0 | 1 | 15 | true |
| mismatched container | 3 | 1 | 3 | 0 | 1 | 9 | true |

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add the failing native replay contract** - `13f9451` (test)
2. **Task 1 GREEN: Capture bounded native diagnostic offsets** - `0ec5512` (feat)
3. **Task 2 RED: Add the failing known-offset transport contract** - `9439eb8` (test)
4. **Task 2 GREEN: Expose known diagnostic offsets** - `4a3817f` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.h` - Declares the native known-bit getter and internal `psimdjson_test_*` replay observation seams.
- `src/native/simdjson_bridge.cpp` - Implements bounded raw/recursive replay, explicit depth checks, checked pointer proof, terminal resource handling, and atomic diagnostic reset.
- `src/runtime/mod.rs` - Wraps the native known-bit getter.
- `src/runtime/registry.rs` - Transports known state through validated parser handles.
- `src/lib.rs` - Exports `pure_simdjson_parser_get_last_error_has_offset` through `ffi_wrap`.
- `tests/rust_shim_diagnostics.rs` - Locks characterization, replay bounds, pointer boundaries, public transport, and stale-state behavior.
- `tests/rust_shim_minimal.rs` - Uses an upstream-unavailable unclosed-string case for its intentional unknown-offset compatibility assertion.

## Decisions Made

- Primary `CAPACITY`, `DEPTH_ERROR`, `MEMALLOC`, and internal failures launch no replay; the same classes terminate whichever replay pass encounters them.
- The recursive pass consumes the root document directly through templated public On-Demand accessors, starting at depth zero and requiring each entered container depth to remain strictly below the configured maximum.
- Generated C header and ABI-audit synchronization remains dependency-ordered Plan 11-09 work; this plan changes the Rust/native implementation only.

## TDD Gate Compliance

- **Task 1 RED:** `13f9451` linked unsuccessfully only because the four planned internal replay observation seams did not exist.
- **Task 1 GREEN:** `0ec5512` implemented those seams and the production replay path; all five native diagnostic tests passed.
- **Task 2 RED:** `9439eb8` failed to compile only because `pure_simdjson_parser_get_last_error_has_offset` did not exist.
- **Task 2 GREEN:** `4a3817f` added the runtime, registry, and public transport; all seven diagnostic tests passed.
- No refactor commit was needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Consumed the On-Demand root document directly**
- **Found during:** Task 1 characterization
- **Issue:** Calling `document.get_value()` made the natural root-token `x` case fail with `SCALAR_DOCUMENT_AS_VALUE`, incorrectly leaving its proven byte-zero location unknown.
- **Fix:** Generalized the recursive public-accessor consumer over both `ondemand::document` and child `ondemand::value`, matching the validated Spike 001 traversal.
- **Files modified:** `src/native/simdjson_bridge.cpp`
- **Verification:** The characterized `x` fixture now reports `(0, true)` through both native and public Rust paths.
- **Committed in:** `0ec5512`

**2. [Rule 1 - Bug] Replaced an obsolete intentional-unknown fixture**
- **Found during:** Task 1 compatibility verification
- **Issue:** The earlier minimal test used `{`, but the upstream replay now truthfully proves that failure at byte zero.
- **Fix:** Switched the test to the documented unclosed-string case, for which upstream cannot provide a location.
- **Files modified:** `tests/rust_shim_minimal.rs`
- **Verification:** All 22 minimal lifecycle/compatibility tests pass and the fixture remains `(UINT64_MAX, false)`.
- **Committed in:** `0ec5512`

---

**Total deviations:** 2 auto-fixed bugs.
**Impact on plan:** Both fixes were required to preserve the planned upstream-truth contract; no new feature or dependency was added.

## Issues Encountered

- Cargo twice reported a transient missing incremental dependency-graph file before tests ran. Retrying the unchanged locked command succeeded; no source, dependency, or cache workaround was applied.
- Repository-wide formatter drift outside this plan's changed lines is recorded in `deferred-items.md`; new Rust code was formatted locally without rewriting unrelated files.

## Verification

- `cargo test --locked --test rust_shim_diagnostics characterize_v464_error_locations -- --exact --nocapture --test-threads=1` — exact nine-case mapping above passed.
- `cargo test --locked --test rust_shim_diagnostics diagnostic_replay_ -- --test-threads=1` — capacity, depth, and terminal-resource replay contracts passed.
- `cargo check --locked` — passed without warnings.
- `cargo test --locked -- --test-threads=1` — all 82 Rust unit/integration tests passed.
- `cargo build --release --locked` — fresh optimized library built against locked inputs.
- `PURE_SIMDJSON_LIB_PATH=<fresh target/release dylib> go test ./... -race` — all four Go packages passed.
- Structural source checks found exactly two fresh replay parser constructions and two exact-limit allocations, no default-limit parser, retry loop, secondary JSON parser, guessed index, or error-message parsing.
- `make verify-contract` remains intentionally deferred to Plan 11-09; the generated header not yet carrying the additive known-bit export is the single planned interim mismatch.

## Threat and Security Impact

- **T-11-05 mitigated:** address addition is overflow-checked, range proof precedes subtraction, only non-end in-bounds pointers become known, and prior diagnostic state is cleared on every attempt/success.
- Replay cannot amplify resource exhaustion beyond two fresh caller-bounded scans; configured capacity/depth is preserved exactly and recursive depth is guarded explicitly.
- **T-11-SC preserved:** no package install, dependency, manifest, lockfile, generated header, publication path, network endpoint, authentication path, schema, or file-access trust boundary changed.
- No unplanned security-relevant surface was introduced; the additive FFI getter and internal test seams are covered by the plan threat model.

## Known Stubs

None. Generated public-header synchronization is owned by Plan 11-09 and is not a placeholder in this implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-06 can add process-global kernel-selection diagnostics on top of truthful per-parser locations.
- Plan 11-09 can expose the additive known-bit getter in the generated header and reconcile all ABI mirrors in one synchronization step.
- No blockers remain for subsequent Phase 11 plans.

## Self-Check: PASSED

- All seven implementation/test artifacts, this summary, and the deferred-item record exist.
- Task commits `13f9451`, `0ec5512`, `9439eb8`, and `4a3817f` are present in repository history.
- The only unrelated dirty paths remain the pre-existing config and Phase 10 learnings files.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
