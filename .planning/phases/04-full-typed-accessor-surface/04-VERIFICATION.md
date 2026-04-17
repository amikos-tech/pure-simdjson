---
phase: 04-full-typed-accessor-surface
verified: 2026-04-17T10:14:22Z
status: passed
score: 10/10 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "9/10"
  gaps_closed:
    - "DOC-03 is closed package-wide for the final v0.1 surface"
  gaps_remaining: []
  regressions: []
---

# Phase 4: Full Typed Accessor Surface Verification Report

**Phase Goal:** Complete the DOM accessor surface so consumers can extract every JSON value type and walk arrays and objects via cursor-pull iteration. This is the v0.1 API users will see.
**Verified:** 2026-04-17T10:14:22Z
**Status:** passed
**Re-verification:** Yes — after gap closure

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Root and descendant values use one locked transport encoding, and string copy-out ownership stays on the Rust side. | ✓ VERIFIED | [src/runtime/mod.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/mod.rs:15) defines `ROOT_VIEW_TAG`, `DESC_VIEW_TAG`, `ARRAY_ITER_TAG`, and `OBJECT_ITER_TAG`; [src/runtime/registry.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/registry.rs:503) resolves views through `with_resolved_view(...)`, emits descendants through `encode_descendant_view_locked(...)`, copies strings in `element_get_string(...)`, and frees them in `bytes_free(...)`. |
| 2 | The hidden Go FFI layer exposes typed scalar/string/bool/null and iterator wrappers with `runtime.KeepAlive(...)` and defer-safe string cleanup. | ✓ VERIFIED | [internal/ffi/bindings.go](/Users/tazarov/experiments/amikos/pure-simdjson/internal/ffi/bindings.go:225) binds `ElementGetUint64`, `ElementGetFloat64`, `ElementGetString`, `BytesFree`, `ElementGetBool`, `ElementIsNull`, `ArrayIter*`, `ObjectIter*`, and `ObjectGetField`; `ElementGetString(...)` installs `defer b.BytesFree(ptr, length)` at [internal/ffi/bindings.go](/Users/tazarov/experiments/amikos/pure-simdjson/internal/ffi/bindings.go:241). |
| 3 | Every JSON scalar value type is extractable through the public Go `Element` API with precise type classification. | ✓ VERIFIED | [element.go](/Users/tazarov/experiments/amikos/pure-simdjson/element.go:17) exports `Type()`, `GetInt64()`, `GetUint64()`, `GetFloat64()`, `GetString()`, `GetBool()`, and `IsNull()`; [element_scalar_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/element_scalar_test.go:34) covers int64, uint64, float64, string, bool, null, array, and object classification. |
| 4 | Closed or invalid views behave explicitly: getters return typed errors, while `Type()` and `IsNull()` collapse to sentinel values. | ✓ VERIFIED | [element.go](/Users/tazarov/experiments/amikos/pure-simdjson/element.go:52) routes through `usableDoc()`; `Type()` returns `TypeInvalid` on non-OK FFI status and `IsNull()` returns `false`; [element_scalar_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/element_scalar_test.go:81) covers closed docs, zero-value roots, and tampered descendant tags/reserved bits. |
| 5 | Numeric edge behavior matches the phase contract exactly. | ✓ VERIFIED | [element_scalar_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/element_scalar_test.go:168) verifies `GetInt64(9223372036854775808) -> ErrNumberOutOfRange`, `GetInt64(1e20) -> ErrWrongType`, and `GetFloat64(9007199254740993) -> ErrPrecisionLoss`; the targeted Go spot-check passed. |
| 6 | Arrays and objects can be traversed in document order through scanner-style cursor-pull iteration without Go callbacks across FFI. | ✓ VERIFIED | [iterator.go](/Users/tazarov/experiments/amikos/pure-simdjson/iterator.go:9) exposes `Next()`, `Value()`, `Key()`, and `Err()`; [iterator_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/iterator_test.go:10) and [tests/rust_shim_iterators.rs](/Users/tazarov/experiments/amikos/pure-simdjson/tests/rust_shim_iterators.rs:1) verify order and terminal behavior; no callback path is present in the iterator/accessor surface. |
| 7 | Iterator transport is inline-only and uses the document plus transport struct rather than native heap iterator ownership. | ✓ VERIFIED | [src/runtime/registry.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/registry.rs:722) stores iterator progress in `pure_simdjson_array_iter_t` / `pure_simdjson_object_iter_t` (`state0`, `state1`, `index`, `tag`, `reserved`) and advances them in place; the iterator creation/advance path does not allocate native iterator heap state. |
| 8 | `Object.GetField` and `GetStringField` distinguish missing fields from present `null` values, and the string helper is Go composition rather than a new ABI call. | ✓ VERIFIED | [element.go](/Users/tazarov/experiments/amikos/pure-simdjson/element.go:303) implements `GetField(...)` over `bindings.ObjectGetField(...)` and `GetStringField(...)` as `GetField(...)` plus `GetString()` at [element.go](/Users/tazarov/experiments/amikos/pure-simdjson/element.go:324); [iterator_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/iterator_test.go:165) verifies missing-vs-null and helper behavior. |
| 9 | Malformed UTF-8 is rejected as `ErrInvalidJSON` instead of producing corrupted Go strings. | ✓ VERIFIED | [element_scalar_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/element_scalar_test.go:215), [iterator_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/iterator_test.go:286), and [element_fuzz_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/element_fuzz_test.go:9) all reject invalid UTF-8 at parse time and validate copied-string safety on successful parses. |
| 10 | `DOC-03` is complete package-wide, with clean Godoc and examples on every exported type and function in the final public surface. | ✓ VERIFIED | The prior blocker is fixed: [errors.go](/Users/tazarov/experiments/amikos/pure-simdjson/errors.go:48) and [errors.go](/Users/tazarov/experiments/amikos/pure-simdjson/errors.go:100) now document `(*Error).Error` and `(*Error).Unwrap`; `go doc . Error.Error` and `go doc . Error.Unwrap` render those comments, and [example_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/example_test.go:8) provides executable examples for every exported type. |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `src/runtime/mod.rs` | Locked view and iterator tag constants | ✓ VERIFIED | Contains `ROOT_VIEW_TAG`, `DESC_VIEW_TAG`, `ARRAY_ITER_TAG`, and `OBJECT_ITER_TAG`. |
| `src/runtime/registry.rs` | Descendant validation, scalar accessors, inline iterator transport, object lookup | ✓ VERIFIED | Implements `with_resolved_view`, `encode_descendant_view_locked`, `element_get_*`, `array_iter_*`, and `object_*` functions over registry-backed doc state. |
| `src/lib.rs` | Real exported scalar and iterator ABI surface | ✓ VERIFIED | `pure_simdjson_element_get_*`, `pure_simdjson_bytes_free`, `pure_simdjson_array_iter_*`, `pure_simdjson_object_iter_*`, and `pure_simdjson_object_get_field` all route through `ffi_wrap(...)`. |
| `internal/ffi/bindings.go` | Hidden purego wrappers for the full accessor and traversal surface | ✓ VERIFIED | Symbol registration and typed wrapper methods exist for all Phase 4 exports. |
| `internal/ffi/types.go` | Go mirror structs for array/object iterator transport | ✓ VERIFIED | Defines `ArrayIter` and `ObjectIter` with the ABI field layout. |
| `element.go` | Public scalar, type, array/object traversal, and field-helper API | ✓ VERIFIED | Exports the full `Element`, `Array`, and `Object` Phase 4 surface. |
| `iterator.go` | Public scanner-style iterator API | ✓ VERIFIED | `ArrayIter` and `ObjectIter` expose `Next`, `Value`, `Key`, and `Err`. |
| `element_scalar_test.go` | Numeric, type, closed-state, and malformed UTF-8 semantic coverage | ✓ VERIFIED | Covers classification, numeric boundaries, string/bool/null access, and malformed UTF-8 parse rejection. |
| `iterator_test.go` | Traversal order, missing-vs-null, done-state, and close-state coverage | ✓ VERIFIED | Covers ordered traversal, empty iterators, repeated `Next()` after done, and closed-doc behavior. |
| `tests/rust_shim_accessors.rs` | ABI-facing scalar/string/bool/null tests | ✓ VERIFIED | Covers Rust-owned string buffer cleanup, wrong-type handling, precision loss, and invalid-handle rejection. |
| `tests/rust_shim_iterators.rs` | ABI-facing iterator/object lookup tests | ✓ VERIFIED | Covers array/object order, empty/done behavior, direct field lookup, and reserved/tag invalidation. |
| `example_test.go` | Executable examples for every exported public type | ✓ VERIFIED | Example functions exist for `Parser`, `Doc`, `Element`, `ElementType`, `Array`, `ArrayIter`, `Object`, `ObjectIter`, `ParserPool`, and `Error`. |
| `errors.go` | Package-wide DOC-03 coverage for the exported error surface | ✓ VERIFIED | `(*Error).Error` and `(*Error).Unwrap` now both carry Go doc comments and render through `go doc`. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `element.go` | `internal/ffi/bindings.go` | `Element` public accessors call hidden typed bindings | ✓ VERIFIED | `Type`, `GetUint64`, `GetFloat64`, `GetString`, `GetBool`, `IsNull`, `GetField`, and iterator constructors all call `doc.parser.library.bindings.*`. |
| `internal/ffi/bindings.go` | `src/lib.rs` | purego symbol registration and wrapper methods | ✓ VERIFIED | The Go wrapper binds `pure_simdjson_element_get_*`, `pure_simdjson_bytes_free`, `pure_simdjson_array_iter_*`, `pure_simdjson_object_iter_*`, and `pure_simdjson_object_get_field`. |
| `src/lib.rs` | `src/runtime/registry.rs` | exported C ABI forwards through `runtime::registry::*` | ✓ VERIFIED | Each scalar/string/bool/null/iterator/object export delegates to the corresponding registry function inside `ffi_wrap(...)`. |
| `src/runtime/registry.rs` | `src/native/simdjson_bridge.cpp` | registry uses native descendant accessors and iterator helpers | ✓ VERIFIED | The registry calls `native_element_type_at`, `native_element_get_*_at`, `native_element_get_string_view`, `native_array_iter_bounds`, `native_object_iter_bounds`, and `native_object_get_field_index`. |
| `iterator.go` | `internal/ffi/bindings.go` | scanner-style traversal consumes hidden iterator and string-copy bindings | ✓ VERIFIED | `ArrayIter.Next()` uses `ArrayIterNext`; `ObjectIter.Next()` uses `ObjectIterNext` and then `ElementGetString` for copied keys. |
| `example_test.go` | exported types | executable docs cover the final public surface | ✓ VERIFIED | Example functions exist for `Parser`, `Doc`, `Element`, `ElementType`, `Array`, `ArrayIter`, `Object`, `ObjectIter`, `ParserPool`, and `Error`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `element.go` | `value` / `kind` in `Get*` and `Type()` | `Bindings.Element*` -> `src/lib.rs` exports -> `src/runtime/registry.rs` -> native `psimdjson_element_*_at` | simdjson DOM lookups against the live document, not static returns | ✓ FLOWING |
| `iterator.go` | `currentValue` in `ArrayIter.Next()` | `Bindings.ArrayIterNext` -> `registry::array_iter_next` -> `native_element_after_index` + `encode_descendant_view_locked` | Inline iterator state advances through real tape indexes in the parsed document | ✓ FLOWING |
| `iterator.go` | `currentKey` in `ObjectIter.Next()` | `Bindings.ObjectIterNext` -> `registry::object_iter_next` -> descendant key view -> `Bindings.ElementGetString` | Object keys are copied from live native string views into Go-owned strings | ✓ FLOWING |
| `element.go` | `view` returned by `GetField(...)` | `Bindings.ObjectGetField` -> `registry::object_get_field` -> `native_object_get_field_index` | Field lookup returns a descendant view rooted in the live document, with missing-field detection | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| ABI scalar/string/bool/null surface works | `cargo test --test rust_shim_accessors --quiet` | `7 passed; 0 failed` | ✓ PASS |
| ABI iterator/object lookup surface works | `cargo test --test rust_shim_iterators --quiet` | `4 passed; 0 failed` | ✓ PASS |
| Full Rust test surface still passes | `cargo test --quiet` | `11 + 7 + 2 + 4 + 20 tests passed` | ✓ PASS |
| Public numeric, UTF-8, traversal, and close-state behavior works | `go test ./... -run 'Test(ElementTypeClassification|TypeInvalidOnTamperedView|GetUint64|GetInt64BoundaryContract|GetFloat64|ParseRejectsMalformedUTF8Scalars|GetString|GetBool|IsNull|ArrayIterOrder|ArrayIterEmpty|ObjectIterOrder|ObjectIterEmpty|ObjectGetFieldMissingVsNull|GetStringField|GetStringFieldNullValue|IteratorNextAfterDone|IteratorAfterDocClose)' -count=1` | package tests passed | ✓ PASS |
| Executable examples run for the exported surface | `go test ./... -run '^Example' -count=1` | package examples passed | ✓ PASS |
| Race sweep for accessor and iterator hot paths is clean | `go test -race ./... -run 'Test(ArrayIterOrder|ObjectIterOrder|IteratorAfterDocClose|GetStringField|GetFloat64)' -count=1` | package tests passed under `-race` | ✓ PASS |
| Build and vet still hold on the shipped surface | `cargo build --release --quiet && go build ./... && go vet ./...` | build and vet passed; release build emitted one non-blocking dead-code warning for `err_internal` | ✓ PASS |
| Exported `Error` docs render through Go tooling | `go doc . Error.Error && go doc . Error.Unwrap` | both methods display their new doc comments | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `API-04` | `04-01`, `04-02`, `04-05` | Distinct typed number accessors on `Element` with overflow and precision-loss semantics | ✓ SATISFIED | `element.go` exports `GetInt64`, `GetUint64`, and `GetFloat64`; `element_scalar_test.go` and the targeted `go test` sweep cover the exact boundary cases from the roadmap. |
| `API-05` | `04-01`, `04-02`, `04-04`, `04-05` | `GetString() (string, error)` returns a Go string copy | ✓ SATISFIED | `src/runtime/registry.rs` copies borrowed bytes into a Rust-owned allocation; `internal/ffi/bindings.go` copies them again into a Go string and frees native memory via `defer b.BytesFree(...)`; `TestGetString` verifies copied-string behavior. |
| `API-06` | `04-01`, `04-02`, `04-05` | `GetBool()`, `IsNull()`, and `Type() ElementType` | ✓ SATISFIED | `element.go` exports all three; `element_scalar_test.go` covers bool/null/type classification, tampered views, zero values, and closed docs. |
| `API-07` | `04-03`, `04-04`, `04-05` | Cursor/pull iteration with `Array.Iter`, `Object.Iter`, `Next`, `Value`, and `Key`, without Go callbacks | ✓ SATISFIED | `iterator.go` implements scanner-style iterators; no callback API appears in the Phase 4 path; Go and Rust iterator tests verify ordered traversal and terminal behavior. |
| `API-08` | `04-03`, `04-04`, `04-05` | `Object.GetField(key string) (Element, error)` direct lookup | ✓ SATISFIED | `element.go` exposes `GetField(...)`; `src/runtime/registry.rs` and `src/native/simdjson_bridge.cpp` implement direct field lookup by key; `iterator_test.go` and `tests/rust_shim_iterators.rs` cover present, missing, `null`, and empty-key cases. |
| `DOC-03` | `04-05` | Godoc on every exported type/function in `purejson` | ✓ SATISFIED | The previous `errors.go` blocker is fixed; `go doc` renders the exported `Error` methods, and `example_test.go` plus `go test ./... -run '^Example'` confirm executable example coverage across every exported type. |

