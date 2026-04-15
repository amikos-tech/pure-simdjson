---
phase: 02-rust-shim-minimal-parse-path
reviewed: 2026-04-15T13:52:04Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - .github/workflows/phase2-rust-shim-smoke.yml
  - .gitignore
  - .gitmodules
  - Cargo.toml
  - Makefile
  - build.rs
  - cbindgen.toml
  - include/pure_simdjson.h
  - src/lib.rs
  - src/native/simdjson_bridge.cpp
  - src/native/simdjson_bridge.h
  - src/runtime/mod.rs
  - src/runtime/registry.rs
  - tests/rust_shim_fallback_gate.rs
  - tests/rust_shim_minimal.rs
  - tests/smoke/README.md
  - tests/smoke/minimal_parse.c
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-04-15T13:52:04Z
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found

## Summary

The Phase 2 minimal parse path is largely in good shape: `cargo test`, `cargo build --release`, `make phase2-smoke-linux`, and `make verify-contract` all pass. The remaining issues are contract-level mismatches in the native bridge and public API surface: one wrong error-code mapping, one exported diagnostic function that never returns real data, and stale public docs that still describe already-live exports as stub-only.

## Warnings

### WR-01: Unsupported CPU parse failures are collapsed into `ERR_INTERNAL`

**File:** `src/native/simdjson_bridge.cpp:56-76`
**Issue:** The bridge maps `simdjson::UNSUPPORTED_ARCHITECTURE` to `PURE_SIMDJSON_ERR_INTERNAL` even though the public ABI reserves `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` for this exact condition. That means callers can correctly get `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` from the Rust-side fallback gate in `pure_simdjson_parser_new`, but receive a generic internal failure if simdjson reports the same condition during parse. This is a user-visible contract mismatch, and the current tests only cover the `parser_new` path (`tests/rust_shim_fallback_gate.rs:17-45`), so the parse path regression would not be caught.
**Fix:**
```cpp
pure_simdjson_error_code_t map_error(simdjson::error_code error) noexcept {
  switch (error) {
    case simdjson::UNSUPPORTED_ARCHITECTURE:
      return PURE_SIMDJSON_ERR_CPU_UNSUPPORTED;
    // keep the existing mappings for the other cases
    default:
      ...
  }
}
```
Add a regression test that forces or injects the unsupported-architecture parse path, not just the `parser_new` fallback gate.

### WR-02: `parser_get_last_error_offset` never reports a real offset

**File:** `src/native/simdjson_bridge.cpp:17-20`, `src/native/simdjson_bridge.cpp:104-112`, `src/native/simdjson_bridge.cpp:241-250`, `tests/rust_shim_minimal.rs:153-166`
**Issue:** `psimdjson_parser::last_error_offset` is initialized to `UINT64_MAX`, reset to `UINT64_MAX`, and set to `UINT64_MAX` on every parse error. The getter simply returns that field, so `pure_simdjson_parser_get_last_error_offset` can never return a real byte offset for malformed JSON. The integration test currently hard-codes that sentinel as the expected value, which locks in the missing behavior instead of catching it.
**Fix:** Capture the parser error position from simdjson when `parse_into_document` fails, store that byte index in `last_error_offset`, and update the failing-parse tests to assert a concrete offset for inputs like `b"{"` or `"{\"x\":}"`. If Phase 2 intentionally does not support offsets yet, document the `UINT64_MAX` sentinel explicitly in the public header and smoke README instead of exposing the function as though it were fully implemented.

## Info

### IN-01: Public comments still describe live Phase 2 exports as stub-only

**File:** `include/pure_simdjson.h:173-317`, `src/lib.rs:308-539`
**Issue:** The header and Rust export docs still say that `parser_new`, `parser_free`, `parser_parse`, `doc_free`, `doc_root`, `element_type`, and `element_get_int64` are "contract-only stub[s]" that return `PURE_SIMDJSON_ERR_INTERNAL`. Those functions are now implemented and covered by the smoke harness and integration tests. Leaving the old wording in place will mislead C consumers and any generated documentation.
**Fix:** Remove the stub disclaimer from the live Phase 2 exports and keep it only on the still-unimplemented entry points. A small doc check that matches the smoke-covered export list against the header comments would stop this from drifting again.

---

_Reviewed: 2026-04-15T13:52:04Z_  
_Reviewer: Claude (gsd-code-reviewer)_  
_Depth: standard_
