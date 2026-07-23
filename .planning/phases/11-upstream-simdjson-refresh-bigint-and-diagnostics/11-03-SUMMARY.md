---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 03
subsystem: native-abi
tags: [simdjson, bigint, ffi, rust, cpp, ownership]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-02 native kind 9 classification and exact document-owned BigInt spans
provides:
  - Private C++ and Rust BigInt view seam backed by upstream `get_bigint()`
  - Strict panic-safe `pure_simdjson_element_get_bigint` C export
  - Exact copied BigInt bytes tracked by the existing allocation/free registry
  - Root, lookup, and iterator kind hint 9 propagation without precision-loss fallback
affects: [11-07, 11-09, 11-10, native-abi, go-dom]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Copy document-owned scalar spans before crossing the public ABI
    - Share one tracked byte allocation/free path across strings and BigInts

key-files:
  created:
    - tests/rust_shim_bigint.rs
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-03-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/runtime/registry.rs
    - src/lib.rs

key-decisions:
  - "Reuse one `element_get_bytes_copy` helper and tracked byte registry for string and BigInt copies; do not add an allocator or borrowed public pointer."
  - "Propagate successful native kind hints directly for roots and descendants, including raw kind 9."
  - "Keep precision-loss only for in-range int64/uint64-to-float64 conversion; BigInt numeric getters remain strict wrong-type operations."

patterns-established:
  - "Copied scalar ABI: validate the view once, borrow only inside the registry call, copy immediately, register the owned allocation, and release through `pure_simdjson_bytes_free`."
  - "Write-on-success outputs: public pointer/length outputs remain untouched for validation and wrong-type failures."

requirements-completed: [NUM-01, NUM-02]

# Metrics
duration: 20min
completed: 2026-07-23
---

# Phase 11 Plan 03: Copied BigInt ABI Summary

**Strict exact BigInt text now crosses the native C ABI as a tracked caller-owned copy that remains valid after document release**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-23T11:44:30Z
- **Completed:** 2026-07-23T12:05:29Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added a private C++ bridge helper that calls upstream `get_bigint()` and exposes its document-owned span only to the immediate Rust copy boundary.
- Added `pure_simdjson_element_get_bigint` with `ffi_wrap`, null-output validation, strict kind-9 behavior, and write-on-success pointer/length outputs.
- Reused the existing tracked byte allocation/free family, proving exact positive and negative text survives document free and rejects a second release.
- Removed the obsolete invalid-kind/precision-loss fallback so root, object lookup, and array iterator views retain native kind hint `9`.
- Corrected authoritative Rust API documentation so BigInt classification and strict numeric-getter behavior match D-05 before cbindgen synchronization.

## Task Commits

Each task was committed atomically:

1. **Task 1: Define the native BigInt view seam without exposing borrowed memory** - `2651b06` (feat)
2. **Task 2 RED: Add the failing copied BigInt boundary contract** - `6ab3f86` (test)
3. **Task 2 GREEN: Copy BigInt through the allocation registry and public export** - `055cb9c` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.h` - Declares the private resolved-index BigInt view helper.
- `src/native/simdjson_bridge.cpp` - Calls upstream `get_bigint()`, maps wrong types to status 4, and keeps C++ exceptions inside the existing boundary.
- `src/runtime/mod.rs` - Wraps the borrowed BigInt span for immediate registry consumption.
- `src/runtime/registry.rs` - Copies scalar spans into the shared byte allocation registry and preserves raw kind hints.
- `src/lib.rs` - Exposes the panic-safe copied BigInt C export and corrects root/type/numeric ownership documentation.
- `tests/rust_shim_bigint.rs` - Locks exact text, strict types, untouched errors, post-document lifetime, double-free rejection, and descendant kind hints.

## Decisions Made

- Strings and BigInts share the same private copy helper and allocation registry because their ownership contract is identical.
- BigInt bytes are never parsed or normalized in Rust; the exact upstream span is copied byte-for-byte.
- The generated public header remains unchanged in this plan. Plan 11-09 owns the single complete ABI 1.2 cbindgen synchronization.

## TDD Gate Compliance

- **RED:** `6ab3f86` added the integration contract and failed solely because `pure_simdjson_element_get_bigint` did not yet exist.
- **GREEN:** `055cb9c` added the minimal registry/export implementation and passed all six BigInt integration tests.
- No refactor commit was needed; the shared copy helper landed as part of the minimal implementation.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- `cargo test --locked --test rust_shim_bigint -- --test-threads=1` — 6/6 passed.
- `cargo test --locked --lib -- --test-threads=1` — 15/15 passed.
- `cargo test --locked --test rust_shim_accessors --test rust_shim_iterators --test rust_shim_minimal -- --test-threads=1` — 36/36 passed.
- Registry and documentation guard searches — no obsolete `KIND_HINT_INVALID`, `err_precision_loss`, BigInt-unclassifiable, or BigInt-precision-loss contract remains.
- `cargo tree --locked --depth 1` — unchanged dependency graph; no package manifest or lockfile changed.
- `cargo build --release --locked` — passed against the Plan 11-02 provenance-locked v4.6.4 build-output patch.
- Fresh release dylib inspection — contains `pure_simdjson_element_get_bigint`.
- `PURE_SIMDJSON_LIB_PATH=<fresh target/release dylib> go test ./... -race` — all four Go packages passed.
- `make verify-contract` was intentionally not run because Plan 11-09 owns the generated header synchronization; no Rust, Rust-test, build, or Go failure was accepted.

## Threat and Security Impact

- **T-11-04 mitigated:** the C++ span never crosses the public ABI; Rust copies it before returning, tracks the owned allocation, and proves post-document lifetime plus exactly-once release.
- **T-11-SC preserved:** no dependency, package manifest, lockfile, allocator family, frame field, or hand-edited generated header was added.
- No unplanned network endpoint, authentication path, schema boundary, or file-access surface was introduced.

## Known Stubs

None. The generated C header synchronization is intentional dependency-ordered Plan 11-09 work, not a placeholder implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-07 can synchronize public ABI kind mirrors around native kind `9`.
- Plan 11-09 can emit this export into the complete ABI 1.2 generated header and contract audit.
- Plan 11-10 can bind `Element.GetBigInt()` in Go to the exact copied C export.
- No blockers remain for subsequent Phase 11 plans.

## Self-Check: PASSED

- All six implementation/test artifacts exist.
- Task commits `2651b06`, `6ab3f86`, and `055cb9c` are present in repository history.
- The plan changed neither `Cargo.toml`, `Cargo.lock`, nor `include/pure_simdjson.h`.
- The only remaining dirty paths outside this summary are the two pre-existing unrelated planning files.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
