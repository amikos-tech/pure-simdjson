# Phase 4: Full Typed Accessor Surface - Research

**Date:** 2026-04-17
**Status:** Ready for planning

## Objective

Define the safest implementation shape for Phase 4 so the shipped `purejson` API can expose every DOM value type plus array/object traversal without breaking the already-locked ABI, lifecycle model, or error semantics from Phases 1-3.

The planning question is not "how do we add methods?" It is "how do we finish the remaining native exports, expand the purego layer, preserve document-tied safety for descendant views and iterators, and prove the numeric/string/error edge cases without letting Phase 4 bleed into bootstrap or On-Demand work?"

## Constraints That Are Already Locked

- `Element.Type()` must expose concrete JSON kinds directly, including distinct `Int64`, `Uint64`, and `Float64` classifications.
- `Type()` is total: on stale/closed elements it returns an invalid sentinel instead of an error.
- Public iteration must be scanner-style: `Next() bool`, then `Value()` / `Key()`, then `Err() error`.
- `ObjectIter.Key()` returns a copied Go `string`, not a generic `Element`.
- `Object.GetField(key)` must distinguish missing key (`ErrElementNotFound`) from present `null` (valid `Element`, then `IsNull()` is true).
- Typed getters on `null` or any wrong concrete type must return `ErrWrongType`.
- `GetStringField(name)` is in scope for Phase 4, but broader helper families are deferred.
- The C ABI is already fixed in `include/pure_simdjson.h`; planning should avoid assuming new public exports can be added casually.

## Current Repo Implications

### 1. The bridge and Rust ABI still stop at the Phase 3 happy path

- `src/native/simdjson_bridge.h` and `src/native/simdjson_bridge.cpp` currently expose only parser lifecycle, diagnostics, `doc_root`, `element_type`, and `element_get_int64`.
- `src/lib.rs` already contains the full Phase 4 export names, but `pure_simdjson_element_get_uint64`, `pure_simdjson_element_get_float64`, `pure_simdjson_element_get_string`, `pure_simdjson_bytes_free`, `pure_simdjson_element_get_bool`, `pure_simdjson_element_is_null`, `pure_simdjson_array_iter_new`, `pure_simdjson_array_iter_next`, `pure_simdjson_object_iter_new`, `pure_simdjson_object_iter_next`, and `pure_simdjson_object_get_field` are still contract-only stubs.
- Phase 4 therefore requires real work in the C++ bridge, the Rust runtime layer, and the Rust ABI shim before the public Go API can do anything meaningful.

### 2. Descendant views are the biggest hidden structural gap

- `src/runtime/registry.rs` currently validates `pure_simdjson_value_view_t` through `with_validated_view(...)`, but that helper accepts only root views: `state1` must equal `ROOT_VIEW_TAG`, and `state0` must match the document's root pointer.
- That is sufficient for `Doc.Root().GetInt64()` and nothing more.
- Array iteration, object iteration, and `Object.GetField(...)` all require child views that are still document-tied but are not the root pointer.
- Phase 4 needs a generalized internal view encoding/validation scheme before field lookup or iteration can be implemented safely.

### 3. The Go FFI layer is intentionally narrow today

- `internal/ffi/bindings.go` binds only the Phase 3 symbols and exposes wrappers for `ElementType()` and `ElementGetInt64()`.
- `internal/ffi/types.go` already mirrors the split `ValueKind` enum, which is good news for `ElementType` because the concrete numeric distinction is already represented in the hidden layer.
- Iterator transport structs are not mirrored in Go yet, and there is no wrapper for native string allocation/free.

### 4. The public Go surface is a skeleton that should be extended, not replaced

- `element.go` already defines `Element`, `Array`, and `Object`, plus `GetInt64()`, `AsArray()`, and `AsObject()`.
- `doc.go` already centralizes live-doc checks and cached-root behavior.
- `errors.go` already carries the sentinel model required for Phase 4 (`ErrWrongType`, `ErrElementNotFound`, `ErrNumberOutOfRange`, `ErrPrecisionLoss`, `ErrInvalidJSON`, `ErrClosed`).
- Planning should extend those shapes rather than introduce a second access layer or a pointer-heavy redesign.

### 5. `GetStringField(name)` cannot be a new ABI export in Phase 4 without reopening the contract

- The committed header has `object_get_field`, not `object_get_string_field`.
- That means the safe v0.1 implementation path is a Go helper composed from `GetField(name)` + `GetString()`.
- A true single-call fast path would require either a contract change or a hidden non-ABI path that purego still cannot call directly. That is not a good Phase 4 bet.

