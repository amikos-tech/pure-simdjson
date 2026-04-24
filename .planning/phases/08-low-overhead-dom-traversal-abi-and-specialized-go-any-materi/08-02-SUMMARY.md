---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
plan: 02
subsystem: ffi
tags: [rust, c++, simdjson, ffi, materializer, cbindgen]

requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: Wave 0 header guards, internal-symbol audit, and fast-materializer test scaffold
provides:
  - Doc-owned native frame-stream traversal for root and subtree views
  - Internal Rust exports for frame-span build and reentrancy-guard seam
  - cbindgen exclusions that keep Phase 8 traversal symbols out of the public header
  - Rust integration coverage for root, subtree, oversized-literal rejection, and reentrancy
affects: [phase-08-plan-03, phase-08-plan-04, phase-09, abi-validation]

tech-stack:
  added: []
  patterns:
    - Doc-owned native frame scratch with RAII reentrancy guard
    - Internal-only ffi_wrap exports excluded from cbindgen-generated headers

key-files:
  created:
    - tests/rust_shim_fast_materializer.rs
  modified:
    - src/lib.rs
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - cbindgen.toml

key-decisions:
  - "Oversized integer literals normalize to parse-time PURE_SIMDJSON_ERR_INVALID_JSON at the parser boundary, and the internal materializer never emits BIGINT frames."
  - "Internal frame spans live in doc-owned native scratch guarded by materialize_in_progress so nested builds fail deterministically with PURE_SIMDJSON_ERR_PARSER_BUSY."

patterns-established:
  - "Internal traversal ABI: validate ValueView once in the Rust registry, then call native traversal with doc pointer plus json index."
  - "Public-header hygiene: add internal Rust export names to cbindgen exclude so verify-contract proves no psdj_internal_ or psimdjson_ symbols leak."

requirements-completed: [D-01, D-02, D-03, D-04, D-05, D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13, D-14]

duration: 12 min
completed: 2026-04-23
---

# Phase 08 Plan 02: Native Frame Stream Summary

**Internal doc-owned frame-stream traversal with Rust-only exports, reentrancy guard seam, and root/subtree Rust integration tests**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-23T17:17:27Z
- **Completed:** 2026-04-23T17:29:57Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added a private `psdj_internal_frame_t` ABI and native `psimdjson_materialize_build` path that walks a root or descendant subtree once and returns preorder frames from doc-owned scratch.
- Reused `with_resolved_view` in the Rust registry so stale handles, forged descendants, closed docs, reserved bits, and unknown tags are rejected before native traversal begins.
- Exported internal Rust entrypoints for frame-span build and the reentrancy test seam while keeping `include/pure_simdjson.h` free of every new internal symbol.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: native frame-stream tests** - `998b82b` (test)
2. **Task 1 GREEN: native frame builder** - `5faf6fb` (feat)
3. **Task 2 RED: internal export tests** - `a4bb287` (test)
4. **Task 2 GREEN: internal exports and header exclusions** - `e6942dc` (feat)

**Plan metadata:** committed separately after summary/state/roadmap updates.

## Files Created/Modified

- `tests/rust_shim_fast_materializer.rs` - Internal export coverage for root frames, Subtree frames, closed-doc rejection, oversized-literal parse rejection, and reentrant guard behavior.
- `src/native/simdjson_bridge.h` - Private frame struct and native traversal/test-seam declarations.
- `src/native/simdjson_bridge.cpp` - Doc-owned frame scratch, RAII build guard, traversal builder, parse-time BIGINT normalization, and guard-hold seam.
- `src/runtime/mod.rs` - Rust mirror of `psdj_internal_frame_t` plus wrappers for native traversal and guard-hold calls.
- `src/runtime/registry.rs` - One-shot view validation path into native traversal plus executable reentrancy seam plumbing.
- `src/lib.rs` - `ffi_wrap` internal exports for `psdj_internal_materialize_build` and `psdj_internal_test_hold_materialize_guard`.
- `cbindgen.toml` - Explicit exclusions for internal frame/export names so the generated public header stays unchanged.

## Verification

All plan-level checks passed:

```sh
cargo test -- --test-threads=1
make verify-contract
```

Acceptance criteria were also checked directly with `rg` against the expected private frame type, mutable doc-pointer propagation, guard symbols, pointer-stability comment, oversized-literal fixture, and header-exclusion entries.

## Decisions Made

- The parser boundary now maps `simdjson::BIGINT_ERROR` to `PURE_SIMDJSON_ERR_INVALID_JSON` during parse so oversized integer literals never produce a `ValueView` or a partial frame span.
- The native traversal builder reserves container capacity only when simdjson’s immediate child count is not saturated, then patches the exact `child_count` after the walk.
- The reentrancy seam is native-first: `psimdjson_test_hold_materialize_guard` acquires the same guard used by traversal and attempts a nested build to prove `PURE_SIMDJSON_ERR_PARSER_BUSY`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Normalized nested oversized integers to parse-time INVALID_JSON**
- **Found during:** Task 1 (Build the private native frame stream from validated DOM views)
- **Issue:** Parsing `{"ok":1,"big":99999999999999999999999}` returned `PURE_SIMDJSON_ERR_PRECISION_LOSS`, which contradicted the Phase 8 contract that oversized integer literals are rejected at parse time before any materialization call exists.
- **Fix:** Added a parse-specific error mapper in `psimdjson_parser_parse` that converts `simdjson::BIGINT_ERROR` to `PURE_SIMDJSON_ERR_INVALID_JSON` while leaving traversal/accessor-side mappings unchanged.
- **Files modified:** `src/native/simdjson_bridge.cpp`
- **Verification:** `cargo test -- --test-threads=1` and the new `oversized_literal_parse_rejected_before_materialize` integration test pass.
- **Committed in:** `5faf6fb`

---

**Total deviations:** 1 auto-fixed (1 Rule 1)
**Impact on plan:** The fix tightened behavior to match the planned contract without widening the public ABI or changing the Phase 8 scope.

## Issues Encountered

- Running targeted `cargo test` commands in parallel briefly contended on Cargo’s build-directory lock. Re-running the checks serially resolved it with no code changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan `08-03` can bind the internal frame span into Go and replace the current accessor-shaped materializer without reworking the native contract. The public header remains stable, and the root/subtree/reentrancy fixtures are already in place for the next wave.

## Self-Check: PASSED

- The summary file and all key implementation files exist on disk.
- Task commits `998b82b`, `5faf6fb`, `a4bb287`, and `e6942dc` are reachable in git history.
- Stub scan across the 08-02 files found no TODO/FIXME/placeholder markers that would block the plan goal.

---
*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi*
*Completed: 2026-04-23*
