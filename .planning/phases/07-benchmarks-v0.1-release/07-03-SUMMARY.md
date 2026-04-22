---
phase: 07-benchmarks-v0.1-release
plan: "03"
subsystem: ffi
tags: [benchmarks, ffi, telemetry, rust, c++, purego]
requires:
  - phase: "07-01"
    provides: "repo-local benchmark inputs and oracle assets that keep BENCH-05 grounded in committed testdata"
  - phase: "07-02"
    provides: "the benchmark harness and comparator surface that will consume the new allocator telemetry"
provides:
  - "C++ allocator telemetry compiled into the native cdylib"
  - "Public reset/snapshot ABI for native allocator stats"
  - "Go FFI helpers for benchmark-side allocator snapshots"
  - "Contract, audit, Rust-test, and C-smoke coverage for the new telemetry surface"
affects: [BENCH-05, 07-04, benchmark helpers, internal/ffi]
tech-stack:
  added: []
  patterns:
    - "Diagnostic-only reset/snapshot ABI additions ship with header regeneration, contract docs, header audits, and smoke coverage in one change"
    - "Allocator telemetry uses resettable epochs so persistent pre-reset allocations do not pollute benchmark snapshots"
key-files:
  created:
    - src/native/native_alloc_telemetry.h
    - src/native/native_alloc_telemetry.cpp
  modified:
    - build.rs
    - src/lib.rs
    - src/runtime/mod.rs
    - include/pure_simdjson.h
    - internal/ffi/bindings.go
    - internal/ffi/types.go
    - docs/ffi-contract.md
    - tests/abi/check_header.py
    - tests/rust_shim_minimal.rs
    - tests/smoke/ffi_export_surface.c
key-decisions:
  - "Native allocator telemetry is epoch-based: reset excludes pre-existing live allocations from later snapshots instead of claiming process-wide totals."
  - "The allocator stats surface remains diagnostic-only and is published strictly as reset/snapshot exports plus a fixed four-field struct."
  - "Header-audit verification must work both through Makefile rules and the planner's direct `python3 tests/abi/check_header.py include/pure_simdjson.h` command."
patterns-established:
  - "Every new public ABI struct must be reflected in cbindgen output, contract docs, header rules, Rust integration tests, and C export smoke in the same plan."
  - "Go internal FFI helpers expose new diagnostic exports as typed helper methods before benchmark code consumes them."
requirements-completed: [BENCH-05]
duration: 20 min
completed: 2026-04-22
---

# Phase 7 Plan 03: Native Allocator Telemetry Summary

**Epoch-based native allocator telemetry in the C++ cdylib with audited reset/snapshot ABI and Go binding hooks**

## Performance

- **Duration:** 20 min
- **Started:** 2026-04-22T19:36:30Z
- **Completed:** 2026-04-22T19:56:07Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments

- Added `src/native/native_alloc_telemetry.cpp` and wired it into `build.rs` so the cdylib now carries a concrete C++ allocation counter implementation instead of a Rust-only guess.
- Exposed `pure_simdjson_native_alloc_stats_t`, `pure_simdjson_native_alloc_stats_reset`, and `pure_simdjson_native_alloc_stats_snapshot` through Rust, the generated header, and Go `internal/ffi` helpers.
- Updated the ABI contract, header-audit rules, Rust integration tests, and the C export-surface smoke binary so the new diagnostic surface is documented and executable.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the C++ telemetry implementation and bridge hooks** - `cc54c5d` (`feat`)
2. **Task 2: Thread the telemetry through the ABI, bindings, docs, and smoke tests** - `d22d9b3` (`feat`)

## Files Created/Modified

