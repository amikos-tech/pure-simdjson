---
phase: 02-rust-shim-minimal-parse-path
plan: "01"
subsystem: infra
tags: [rust, simdjson, c++, ffi, build-rs, cc]
requires:
  - phase: 01-ffi-contract-design
    provides: committed public ABI, error codes, and FFI contract docs
provides:
  - pinned simdjson v4.6.1 vendoring under third_party/simdjson
  - MSVC-safe build.rs compilation of the simdjson amalgamation and internal bridge
  - internal psimdjson bridge with noexcept entry points and catch-all exception containment
affects: [02-02 runtime core, phase-2 smoke harness, rust ffi bindings]
tech-stack:
  added: [cc crate, vendored simdjson v4.6.1 git submodule]
  patterns: [explicit native build inputs, internal pointer-based bridge ABI, catch-all C++ exception containment]
key-files:
  created: [build.rs, .gitmodules, src/native/simdjson_bridge.h, src/native/simdjson_bridge.cpp]
  modified: [Cargo.toml, third_party/simdjson]
key-decisions:
  - "Pinned simdjson as a git submodule at tag v4.6.1 and compiled the singleheader amalgamation directly from build.rs."
  - "Kept the bridge internal and pointer-based so Rust can call simdjson without widening include/pure_simdjson.h."
  - "Restricted GNU-specific static C++ runtime link args to linux-gnu targets while keeping MSVC configuration flag-free."
patterns-established:
  - "Native build inputs are explicit: third_party/simdjson/singleheader/simdjson.cpp plus src/native/simdjson_bridge.cpp only."
  - "Every psimdjson bridge entry point is noexcept and converts catch(...) into PURE_SIMDJSON_ERR_CPP_EXCEPTION."
requirements-completed: [SHIM-01, SHIM-02, SHIM-03, SHIM-04]
duration: 6m
completed: 2026-04-15
---

# Phase 02 Plan 01: Native Build and Bridge Summary

**Pinned simdjson v4.6.1 native build plumbing with an internal noexcept bridge for the minimal parse path**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-15T13:01:56Z
- **Completed:** 2026-04-15T13:08:05Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Pinned `third_party/simdjson` to upstream tag `v4.6.1` and recorded it as a git submodule.
- Added `build.rs` with `cc::Build::cpp(true).std("c++17")`, explicit simdjson/bridge inputs, rerun guards, and linux-gnu-only static C++ runtime link args.
- Replaced the temporary bridge stub with a real internal `psimdjson_*` bridge that exposes implementation metadata, padding, parse/root helpers, diagnostics, type/int64 access, and deterministic C++ exception containment.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pin the vendored simdjson amalgamation and make `build.rs` MSVC-safe** - `a898eb9` (`feat`)
2. **Task 2: Add a narrow bridge with explicit C++ exception containment and padding introspection** - `9b46e02` (`feat`)

## Files Created/Modified

- `Cargo.toml` - Added the `cc` build dependency while preserving `crate-type = ["cdylib", "staticlib"]`.
- `build.rs` - Compiles the vendored simdjson amalgamation and bridge with deterministic inputs and rerun triggers.
- `.gitmodules` - Records the vendored simdjson submodule location and remote.
- `third_party/simdjson` - Pinned submodule checkout at `v4.6.1`.
- `src/native/simdjson_bridge.h` - Declares the narrow internal bridge ABI with `noexcept` entry points only.
- `src/native/simdjson_bridge.cpp` - Implements non-throwing simdjson helpers with `catch (...)` translation to `PURE_SIMDJSON_ERR_CPP_EXCEPTION`.

## Decisions Made

- Used the simdjson singleheader amalgamation directly instead of cmake to keep the Phase 2 native build deterministic and minimal.
- Kept bridge symbols internal under `psimdjson_*` and out of `include/pure_simdjson.h` so public ABI scope stays locked to the Phase 1 contract.
- Used `parse_into_document(..., false)` in the bridge so Phase 2 Rust code can supply already-padded buffers and keep padding ownership on the Rust side.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added a minimal bridge skeleton in Task 1 so `build.rs` had valid native inputs**
- **Found during:** Task 1 (Pin the vendored simdjson amalgamation and make `build.rs` MSVC-safe)
- **Issue:** The plan required `build.rs` to compile `src/native/simdjson_bridge.cpp` before Task 2 created that bridge surface.
- **Fix:** Created a temporary header/source skeleton in Task 1, then replaced it with the real bridge implementation in Task 2.
- **Files modified:** `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp`
- **Verification:** `git -C third_party/simdjson describe --tags --exact-match | grep -x 'v4.6.1'` and `cargo build --release`
- **Committed in:** `a898eb9` (part of Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix was required to satisfy Task 1's build verification without changing scope. Task 2 still delivered the full planned bridge surface.

## Issues Encountered

- `git submodule add` left a partially initialized gitdir without writing `.gitmodules`; resolved by checking out `v4.6.1`, writing `.gitmodules` explicitly, and staging the gitlink.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `src/lib.rs` can now bind against a stable internal `psimdjson_*` bridge for implementation name, padding, parse, root, and int64 operations.
- Release builds already emit the expected Rust static and dynamic artifacts with vendored simdjson compiled in.

## Self-Check: PASSED

- Summary file exists at `.planning/phases/02-rust-shim-minimal-parse-path/02-01-SUMMARY.md`.
- Task commit `a898eb9` exists in git history.
- Task commit `9b46e02` exists in git history.

---
*Phase: 02-rust-shim-minimal-parse-path*
*Completed: 2026-04-15*