## Recommended Technical Direction

### 1. Make value views descendant-safe before building iterator APIs

Phase 4 should first introduce internal helpers in the Rust runtime that can:

- encode a validated child `simdjson::dom::element` pointer into `pure_simdjson_value_view_t`
- distinguish root views from descendant views through implementation-owned tags in `state1`
- reject zero pointers, mismatched tags, mismatched doc handles, and non-zero `reserved` fields as `PURE_SIMDJSON_ERR_INVALID_HANDLE`
- preserve the current rule that views remain valid only while the owning doc is live

Without that, any iterator or field-lookup implementation will either:

- duplicate unsafe validation logic in multiple places, or
- silently treat child views as root views, which will mis-handle stale references.

### 2. Implement the remaining scalar/string/bool/null exports in one native slice

The scalar/string/bool/null family belongs together because they all depend on the same generalized descendant-view validation:

- `element_get_uint64`
- `element_get_float64`
- `element_get_string`
- `bytes_free`
- `element_get_bool`
- `element_is_null`

Planning should require:

- new bridge declarations/definitions in `src/native/simdjson_bridge.h` and `.cpp`
- new `extern "C"` declarations and thin wrappers in `src/runtime/mod.rs`
- registry entry points in `src/runtime/registry.rs`
- real `src/lib.rs` exports routed through `ffi_wrap(...)`
- purego bindings in `internal/ffi/bindings.go`

### 3. Treat string copy-out and free as a first-class correctness boundary

The Phase 1 contract says `element_get_string` allocates a byte buffer and the caller releases it with `bytes_free`.

That creates a cross-language ownership edge the current Go wrapper does not have yet.

Planning should force the Go binding/wrapper path to:

- copy native bytes into a Go `string`
- always release the native allocation with `bytes_free`
- handle empty strings and zero-length buffers explicitly
- keep the owning `Doc` live across the native call via `runtime.KeepAlive(...)`

This is one of the few places where a successful accessor still has manual native cleanup work.

### 4. Keep public type classification thin and exact

The best public API shape is a direct public enum that mirrors the hidden `ffi.ValueKind` split:

- `TypeInvalid`
- `TypeNull`
- `TypeBool`
- `TypeInt64`
- `TypeUint64`
- `TypeFloat64`
- `TypeString`
- `TypeArray`
- `TypeObject`

`Type()` should:

- return `TypeInvalid` when the doc is closed or the `Element` has no live doc
- otherwise call the already-bound/native `element_type`
- not invent a second number classifier in v0.1

That aligns with the locked D-01/D-02 decisions and keeps the API explainable.

### 5. Build iterators as a separate native slice after descendant views exist

Array/object traversal and direct field lookup belong together:

- `array_iter_new`
- `array_iter_next`
- `object_iter_new`
- `object_iter_next`
- `object_get_field`

Why separate them from the scalar accessors:

- they introduce new transport structs in both ABI and Go mirror layers
- they need iterator-state validation in addition to value-view validation
- they determine the eventual ergonomics of `ArrayIter` / `ObjectIter`

The recommended Go scanner shape is:

- `Array.Iter() *ArrayIter`
- `Object.Iter() *ObjectIter`
- `Next() bool`
- `Value() Element`
- `Key() string` on `ObjectIter`
- `Err() error`

Implementation detail to lock during planning:

- `Key()` should be populated by converting the key view to a Go string during `Next()`, not by exposing a persistent native pointer
- `GetStringField(name)` should be implemented in Go as `GetField(name)` followed by `GetString()`

### 6. Treat malformed UTF-8 verification as a parse-and-access contract, not only a parser unit test

The existing Phase 3 tests already prove that malformed UTF-8 at parse time returns `ErrInvalidJSON`.

For Phase 4, the verification burden is broader:

- malformed UTF-8 payloads that should fail parse must still map to `ErrInvalidJSON`
- successful string accessor calls must never hand back corrupted Go strings
- object-field and iterator code paths must not bypass the same native error mapping used by root accessors

The likely outcome is that most malformed UTF-8 cases still fail at `Parse(...)`, but the plan should still force coverage through the Phase 4 string/object paths so the success criterion is closed honestly.

## Planning Risks To Address Explicitly

### Risk 1: Phase 4 looks like "just Go wrappers" but is blocked by native view semantics