- `src/native/native_alloc_telemetry.h` - Declares the native telemetry reset/snapshot helpers.
- `src/native/native_alloc_telemetry.cpp` - Implements the replaceable `new/delete` telemetry shim and epoch-based counters.
- `build.rs` - Compiles the new telemetry translation unit into the native cdylib.
- `src/lib.rs`, `src/runtime/mod.rs` - Publish the allocator stats struct plus reset/snapshot exports through the Rust ABI.
- `include/pure_simdjson.h`, `docs/ffi-contract.md`, `tests/abi/check_header.py`, `tests/abi/test_check_header.py`, `Makefile` - Lock the public contract and header-audit behavior around the new diagnostic surface.
- `internal/ffi/bindings.go`, `internal/ffi/types.go` - Add typed Go helpers for benchmark-side allocator resets and snapshots.
- `tests/rust_shim_minimal.rs`, `tests/smoke/ffi_export_surface.c` - Verify the new exports before and after parse activity in both Rust and C.

## Decisions Made

- Native allocation reporting is scoped to allocations routed through the shim/simdjson cdylib path. It does not claim process-wide or Go-heap totals.
- Reset semantics are epoch-based so benchmark helpers can zero the diagnostic counters without invalidating pre-existing live native allocations.
- The header audit CLI now defaults to all rules when no `--rule` flags are passed, because the plan's explicit verification command depends on that path working directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added a typed Go stats carrier for the new snapshot helper**
- **Found during:** Task 2
- **Issue:** `internal/ffi/bindings.go` could not expose `NativeAllocStatsSnapshot()` cleanly without a concrete `NativeAllocStats` type.
- **Fix:** Added `internal/ffi/types.go::NativeAllocStats` and returned it from the new Go binding helper.
- **Files modified:** `internal/ffi/types.go`, `internal/ffi/bindings.go`
- **Verification:** `go test ./internal/ffi`
- **Committed in:** `d22d9b3`

**2. [Rule 3 - Blocking] Prevented private bridge hooks from leaking into the generated public header**
- **Found during:** Task 2 verification
- **Issue:** `cbindgen` initially emitted `psimdjson_native_alloc_stats_*` into `include/pure_simdjson.h`, which broke the C++ build by mixing private bridge declarations into the public contract.
- **Fix:** Added the new bridge symbols to `cbindgen.toml`'s exclude list and regenerated the header.
- **Files modified:** `cbindgen.toml`, `include/pure_simdjson.h`
- **Verification:** `cargo test --release native_alloc_stats`
- **Committed in:** `d22d9b3`

**3. [Rule 3 - Blocking] Made the direct header-audit CLI match the plan's verification command**
- **Found during:** Task 2 verification
- **Issue:** `tests/abi/check_header.py` required explicit `--rule` flags, but the plan verifies it as `python3 tests/abi/check_header.py include/pure_simdjson.h`.
- **Fix:** Defaulted the script to all registered rules when no `--rule` flags are supplied and updated the unit test fixture generator for zero-arg functions.
- **Files modified:** `tests/abi/check_header.py`, `tests/abi/test_check_header.py`
- **Verification:** `python3 tests/abi/check_header.py include/pure_simdjson.h`, `python3 tests/abi/test_check_header.py`
- **Committed in:** `d22d9b3`

---

**Total deviations:** 3 auto-fixed (3 blocking)
**Impact on plan:** All three fixes were necessary to keep the public/private ABI split correct and to make the plan's own verification path executable. No scope creep beyond BENCH-05 support.

## Issues Encountered

- The generated header briefly leaked private `psimdjson_*` bridge hooks until the cbindgen exclude list was extended for the new telemetry symbols.
- The task plan's direct header-audit command did not match the current CLI contract, so the audit script had to be aligned with the planned verification path.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BENCH-05 now has a concrete native telemetry source that benchmark helpers can reset and snapshot without guessing from Go heap statistics.
- The public contract, Go binding hooks, Rust tests, and C smoke coverage are all in place for later benchmark plans to consume directly.
- No blockers remain for benchmark consumers of the allocator telemetry surface.

## Self-Check: PASSED

- Found `.planning/phases/07-benchmarks-v0.1-release/07-03-SUMMARY.md`
- Found task commit `cc54c5d`
- Found task commit `d22d9b3`

---
*Phase: 07-benchmarks-v0.1-release*
*Completed: 2026-04-22*
