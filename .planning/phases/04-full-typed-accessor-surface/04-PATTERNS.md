# Phase 4: Full Typed Accessor Surface - Patterns

## Purpose

Map the Phase 4 work to the closest existing implementation patterns in the repo so the plans can name exact analog files and reuse points.

## Pattern 1: Rust export wrappers in `src/lib.rs`

### Best analog

- `src/lib.rs` `pure_simdjson_element_get_int64(...)`

### Reuse pattern

- Public export stays thin.
- Every export is wrapped by `ffi_wrap(...)`.
- Real logic lives under `runtime::registry::*`.
- Success writes through `write_out(...)`.
- Failure returns the exact `pure_simdjson_error_code_t` from runtime helpers.

### Phase 4 mapping

Use the same pattern for:

- `pure_simdjson_element_get_uint64`
- `pure_simdjson_element_get_float64`
- `pure_simdjson_element_get_string`
- `pure_simdjson_bytes_free`
- `pure_simdjson_element_get_bool`
- `pure_simdjson_element_is_null`
- `pure_simdjson_array_iter_new`
- `pure_simdjson_array_iter_next`
- `pure_simdjson_object_iter_new`
- `pure_simdjson_object_iter_next`
- `pure_simdjson_object_get_field`

## Pattern 2: Runtime bridge shims in `src/runtime/mod.rs`

### Best analogs

- `native_element_type(...)`
- `native_element_get_int64(...)`
- existing `extern "C"` declarations for `psimdjson_*`

### Reuse pattern

- Declare the bridge symbol in the `unsafe extern "C"` block.
- Add one small Rust helper that calls the bridge and returns `Result<T, pure_simdjson_error_code_t>`.
- Keep bridge-specific pointer handling here, not in `src/lib.rs`.

### Phase 4 mapping

Add the remaining scalar/string/bool/null helpers first, then iterator/object helpers:

- `psimdjson_element_get_uint64`
- `psimdjson_element_get_float64`
- `psimdjson_element_get_string`
- `psimdjson_bytes_free`
- `psimdjson_element_get_bool`
- `psimdjson_element_is_null`
- `psimdjson_array_iter_new`
- `psimdjson_array_iter_next`
- `psimdjson_object_iter_new`
- `psimdjson_object_iter_next`
- `psimdjson_object_get_field`

## Pattern 3: Registry validation in `src/runtime/registry.rs`

### Best analogs

- `doc_root(...)`
- `with_validated_view(...)`
- `element_type(...)`
- `element_get_int64(...)`

### Reuse pattern

- Validate public transport structs before touching native pointers.
- Reject null, zero, non-zero reserved fields, and stale handles as `ERR_INVALID_HANDLE`.
- Resolve the owning doc from the registry, then drop the lock before native calls.
- Keep doc lifetime as the single source of truth for view validity.

### Phase 4 mapping

Phase 4 needs the same structure, but generalized for descendant views and iterators:

- one helper to validate any `pure_simdjson_value_view_t`
- one helper to encode native child elements back into a view
- one helper each for validating `pure_simdjson_array_iter_t` and `pure_simdjson_object_iter_t`

## Pattern 4: purego bindings in `internal/ffi/bindings.go`

### Best analogs

- `Bind(...)` symbol table
- `ElementType(...)`
- `ElementGetInt64(...)`
- string-copy helpers like `ImplementationName()` and `ParserLastError()`

### Reuse pattern

- Add one function pointer field per symbol.
- Register it once in `Bind(...)`.
- Expose a typed wrapper method that does the native call and then `runtime.KeepAlive(...)` on any buffer/owner inputs.

### Phase 4 mapping

- Scalar/string/bool/null bindings should mirror the `ElementGetInt64(...)` shape.
- `ElementGetString(...)` should follow the "copy then convert" style already used by `ImplementationName()` / `ParserLastError()`, but it must also free native bytes.
- Iterator wrappers should return mirror structs from `internal/ffi/types.go`, not raw `unsafe.Pointer` values at the public layer.

## Pattern 5: Public Go wrappers in `element.go`

### Best analogs

- `Element.GetInt64()`
- `Element.AsArray()`
- `Element.AsObject()`

### Reuse pattern

- Check `e.doc == nil || e.doc.isClosed()` first.
- Delegate to the hidden FFI binding.
- Use `wrapStatus(...)` for native return-code translation.
- Call `runtime.KeepAlive(e.doc)` after the purego call.
- Return small value wrappers (`Element`, `Array`, `Object`) rather than pointer-owned wrapper objects.

### Phase 4 mapping

- Keep `GetUint64`, `GetFloat64`, `GetString`, `GetBool`, `IsNull`, and `Type` on `Element`.
- Keep `Array` and `Object` as lightweight wrappers over `Element`.
- Add iterator types in a separate Go file if `element.go` becomes too large.

## Pattern 6: Public Go lifecycle checks in `doc.go`

### Best analogs

- `Doc.Root()`
- `Doc.Close()`
- `Doc.isClosed()`

### Reuse pattern

- Doc liveness is centralized on `Doc`.
- Element accessors consult doc state instead of guessing from value contents.
- Cleanup paths call `runtime.KeepAlive(...)` after FFI calls.

### Phase 4 mapping

- Iterators should also carry or reference the owning doc so `Next()` can fail cleanly after `Doc.Close()`.
- Do not introduce a second liveness model for iterators or field lookups.

## Pattern 7: Go semantic tests

### Best analogs

- `parser_test.go`
- `phase3_followup_test.go`
- `pool_test.go`

### Reuse pattern

- Test behavior semantically, not only via grep/static checks.
- Use table-driven cases for wrong-type, closed-state, and error mapping.
- Keep top-level names explicit and grep-able.

### Phase 4 mapping

Add focused tests for:

- element type classification
- uint64 / float64 / string / bool / null access
- numeric overflow and precision-loss behavior
- array iteration order
- object iteration order and key copying
- missing-vs-null field behavior
- closed-doc iterator/accessor behavior
- `GetStringField(name)` behavior

## Pattern 8: Rust native integration tests

### Best analogs

- `tests/rust_shim_minimal.rs`
- `tests/rust_shim_fallback_gate.rs`

### Reuse pattern

- Exercise the Rust ABI through the public exports, not by calling bridge internals directly.
- Assert exact error-code behavior where the ABI contract locks it.

### Phase 4 mapping

Create focused Rust integration tests for:

- scalar/string/bool/null exports
- bytes allocation/free path
- iterator and object-lookup exports
- invalid-handle / reserved-bit rejection on views and iterator structs

## Suggested Target -> Analog Map

| Phase 4 target | Closest analog | Why |
|----------------|----------------|-----|
| `src/lib.rs` new exports | `pure_simdjson_element_get_int64` | Exact export wrapper structure |
| `src/runtime/mod.rs` new bridge helpers | `native_element_get_int64` | Thin native call / `Result` conversion |
| `src/runtime/registry.rs` child-view validation | `with_validated_view` | Existing doc-tied validation model |
| `internal/ffi/bindings.go` scalar wrappers | `ElementGetInt64` | Existing purego wrapper shape |
| `internal/ffi/bindings.go` string wrapper | `ImplementationName` | Existing copy-into-Go helper style |
| `element.go` new accessors | `GetInt64` | Existing public accessor behavior |
| `ArrayIter` / `ObjectIter` | `ParserPool` tests + `Element` wrappers | Small public structs with semantic tests |
| Go docs/examples | Phase 3 source comments and docs plan | Existing DOC-03 close-out pattern |