If the plans start with only `element.go`, execution will immediately hit the root-only validation limitation in `src/runtime/registry.rs`.

### Risk 2: Native string allocation can leak or double-free

`bytes_free` introduces the first successful-path native cleanup contract in the Go wrapper. Planning must specify where ownership transfers and how cleanup is guaranteed.

### Risk 3: Iterator state can become another stale-handle class

The ABI iterator structs carry `state0`, `state1`, `index`, `tag`, and `reserved`. If planning does not force validation of `reserved` and implementation-owned tags, iterator misuse will surface as silent corruption instead of `ErrInvalidHandle`.

### Risk 4: `GetStringField(name)` could waste time chasing a non-existent ABI fast path

The committed public ABI has no dedicated fast-path symbol. Planning should avoid speculative work here and ship the helper as composition in v0.1.

### Risk 5: DOC-03 can fail even when code works

The roadmap requires clean Godoc plus examples on every exported type. If docs are left as "cleanup at the end," the phase can pass behaviorally and still miss its documentation exit gate.

## Validation Architecture

Phase 4 should keep the same mixed Rust+Go validation strategy as Phase 3, but widen the semantic surface:

- **Quick loop:** `cargo test --lib && go test ./...`
- **Focused scalar loop:** `cargo test --test rust_shim_accessors && go test ./... -run 'Test(ElementType|GetUint64|GetFloat64|GetString|GetBool|IsNull)'`
- **Focused iterator loop:** `cargo test --test rust_shim_iterators && go test ./... -run 'Test(ArrayIter|ObjectIter|ObjectGetField|GetStringField)'`
- **Full suite:** `cargo test && cargo build --release && go test ./... -race`

The planner should also require static documentation verification:

- `rg '^// ElementType|^// ArrayIter|^// ObjectIter|^// GetUint64|^// GetFloat64|^// GetString|^// GetBool|^// IsNull|^// Type|^// Iter|^// GetField|^// GetStringField|^// Next|^// Value|^// Key|^// Err'`
- `go test ./... -run 'Example'`

Estimated local feedback target: keep the quick loop under ~120 seconds.

## Suggested Plan Shape

Phase 4 breaks cleanly into five plans:

1. **Native scalar/string/bool/null substrate**
   - generalize value-view validation beyond root-only
   - implement the remaining scalar/string/bool/null exports plus `bytes_free`
   - expand purego bindings for those symbols

2. **Public Go scalar/type/string/bool/null surface**
   - add `ElementType`, `Type()`, `GetUint64()`, `GetFloat64()`, `GetString()`, `GetBool()`, `IsNull()`
   - cover numeric overflow, precision-loss, wrong-type, closed-doc, and string-copy semantics

3. **Native iterator and object-lookup substrate**
   - implement `array_iter_*`, `object_iter_*`, and `object_get_field`
   - add Go FFI mirror structs and wrappers for iterator transport

4. **Public Go array/object traversal and field helpers**
   - add `ArrayIter`, `ObjectIter`, `Array.Iter()`, `Object.Iter()`, `Object.GetField()`, `Object.GetStringField()`
   - keep `GetStringField(name)` as Go composition over `GetField` + `GetString`

5. **Docs, examples, and phase-close verification**
   - complete DOC-03 across every newly exported symbol
   - add example coverage and the full numeric/UTF-8/iteration verification sweep

## Deliverables The Planner Should Force

- A generalized descendant-safe `ValueView` validation path in the Rust runtime
- Real native bridge exports for Phase 4 scalar/string/bool/null functions plus `bytes_free`
- Real native iterator/object-lookup exports and Go mirror structs for iterator state
- A public `ElementType` enum that mirrors the exact numeric split already present in the hidden FFI layer
- `GetStringField(name)` implemented as composition, not speculative ABI expansion
- Semantic Go tests for overflow, precision loss, missing-vs-null, iteration order, and closed-doc behavior
- Rust integration coverage for the newly activated native exports
- Source docs and examples for every newly exported type/method required by DOC-03

## Research Conclusion

Phase 4 is best planned as two native slices and two public-Go slices, with documentation/verification as a final explicit close-out plan. The key architectural fact is that descendant views and iterator state are not representable safely with the current root-only runtime validation. Once that is fixed, the rest of the phase becomes straightforward: activate the remaining ABI exports, extend the purego layer, expose thin public Go wrappers, and close the phase with numeric/UTF-8/iteration semantics plus Godoc/example coverage.
