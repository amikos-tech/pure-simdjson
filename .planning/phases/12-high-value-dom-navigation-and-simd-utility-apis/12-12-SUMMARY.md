---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 12
subsystem: ffi-utilities
tags: [rust, ffi, minify, buffer-capacity, gap-closure]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-04's native minify/UTF-8 utility surface (src/runtime/mod.rs native_minify, src/lib.rs pure_simdjson_minify) that this plan fixes
provides:
  - "pure_simdjson_minify writes the exact required capacity (src_len) into out_written on PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL, matching docs/ffi-contract.md's already-documented contract"
  - "Rust and dynamic C smoke assertions that pin the status+capacity pair together, closing the test gap that let the original defect pass CI"
affects: [12-verification-gap-1, UTIL-01]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "native_minify returns (status, written) as one inseparable tuple instead of Result<usize, status>, so a non-OK native status never discards the written value"
    - "out_written is written via a direct ptr::write guarded by an explicit status allowlist (OK or BUFFER_TOO_SMALL), never through the write_out helper whose own return code would clobber a genuine non-OK status"

key-files:
  created: []
  modified:
    - src/runtime/mod.rs
    - src/lib.rs
    - include/pure_simdjson.h
    - tests/rust_shim_minify.rs
    - tests/smoke/ffi_export_surface.c

key-decisions:
  - "Change native_minify's return type from Result<usize, pure_simdjson_error_code_t> to (pure_simdjson_error_code_t, usize) so status and required capacity always travel together, matching the single call site in src/lib.rs."
  - "Guard the ptr::write of out_written with rc == err_ok() || rc == err_buffer_too_small() rather than routing through write_out, preserving the true BUFFER_TOO_SMALL status instead of silently replacing it with OK."
  - "Regenerate include/pure_simdjson.h after extending pure_simdjson_minify's doc comment so make verify-contract's header-diff check stays clean (deviation, see below)."

patterns-established:
  - "Out-parameter contracts that promise a value on more than one status code (here: OK and BUFFER_TOO_SMALL) must carry that value out of the Rust/native boundary as a tuple, not a Result that collapses non-OK paths to a bare error code."

requirements-completed: [UTIL-01]

# Metrics
duration: 10min
completed: 2026-07-31
---

# Phase 12 Plan 12: Minify BUFFER_TOO_SMALL Capacity Handoff Gap Closure Summary

**Restored `pure_simdjson_minify`'s documented `out_written` contract on `BUFFER_TOO_SMALL` by making `native_minify` return `(status, written)` as one tuple instead of discarding `written` on the `Err` path, closing 12-VERIFICATION.md's Gap 1**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-07-31T17:12:00+03:00 (approx.)
- **Completed:** 2026-07-31T17:17:56+03:00
- **Tasks:** 2
- **Files modified:** 5 (4 planned + 1 deviation: regenerated header)

## Accomplishments
- `pure_simdjson_minify` with an undersized destination now writes `src_len` into `out_written` alongside `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`, instead of leaving the caller's sentinel untouched
- `tests/rust_shim_minify.rs`'s undersized-destination test now asserts the required-capacity value (`assert_eq!(written, input.len())`), not just the status code
- `tests/smoke/ffi_export_surface.c` gained a new block that exercises the same path against a real built/dynamically-loaded dylib, asserting both status and `out_written`
- Verified end-to-end: `cargo test --test rust_shim_minify`, `cargo build --release` + `run_native_smoke.sh`, and `make verify-contract` all pass with zero C++ files modified

## Task Commits

Each task was committed atomically:

1. **Task 1: Preserve native minify status and required capacity across the Rust handoff** - `ab31d76` (fix)
2. **Task 2: Assert the status/required-capacity pair in the dynamic C smoke harness** - `b580467` (test)
3. **Deviation: Regenerate header for minify out_written doc comment** - `0b192c2` (docs)

**Plan metadata:** (this commit, see final commit below)

