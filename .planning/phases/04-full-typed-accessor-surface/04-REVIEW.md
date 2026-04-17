---
phase: 04-full-typed-accessor-surface
reviewed: 2026-04-17T09:58:46Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - cbindgen.toml
  - element.go
  - element_fuzz_test.go
  - element_scalar_test.go
  - example_test.go
  - include/pure_simdjson.h
  - internal/ffi/bindings.go
  - internal/ffi/types.go
  - iterator.go
  - iterator_test.go
  - purejson.go
  - src/lib.rs
  - src/native/simdjson_bridge.cpp
  - src/native/simdjson_bridge.h
  - src/runtime/mod.rs
  - src/runtime/registry.rs
  - tests/rust_shim_accessors.rs
  - tests/rust_shim_iterators.rs
findings:
  critical: 0
  warning: 2
  info: 0
  total: 2
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-04-17T09:58:46Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

I reviewed the full typed-accessor and iterator surface across the Go wrapper, Rust shim/runtime, regenerated header, and C++ bridge. The accessor paths are internally consistent and the shipped tests currently pass (`go test ./...` and `cargo test --tests`), but there are two correctness risks left in the new ABI surface:

1. `pure_simdjson_array_iter_t` is still only partially authenticated, so a forged in-range iterator state can return `PURE_SIMDJSON_OK` instead of failing as an invalid handle.
2. The descendant reconstruction helper in the C++ bridge writes a `tape_ref` through a `reinterpret_cast` to `dom::element`, which relies on undefined aliasing behavior.

## Warnings

### WR-01: Public Array Iterator State Is Not Fully Authenticated

**File:** `src/runtime/registry.rs:695-783`
**Issue:** `with_iter_doc` only validates `doc`, `tag`, `reserved`, and that `state0/state1` are in-range. `array_iter_next` then trusts `state0` as a valid element boundary and immediately calls `encode_descendant_view_locked(entry, iter.doc, iter.state0)`. Because `pure_simdjson_array_iter_t` is a public ABI struct (`include/pure_simdjson.h:95-109`), callers can mutate `state0` while keeping it in range. I validated this with a small harness against the built Rust shim: on `[1,2,3]`, changing `state0` to another in-range value made `pure_simdjson_array_iter_next` return `PURE_SIMDJSON_OK` and hand back a bogus view, instead of rejecting the iterator as `PURE_SIMDJSON_ERR_INVALID_HANDLE`. That weakens the handle-hardening guarantees added elsewhere in this phase and makes accidental or malicious iterator corruption observable as silent misbehavior rather than a deterministic handle failure.
**Fix:** Authenticate iterator state instead of accepting any in-range `state0`. Since the ABI shape is already fixed, store an implementation-owned cookie in one of the existing iterator fields (for example `index`) derived from `(doc, state0, state1, tag)`, recompute it in `with_iter_doc`, and reject mismatches before calling `encode_descendant_view_locked`. An alternative is keeping a per-document registry of issued iterator positions and validating `(state0, state1)` against that registry on every `*_iter_next` call.

### WR-02: `element_at` Reconstructs `dom::element` Via Undefined Aliasing

**File:** `src/native/simdjson_bridge.cpp:172-185`
**Issue:** `element_at` creates a `simdjson::dom::element`, reinterprets its storage as `simdjson::internal::tape_ref*`, and writes through that pointer. Even with the size checks already in place, this is still an aliasing violation between unrelated types, so the compiler is free to miscompile it under optimization. `tape_ref_of` a few lines below already uses `memcpy`; `element_at` should use the same representation-copy approach instead of a typed write through `reinterpret_cast`.
**Fix:**
```cpp
static_assert(std::is_trivially_copyable_v<simdjson::dom::element>);
static_assert(sizeof(simdjson::dom::element) == sizeof(simdjson::internal::tape_ref));

const auto tape = simdjson::internal::tape_ref(&doc->document, size_t(json_index));
simdjson::dom::element element;
std::memcpy(&element, &tape, sizeof(element));
return element;
```

---

_Reviewed: 2026-04-17T09:58:46Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
