---
phase: 04-full-typed-accessor-surface
plan: 01
subsystem: api
tags: [ffi, purego, rust, simdjson, abi]
requires:
  - phase: 02-rust-shim-minimal-parse-path
    provides: parser/doc handle registry, root view transport, and minimal Rust shim exports
  - phase: 03-go-public-api-purego-happy-path
    provides: purego loader/binding scaffolding and the public Element wrapper skeleton
provides:
  - descendant-safe value-view encoding with locked PSDJROOT and PSDJDESC tags
  - real uint64, float64, string, bool, null, and bytes_free ABI exports
  - purego wrappers for the new scalar/string/bool/null helpers with defer-safe string cleanup
  - Rust ABI tests covering string ownership, wrong-type handling, and invalid descendant handles
affects: [04-02, 04-03, 04-04, typed-accessors, iterators]
tech-stack:
  added: []
  patterns: [doc-plus-json-index descendant views, Rust-owned string copy-out, defer-safe purego string cleanup]
key-files:
  created: [tests/rust_shim_accessors.rs]
  modified: [src/native/simdjson_bridge.h, src/native/simdjson_bridge.cpp, src/runtime/mod.rs, src/runtime/registry.rs, src/lib.rs, internal/ffi/bindings.go]
key-decisions:
  - "Lock descendant views to state0=json_index plus state1=PSDJDESC while root views keep the existing root-pointer transport."
  - "Keep string copy-out ownership entirely in Rust by copying borrowed doc bytes into an exact-capacity Vec and freeing through pure_simdjson_bytes_free."
  - "Use a direct defer b.BytesFree(ptr, len) cleanup path in the hidden purego string wrapper so success-path frees survive later conversion work."
patterns-established:
  - "Descendant validation is registry-backed: only json_index values emitted by encode_descendant_view are accepted later."
  - "Native scalar/string/bool/null access flows through doc+json_index helpers instead of raw descendant element pointers."
requirements-completed: [API-04, API-05, API-06]
duration: 16m
completed: 2026-04-17
---

# Phase 4 Plan 01: Native Typed Accessor Substrate Summary

**Descendant-safe scalar/string/bool/null ABI with Rust-owned string copy-out and purego cleanup wrappers**

## Performance

- **Duration:** 16m
- **Started:** 2026-04-17T07:58:23Z
- **Completed:** 2026-04-17T08:14:30Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Locked the Phase 4 value-view transport to `PSDJROOT` for roots and `PSDJDESC` for descendants, with one registry validation path for both.
- Replaced the Phase 4 scalar/string/bool/null Rust ABI stubs with real exports, including Rust-owned string allocation and `pure_simdjson_bytes_free`.
- Extended the hidden purego layer and added ABI-facing Rust tests for string ownership, empty-string handling, wrong-type semantics, and invalid descendant handles.

## Task Commits

Each task was committed atomically:

1. **Task 1: Lock descendant-view encoding and implement native scalar/string/bool/null exports** - `484c09f` (`feat`)
2. **Task 2: Expand the hidden purego layer with defer-safe string cleanup and native ownership tests** - `421e706` (`feat`)

## Files Created/Modified
- `src/native/simdjson_bridge.h` - Added hidden `doc + json_index` bridge helpers for descendant type/scalar/string/bool/null reads.
- `src/native/simdjson_bridge.cpp` - Reconstructed descendant elements from the DOM tape and implemented the new hidden accessor helpers.
- `src/runtime/mod.rs` - Declared `DESC_VIEW_TAG` and the new native wrappers that operate on `psimdjson_doc + json_index`.
- `src/runtime/registry.rs` - Added descendant encoding/validation, Rust-owned string copy-out, `bytes_free`, and the new scalar/string/bool/null registry entry points.
- `src/lib.rs` - Activated the public Phase 4 exports for `uint64`, `float64`, `string`, `bytes_free`, `bool`, and `is_null`.
- `internal/ffi/bindings.go` - Bound the new symbols and added typed purego wrappers, including defer-safe string cleanup.
- `tests/rust_shim_accessors.rs` - Added ABI-facing integration coverage for ownership, wrong-type behavior, and invalid descendant handles.

## Decisions Made

- Locked descendant transport to `doc + json_index` with `PSDJDESC` instead of inventing per-call shadow state.
- Kept string allocation symmetry entirely on the Rust side to avoid cross-runtime allocator assumptions.
- Returned empty strings as the explicit `ptr == nil && len == 0` sentinel so the free path has one no-op case.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added registry-backed descendant index validation**
- **Found during:** Task 1 (Lock descendant-view encoding and implement native scalar/string/bool/null exports)
- **Issue:** A raw `json_index` transport alone was not sufficient to reject forged descendant views before touching the DOM tape.
- **Fix:** Tracked emitted descendant indices per live document and rejected unknown `PSDJDESC` views as `PURE_SIMDJSON_ERR_INVALID_HANDLE`.
- **Files modified:** `src/runtime/registry.rs`
- **Verification:** `cargo test --lib`; `cargo test --test rust_shim_accessors && cargo build --release && go test ./...`; `descendant_tag_and_reserved_bits_return_invalid_handle`
- **Committed in:** `484c09f`

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The extra validation was necessary for handle safety and did not expand the plan scope beyond the locked descendant transport model.

## Issues Encountered

- The single-header simdjson build keeps `dom::element(tape_ref)` private. The bridge now reconstructs descendant elements through the internal `tape_ref` layout instead of relying on that private constructor.

## User Setup Required

None - no external service configuration required.

## Known Stubs

- `src/lib.rs:687` - `pure_simdjson_array_iter_new` remains a contract-only stub for Plan 04-03.
- `src/lib.rs:707` - `pure_simdjson_array_iter_next` remains a contract-only stub for Plan 04-03.
- `src/lib.rs:726` - `pure_simdjson_object_iter_new` remains a contract-only stub for Plan 04-03.
- `src/lib.rs:747` - `pure_simdjson_object_iter_next` remains a contract-only stub for Plan 04-03.
- `src/lib.rs:769` - `pure_simdjson_object_get_field` remains a contract-only stub for Plan 04-04.

## Next Phase Readiness

- The public Go accessor work in Plan 04-02 can now bind to real native uint64/float64/string/bool/null behavior instead of stubs.
- The iterator and object-field plans can build on one fixed descendant-view representation and the established `bytes_free` ownership contract.

## Self-Check: PASSED

---
*Phase: 04-full-typed-accessor-surface*
*Completed: 2026-04-17*