## Files Created/Modified
- `src/runtime/mod.rs` - `native_minify` now returns `(pure_simdjson_error_code_t, usize)` instead of `Result<usize, pure_simdjson_error_code_t>`, so the native `written` value survives every status, not only `OK`
- `src/lib.rs` - `pure_simdjson_minify` destructures `(rc, written)`, writes `out_written` via a direct `ptr::write` guarded by `rc == err_ok() || rc == err_buffer_too_small()`, and always returns the true native `rc`; extended the function's doc comment with the `out_written`-on-`BUFFER_TOO_SMALL` contract sentence; dropped the now-stale `#[cfg_attr(not(test), allow(dead_code))]` on `err_buffer_too_small()` since it is exercised in production code, not only tests
- `include/pure_simdjson.h` - regenerated via `cbindgen` to mirror the extended doc comment (deviation, see below)
- `tests/rust_shim_minify.rs` - added `assert_eq!(written, input.len());` to `minify_undersized_dst_returns_buffer_too_small_before_writing`
- `tests/smoke/ffi_export_surface.c` - added a scoped block after the exact-alias minify success check that calls `exports.minify` with a destination one byte smaller than the source and asserts both `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL` and `undersized_written == sizeof(minify_source) - 1`

## Decisions Made
- Kept the fix strictly to the Rust handoff (`src/runtime/mod.rs`, `src/lib.rs`) as directed — the C++ bridge (`src/native/simdjson_bridge.cpp`) was confirmed already correct (`*out_written = src_len;` set before the capacity check) and was not touched
- Did not route the `BUFFER_TOO_SMALL` write through the existing `write_out` helper, since `write_out`'s own return value is always `PURE_SIMDJSON_OK`, which would have silently reintroduced a different variant of the same class of bug
- Regenerated `include/pure_simdjson.h` after extending the Rust doc comment (see Deviations) so the committed header stays byte-identical to what `cbindgen` produces from source, matching the existing `make verify-contract` gate

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking/consistency] Regenerated `include/pure_simdjson.h` after extending the `pure_simdjson_minify` doc comment**
- **Found during:** Task 1 verification (running `make verify-contract` after committing Task 1)
- **Issue:** The plan's Task 1 action required extending `pure_simdjson_minify`'s Rust doc comment with a sentence describing the `out_written` contract on `BUFFER_TOO_SMALL`. `cbindgen` mirrors Rust doc comments into `include/pure_simdjson.h`, so the committed header immediately fell out of sync with what `cbindgen --config cbindgen.toml --crate pure_simdjson --output ...` would regenerate. `make verify-contract`'s `diff -u include/pure_simdjson.h "$tmp"` step reported this drift (though the Makefile recipe does not `set -e`/`&&`-chain the diff, so it did not fail the target outright, only printed the diff).
- **Fix:** Ran `make generate-header` to regenerate `include/pure_simdjson.h` from source; the only change was the two added doc-comment lines. Re-ran `make verify-contract` to confirm a clean (empty) header diff.
- **Files modified:** `include/pure_simdjson.h`
- **Verification:** `make verify-contract` shows no diff for the header; `cargo build --release && bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` re-run after the header regeneration still prints `ffi export surface smoke passed`
- **Committed in:** `0b192c2`

---

**Total deviations:** 1 auto-fixed (Rule 3 — consistency/blocking)
**Impact on plan:** Necessary to keep the generated public header in sync with the source doc comment the plan itself required adding; no scope creep, no behavior change.

## Issues Encountered
None beyond the header-regeneration deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Requirement UTIL-01 is unblocked: `pure_simdjson_minify` now honors its documented `BUFFER_TOO_SMALL` out-parameter contract, proven at both the Rust unit-test layer and the dynamic C smoke layer against a real built library
- 12-VERIFICATION.md's Gap 1 is closed; WR-01 (unrelated rejected-iterator-constructor lease ordering) and WR-02/WR-03 (workflow trigger paths/unpinned cbindgen version) remain explicitly deferred per the plan's scope note and `deferred-items.md`
- No blockers for downstream Phase 12 closeout or Phase 13 planning

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