All requirement IDs declared in phase 04 plan frontmatter (`API-04`, `API-05`, `API-06`, `API-07`, `API-08`, `DOC-03`) exist in [.planning/REQUIREMENTS.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/REQUIREMENTS.md:36), are mapped to Phase 4 in the traceability table, and are accounted for above. No orphaned Phase 4 requirement IDs were found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `src/runtime/registry.rs` | 695 | Iterator validation authenticates tag, reserved bits, doc handle, and range, but not whether an in-range `state0` was actually issued by the runtime | ⚠️ Warning | Forged in-range iterator state can plausibly survive validation and produce a bogus descendant view instead of deterministic `ERR_INVALID_HANDLE`. This weakens the hardening story but does not break the verified user-facing API contract. |
| `src/native/simdjson_bridge.cpp` | 172 | `element_at(...)` reconstructs `dom::element` through `reinterpret_cast` aliasing | ⚠️ Warning | This is a release-build correctness risk on the descendant accessor path because it relies on undefined aliasing behavior, even though current tests still pass. |

### Disconfirmation Notes

| Check | Result |
| --- | --- |
| Partially met requirement | Iterator-state hardening is only partial: `with_iter_doc(...)` validates structure and range, but not issuance provenance for in-range `state0` values. |
| Passing test that does not fully prove the claim | `go test ./... -run '^Example'` proves the examples compile and run, but it does not prove every rendered Godoc page is aesthetically ideal; the DOC-03 pass here relies on source-comment presence plus `go doc` rendering for the previously missing exported methods. |
| Error path without direct test coverage | No automated test currently forges an in-range iterator `state0` and asserts `ERR_INVALID_HANDLE`; existing iterator tests cover wrong tag/reserved bits, empty iterators, done-state, and close-state instead. |

### Gaps Summary

The previous gating failure is closed. `errors.go` now documents both exported `Error` methods, `go doc` renders those comments, the example matrix still passes, and the accessor/traversal surface remains wired and behaviorally correct under targeted Rust, Go, and race spot-checks.

Phase 04 now achieves its goal and satisfies the declared Phase 4 requirements (`API-04` through `API-08`, plus `DOC-03`). Two non-gating warnings remain around iterator-handle hardening and C++ aliasing safety; they are real residual risks, but they do not invalidate the shipped v0.1 accessor and cursor-pull API that this phase was responsible for delivering.

---

_Verified: 2026-04-17T10:14:22Z_
_Verifier: Claude (gsd-verifier)_
