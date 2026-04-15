---
phase: 01-ffi-contract-design
plan: "02"
subsystem: api
tags: [rust, ffi, abi, cbindgen, header-generation]
requires:
  - phase: "01-01"
    provides: "Rust ABI-source crate bootstrap plus cbindgen header pipeline"
provides:
  - Full contract-only Phase 1 ABI surface in Rust with stable handles, views, iterators, and error codes
  - Generated C header that exports the finalized ABI symbols, enums, and struct layouts
  - Public parser-busy, diagnostics, string-copy, and ABI-version contract comments encoded into source and header
affects: [phase-1-plan-03, phase-2-shim-implementation, go-bindings, header-regeneration]
tech-stack:
  added: []
  patterns: [contract-only-abi-source, exported-ffi-enums-and-structs, generated-header-as-authority]
key-files:
  created: []
  modified: [src/lib.rs, cbindgen.toml, include/pure_simdjson.h]
key-decisions:
  - "Keep every public export on the lowest-common-denominator ABI shape: int32_t returns with pointer out-params only."
  - "Represent Parser and Doc as packed u64 handles while Element and iterator state cross the ABI as doc-tied view structs."
  - "Export standalone enums and handle/view structs through cbindgen so the committed header fully captures the contract surface."
patterns-established:
  - "Contract comments live beside the Rust ABI items and flow into the generated header."
  - "Standalone ABI enums and structs must be explicitly exported through cbindgen, not left implicit through function reachability."
requirements-completed: [FFI-02, FFI-03, FFI-04, FFI-07, FFI-08]
duration: 9m
completed: 2026-04-14
---

# Phase 1 Plan 2: FFI Contract Design Summary

**Contract-only Rust ABI source with finalized handle/view layouts, explicit error space, and a generated pure_simdjson header for later shim work**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-14T11:58:00Z
- **Completed:** 2026-04-14T12:06:55Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Replaced the bootstrap ABI file with the full contract-only Phase 1 export set, including fixed error codes, value kinds, handles, views, iterators, metadata helpers, and stubbed entry points.
- Encoded the irreversible parser-busy, Rust-owned padded copy-in, diagnostics, and string-copy semantics directly in Rust doc comments so they flow into the public header.
- Regenerated `include/pure_simdjson.h` from Rust with the finalized ABI items exported explicitly, including the standalone enums and tagged structs needed by later phases.

## Task Commits

Each task was committed atomically:

1. **Task 1: Encode the stable ABI types, enums, and exported stub signatures in `src/lib.rs`** - `188b567` (`feat`)
2. **Task 2: Regenerate `include/pure_simdjson.h` from the finalized ABI source** - `254fd67` (`feat`)

## Files Created/Modified

- `src/lib.rs` - Defines the complete contract-only ABI surface, fixed metadata helpers, and lifecycle/ownership comments for later implementation phases.
- `cbindgen.toml` - Exports the standalone ABI enums and structs explicitly and uses generation settings that keep the header clean and tagged.
- `include/pure_simdjson.h` - Committed generated header that mirrors the finalized Rust ABI source byte-for-byte.

## Decisions Made

- Returned raw `i32` from Rust exports so cbindgen emits `int32_t` prototypes instead of a typedef alias, matching the portable ABI rule directly in the header.
- Kept `Parser` and `Doc` opaque while making element/iterator results lightweight doc-bound structs, which locks the Phase 1 ownership model without inventing future allocation APIs.
- Used explicit cbindgen export settings for the contract enums and layout structs because relying on function reachability alone omitted required standalone ABI items from the generated header.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- cbindgen initially omitted the standalone error/value enums and handle-parts struct because they were not referenced by function signatures alone; exporting them explicitly in `cbindgen.toml` resolved the gap within Task 2 scope.
- `cargo check` created transient `Cargo.lock` and `target/` artifacts; both were removed before task commits so the repository stayed limited to plan-owned files.

## Known Stubs

- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:154) - `pure_simdjson_parser_new`, `pure_simdjson_parser_free`, and `pure_simdjson_parser_parse` still return `PURE_SIMDJSON_ERR_INTERNAL`; this plan locks ABI shape only and defers parser behavior to later shim phases.
- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:188) - Parser diagnostic helpers (`pure_simdjson_parser_get_last_error_len`, `pure_simdjson_parser_copy_last_error`, `pure_simdjson_parser_get_last_error_offset`) remain compile-only stubs by plan design.
- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:220) - Document lifecycle and root lookup (`pure_simdjson_doc_free`, `pure_simdjson_doc_root`) are contract stubs pending actual parser/doc registry implementation.
- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:243) - Scalar and kind accessors (`pure_simdjson_element_type`, numeric getters, bool/null getters) remain stubbed until native value decoding lands.
- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:283) - String copy-out and `pure_simdjson_bytes_free` are exposed contractually but intentionally unimplemented in this plan.
- [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:324) - Array/object iterator constructors, next-step helpers, and `pure_simdjson_object_get_field` remain compile-only stubs while preserving the finalized traversal ABI.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03 can add mechanical ABI verification and contract docs against a concrete Rust/header surface instead of draft signatures.
- Phase 2 can implement parser/doc registries and native shims without renegotiating handles, value views, diagnostics, or traversal entry points.

## Self-Check: PASSED

- Verified summary file exists: `.planning/phases/01-ffi-contract-design/01-02-SUMMARY.md`
- Verified task commits exist in git history: `188b567`, `254fd67`

---
*Phase: 01-ffi-contract-design*
*Completed: 2026-04-14*
