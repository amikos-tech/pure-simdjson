---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 04
subsystem: native-abi
tags: [simdjson, parser-limits, capacity, depth, ffi, rust, cpp]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plans 11-02/11-03 BigInt-enabled native parser and copied scalar ABI
provides:
  - Additive configured parser construction with normalized immutable capacity and depth
  - Capacity status 9 enforced before Rust arena detachment, padding, growth, or copy
  - Exact configured and default upstream depth-boundary tests
  - Diagnostic reset before Rust-side capacity rejection
affects: [11-05, 11-07, 11-09, parser-options, diagnostics]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Normalize parser limits once and store the effective values in both native and registry state
    - Validate work bounds while reusable storage is still attached to its owner

key-files:
  created:
    - tests/rust_shim_limits.rs
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-04-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/lib.rs

key-decisions:
  - "Normalize zero to capacity 0xFFFFFFFF and depth 1024 before native allocation, then store those exact effective values in C++ and Rust."
  - "Initialize simdjson's configured depth with a zero-capacity native allocation so no input-sized work occurs during construction."
  - "Clear native diagnostics only after handle and busy validation, immediately before the authoritative Rust capacity gate."

patterns-established:
  - "Pre-copy capacity gate: validate handle/busy state, clear diagnostics, compare length, then and only then detach and prepare the reusable arena."
  - "Depth contract: configured/default maximum N accepts N-1 nested containers and rejects N."

requirements-completed: [LIMIT-01]

# Metrics
duration: 16min
completed: 2026-07-23
---

# Phase 11 Plan 04: Immutable Native Parser Limits Summary

**Configured parsers now enforce exact capacity and depth bounds, rejecting oversized input before any Rust arena allocation or copy**

## Performance

- **Duration:** 16 min
- **Started:** 2026-07-23T12:19:12Z
- **Completed:** 2026-07-23T12:36:09Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added private configured native construction while preserving the legacy constructor and its current `0xFFFFFFFF`/`1024` defaults.
- Stored the normalized capacity/depth pair on `psimdjson_parser` for primary DOM parsing and Plan 11-05 diagnostic replay.
- Added append-only capacity status `9` and a panic-safe configured Rust C export without renumbering existing statuses.
- Enforced capacity while the reusable input arena remains attached to its parser, before `mem::take`, padding arithmetic, resize, or copy.
- Proved exact 32/33-byte capacity behavior, configured 3/4 and default 1023/1024 depth boundaries, stale-detail clearing, and byte-for-byte arena preservation.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add configured native construction while retaining the legacy constructor** - `b88d747` (feat)
2. **Task 2 RED: Add the failing parser-limit contract** - `5d342a0` (test)
3. **Task 2 GREEN: Enforce configured limits before Rust arena work** - `7eefeec` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.h` - Declares configured construction and the private diagnostic-reset seam.
- `src/native/simdjson_bridge.cpp` - Normalizes, stores, and applies exact native capacity/depth values before first parse.
- `src/runtime/mod.rs` - Wraps configured construction and diagnostic reset across the Rust/C++ boundary.
- `src/runtime/registry.rs` - Stores parser configuration and owns the authoritative pre-copy capacity gate.
- `src/lib.rs` - Appends capacity status `9` and exports `pure_simdjson_parser_new_configured` through `ffi_wrap`.
- `tests/rust_shim_limits.rs` - Locks constructor validation, capacity, depth, and stale-diagnostic behavior.

## Decisions Made

- The legacy Rust path continues through the legacy native constructor; configured calls use the additive constructor. Both retain `number_as_string(true)`.
- Native depth is established with `parser.allocate(0, effective_max_depth)`, which configures upstream recursion state without allocating an input-sized parser buffer.
- Capacity is normalized and validated in Rust before native allocation, then validated again by the private C++ constructor as defense in depth.

## TDD Gate Compliance

- **RED:** `5d342a0` added integration and white-box limit contracts. The integration target failed only because status `9` and `pure_simdjson_parser_new_configured` did not exist.
- **GREEN:** `7eefeec` added the minimal status, export, stored configuration, diagnostic reset, and pre-copy gate. All five integration cases and the registry arena test passed.
- No refactor commit was needed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first combined `cargo check`/library-test run hit a transient missing Cargo incremental-cache file after the native compile. Retrying the exact test command succeeded; no source or cache workaround was required.

## Verification

- `cargo check --locked` — passed without warnings.
- `cargo test --locked --test rust_shim_limits -- --test-threads=1` — 5/5 passed.
- `cargo test --locked registry::tests::capacity -- --test-threads=1` — unchanged-arena white-box test passed.
- `cargo test --locked --test rust_shim_minimal -- --test-threads=1` — 22/22 lifecycle and compatibility tests passed.
- `cargo test --locked --lib -- --test-threads=1` — 16/16 passed.
- `cargo build --release --locked` — passed against the provenance-locked patched v4.6.4 source.
- `PURE_SIMDJSON_LIB_PATH=<fresh target/release dylib> go test ./... -race` — all four Go packages passed.
- Source-order inspection confirms the capacity comparison precedes `mem::take`, padding `checked_add`, `resize`, and `copy_from_slice`.
- `make verify-contract` remains intentionally deferred through Plan 11-08 because Plan 11-09 owns the single complete generated-header synchronization.

## Threat and Security Impact

- **T-11-01 mitigated:** oversized input returns status `9` before wrapper-owned size-dependent work and leaves the arena pointer, bytes, length, and capacity unchanged.
- **T-11-05 mitigated for this path:** native message and offset state is cleared before a Rust-side capacity rejection.
- **T-11-SC preserved:** no dependency, manifest, lockfile, generated header, package install, or publication path changed.
- No unplanned network endpoint, authentication path, schema change, or file-access trust boundary was introduced.

## Known Stubs

None. Generated public-header synchronization is dependency-ordered Plan 11-09 work, not a placeholder implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-05 can consume `psimdjson_parser::max_capacity` and `max_depth` directly for both bounded diagnostic replay passes.
- Later Go option and ABI-mirror plans can bind the configured export and status `9` without changing native limit semantics.
- No blockers remain for subsequent Phase 11 plans.

## Self-Check: PASSED

- All six implementation/test artifacts and this summary exist.
- Task commits `b88d747`, `5d342a0`, and `7eefeec` are present in repository history.
- The only remaining unrelated dirty paths are the pre-existing config and Phase 10 learnings files.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
