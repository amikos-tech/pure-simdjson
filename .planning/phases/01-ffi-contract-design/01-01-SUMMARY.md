---
phase: 01-ffi-contract-design
plan: "01"
subsystem: api
tags: [rust, ffi, cbindgen, abi, header-generation]
requires: []
provides:
  - ABI-source Rust crate metadata with cdylib/staticlib outputs
  - Stable cbindgen configuration for reproducible C header generation
  - Committed generated include/pure_simdjson.h baseline with ABI version export
affects: [phase-1-plan-02, ffi-contract, go-bindings, header-regeneration]
tech-stack:
  added: [cbindgen, libc]
  patterns: [rust-abi-source-of-truth, committed-generated-header]
key-files:
  created: [Cargo.toml, src/lib.rs, cbindgen.toml, include/pure_simdjson.h]
  modified: []
key-decisions:
  - "Use src/lib.rs as the single ABI source that drives cbindgen output."
  - "Guard the bootstrap ABI version export against null out-pointers until the full error space lands in Plan 02."
patterns-established:
  - "Bootstrap public ABI items live in Rust first, then flow into include/pure_simdjson.h via cbindgen."
  - "Generated headers are committed artifacts and must round-trip byte-for-byte from repository state."
requirements-completed: [FFI-01]
duration: 11m
completed: 2026-04-14
---

# Phase 1 Plan 1: FFI Contract Design Summary

**Rust ABI-source crate bootstrap with a reproducible cbindgen pipeline and committed pure_simdjson header baseline**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-14T11:46:02Z
- **Completed:** 2026-04-14T11:56:46Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Bootstrapped the repository into a compilable Rust ABI-source crate with `cdylib` and `staticlib` outputs.
- Added a stable `cbindgen.toml` and generated `include/pure_simdjson.h` directly from Rust source.
- Locked the initial ABI version export into both `src/lib.rs` and the committed header so later plans extend a concrete contract.

## Task Commits

Each task was committed atomically:

1. **Task 1: Install `cbindgen` and scaffold the ABI-source crate** - `e10300c` (`feat`)
2. **Task 2: Add stable `cbindgen` config and commit the generated header baseline** - `c5b3bc0` (`feat`)

## Files Created/Modified

- `Cargo.toml` - Defines the `pure_simdjson` ABI-source crate, output kinds, dependency baseline, and release panic policy.
- `src/lib.rs` - Exposes the bootstrap ABI aliases, ABI version constant, and `pure_simdjson_get_abi_version`.
- `cbindgen.toml` - Captures the stable header-generation configuration for the public C surface.
- `include/pure_simdjson.h` - Committed generated header baseline for downstream review and diff checks.

## Decisions Made

- Used a minimal Rust crate as the source of truth for the ABI instead of hand-maintaining a bootstrap header.
- Kept the exported surface to the ABI version handshake only so Plan 02 can add the rest of the contract without undoing bootstrap scaffolding.
- Added a null-pointer guard to the out-param write so the first exported function does not immediately introduce UB at the C boundary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added a null-pointer guard on the bootstrap ABI export**
- **Found during:** Task 1 (Install `cbindgen` and scaffold the ABI-source crate)
- **Issue:** The plan’s literal stub would dereference `out_version` unconditionally and could trigger UB when called from C with a null pointer.
- **Fix:** Returned `-1` on null before writing the ABI version and kept the success path unchanged.
- **Files modified:** `src/lib.rs`
- **Verification:** `cargo check`
- **Committed in:** `e10300c` (part of Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The deviation tightened FFI safety without expanding scope or changing the bootstrap contract goal.

## Issues Encountered

- `cargo check` and `cbindgen` both generated an incidental `Cargo.lock`; it was removed before committing so the task commits stayed limited to the plan-owned ABI files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02 can now extend `src/lib.rs` with the fuller ABI surface and regenerate `include/pure_simdjson.h` from an existing pipeline.
- The repository has a concrete header round-trip check path, so future ABI edits can be validated mechanically instead of by prose review alone.

## Self-Check: PASSED

- Verified created files exist: `Cargo.toml`, `src/lib.rs`, `cbindgen.toml`, `include/pure_simdjson.h`, `.planning/phases/01-ffi-contract-design/01-01-SUMMARY.md`
- Verified task commits exist in git history: `e10300c`, `c5b3bc0`

---
*Phase: 01-ffi-contract-design*
*Completed: 2026-04-14*
