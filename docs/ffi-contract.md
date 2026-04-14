# Scope

This document is the normative FFI contract for `pure-simdjson` ABI `v0.1`. It defines the public C ABI exported in [include/pure_simdjson.h](/Users/tazarov/experiments/amikos/pure-simdjson/include/pure_simdjson.h) and the semantic rules later phases must implement verbatim.

`v0.1` is DOM-only. On-Demand APIs, pinned-input parsing, and borrowed string-view APIs remain deferred work and are not part of this contract.

The generated header is authoritative for exact symbol names, field names, and C types. This document is authoritative for lifecycle, ownership, diagnostics, panic/exception policy, and compatibility rules. Go-side consumers must enforce ABI compatibility against `^0.1.x`.

# ABI invariants

- Every exported `pure_simdjson_*` function returns `int32_t`.
- Success is `PURE_SIMDJSON_OK == 0`. All other return values are stable numeric error codes.
- Multi-value results always flow through pointer out-params. The ABI does not return structs by value.
- Public signatures must not mix floating-point and integer scalar parameters in the same argument list. Numeric results are returned through out-params such as `int64_t *`, `uint64_t *`, or `double *`.
- `pure_simdjson_handle_t` is the only opaque-handle transport type. `pure_simdjson_value_view_t`, `pure_simdjson_array_iter_t`, and `pure_simdjson_object_iter_t` are lightweight document-tied view/iterator structs.
- Control flow is driven by numeric status codes, not by diagnostic strings.

# Error code space

| Symbol | Value | Meaning |
| --- | ---: | --- |
| `PURE_SIMDJSON_OK` | 0 | Success |
| `PURE_SIMDJSON_ERR_INVALID_ARGUMENT` | 1 | Null pointer, inconsistent out-param set, or other caller contract violation |
| `PURE_SIMDJSON_ERR_INVALID_HANDLE` | 2 | Handle generation mismatch, stale handle, or already-freed slot |
| `PURE_SIMDJSON_ERR_PARSER_BUSY` | 3 | Parser already owns a live `Doc` |
| `PURE_SIMDJSON_ERR_WRONG_TYPE` | 4 | Value kind does not match the requested accessor |
| `PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND` | 5 | Requested object field is absent |
| `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL` | 6 | Caller-provided destination buffer is too small |
| `PURE_SIMDJSON_ERR_INVALID_JSON` | 32 | Parse failure, including malformed JSON and invalid UTF-8 in DOM mode |
| `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE` | 33 | Numeric value cannot fit the requested integer domain |
| `PURE_SIMDJSON_ERR_PRECISION_LOSS` | 34 | Requested numeric conversion would lose precision |
| `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` | 64 | simdjson selected an unsupported kernel for this product policy |
| `PURE_SIMDJSON_ERR_ABI_MISMATCH` | 65 | Loader and library ABI versions are incompatible |
| `PURE_SIMDJSON_ERR_PANIC` | 96 | Rust panic trapped at the FFI boundary in unwind-enabled builds |
| `PURE_SIMDJSON_ERR_CPP_EXCEPTION` | 97 | C++ exception trapped before it could cross into Rust |
| `PURE_SIMDJSON_ERR_INTERNAL` | 127 | Internal failure or contract-only stub path |

These values are part of the public ABI. Downstream wrappers may map them to richer errors, but they must preserve the numeric meaning.

# Handle format

`pure_simdjson_handle_t` is a packed `uint64_t` with `slot:u32 | generation:u32`.

- `slot` identifies a registry entry.
- `generation` increments when the slot is freed and reused.
- `pure_simdjson_handle_parts_t` is the explicit split view:

```c
typedef struct pure_simdjson_handle_parts_t {
  uint32_t slot;
  uint32_t generation;
} pure_simdjson_handle_parts_t;
```

`Parser` and `Doc` are represented only by `pure_simdjson_handle_t`. Handles are never raw pointers in the public ABI. Any stale, double-freed, or mismatched generation must fail with `PURE_SIMDJSON_ERR_INVALID_HANDLE` rather than producing undefined behavior.

# Value and iterator model

`pure_simdjson_value_view_t` is a lightweight document-tied view:

```c
typedef struct pure_simdjson_value_view_t {
  pure_simdjson_handle_t doc;
  uint64_t state0;
  uint64_t state1;
  uint32_t kind_hint;
  uint32_t reserved;
} pure_simdjson_value_view_t;
```

Rules:

- A value view is only valid while its owning `Doc` remains live.
- `state0` and `state1` are opaque ABI fields owned by the native implementation.
- `kind_hint` uses `pure_simdjson_value_kind_t` and is advisory; callers still check return codes on accessors.
- `pure_simdjson_array_iter_t` and `pure_simdjson_object_iter_t` are stateful, document-tied iterators driven from Go/C by repeated `*_next` calls.
- `pure_simdjson_doc_root`, `pure_simdjson_object_get_field`, `pure_simdjson_array_iter_next`, and `pure_simdjson_object_iter_next` return new view state through out-params rather than allocating child handles.

Split numeric access is mandatory:

- `pure_simdjson_element_get_int64` returns a signed integer only.
- `pure_simdjson_element_get_uint64` returns an unsigned integer only.
- `pure_simdjson_element_get_float64` returns a `double` only.
- `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE` covers values that cannot fit the requested integer domain.
- `PURE_SIMDJSON_ERR_PRECISION_LOSS` covers values that cannot be represented without loss in the requested floating/integer conversion path.
- `PURE_SIMDJSON_ERR_WRONG_TYPE` is used when the value kind is not numeric for the requested accessor.

This contract forbids a combined "number union" return or automatic widening that hides overflow or rounding behavior.

# Parser lifecycle

The parser/document state machine is fixed:

1. `pure_simdjson_parser_new` creates a parser handle.
2. `pure_simdjson_parser_parse` parses one input buffer into one `Doc`.
3. `pure_simdjson_doc_root` resolves the root view for that `Doc`.
4. Value accessors and iterators operate while the `Doc` remains live.
5. `pure_simdjson_doc_free` releases the `Doc`.
6. `pure_simdjson_parser_free` releases the parser after all associated documents are gone.

Busy-state rule:

- `pure_simdjson_parser_parse` returns `PURE_SIMDJSON_ERR_PARSER_BUSY` while a live `Doc` exists for that parser.
- Re-parse never discards, replaces, or mutates the old `Doc` as a side effect.
- Only `pure_simdjson_doc_free` clears the busy state.
- Generation checks remain the mechanism that turns stale parser/doc/view use into `PURE_SIMDJSON_ERR_INVALID_HANDLE`.

The lifecycle is explicit by design so later phases do not introduce hidden reuse semantics.

# Ownership and padding

Every `pure_simdjson_parser_parse` call copies `input_ptr[..input_len]` into Rust-owned padded storage before simdjson sees it.

Rules:

- The public ABI never stores a Go-owned or caller-owned input pointer beyond the parsing call.
- The copied buffer must satisfy `SIMDJSON_PADDING` semantics.
- `Doc` lifetime owns the backing parsed storage and any document-tied views/iterators derived from it.
- `Doc` release invalidates all derived `pure_simdjson_value_view_t`, `pure_simdjson_array_iter_t`, and `pure_simdjson_object_iter_t` state.

This choice is part of the `v0.1` contract and is not an optimization detail.

# Strings and diagnostics

String access is copy-out only in `v0.1`.

`pure_simdjson_element_get_string` returns:

- `uint8_t **out_ptr`
- `size_t *out_len`

The callee allocates the returned byte buffer, and the caller must release it with `pure_simdjson_bytes_free(ptr, len)`. String getters that expose borrowed document memory are outside this contract.

Diagnostics helpers are part of the ABI surface:

- `pure_simdjson_get_abi_version`
- `pure_simdjson_get_implementation_name_len`
- `pure_simdjson_copy_implementation_name`
- `pure_simdjson_parser_get_last_error_len`
- `pure_simdjson_parser_copy_last_error`
- `pure_simdjson_parser_get_last_error_offset`

Diagnostics are advisory only:

- Callers must branch on the `int32_t` status code first.
- Diagnostic text and offsets help logging and debugging but do not redefine success/failure.
- `pure_simdjson_parser_copy_last_error` and `pure_simdjson_copy_implementation_name` use bounded caller-provided buffers and may return `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`.

# ABI version handshake

The ABI version export is `pure_simdjson_get_abi_version`.

- The current packed ABI version is `0x00010000`.
- The compatibility rule for `v0.1` consumers is `^0.1.x`.
- A loader or wrapper that detects an incompatible version must fail with `PURE_SIMDJSON_ERR_ABI_MISMATCH` rather than attempting best-effort execution.

`pure_simdjson_get_implementation_name_len` plus `pure_simdjson_copy_implementation_name` provide the active implementation identity for diagnostics and support cases; they do not participate in compatibility decisions.

# Panic and exception policy

Every exported Rust ABI function must be authored through an `ffi_fn!`-style wrapper that applies `catch_unwind` in unwind-enabled builds and converts trapped panics into `PURE_SIMDJSON_ERR_PANIC`.

Rules:

- `ffi_fn!` is mandatory for every public export.
- `catch_unwind` is required when unwinding is enabled so Rust panics do not cross the C ABI boundary.
- Release `panic = "abort"` is a policy choice in `Cargo.toml`; it means release panics still terminate the process and must not be described as returned status codes.
- The Rust/C++ seam must use non-throwing simdjson access patterns such as `.get(err)`.
- C++ exceptions must be trapped before re-entering Rust and converted into `PURE_SIMDJSON_ERR_CPP_EXCEPTION`.
- No foreign exception or Rust unwind may cross into Go or C callers.

This section is normative even though the full shim implementation lands in later phases.

# Worked call sequences

## String copy and free

```c
pure_simdjson_handle_t parser = 0;
pure_simdjson_handle_t doc = 0;
pure_simdjson_value_view_t root = {0};
uint8_t *bytes = NULL;
size_t len = 0;

pure_simdjson_parser_new(&parser);
pure_simdjson_parser_parse(parser, json_ptr, json_len, &doc);
pure_simdjson_doc_root(doc, &root);
pure_simdjson_element_get_string(&root, &bytes, &len);
/* caller copies/consumes bytes[0..len] */
pure_simdjson_bytes_free(bytes, len);
pure_simdjson_doc_free(doc);
pure_simdjson_parser_free(parser);
```

## Parser busy failure

```c
pure_simdjson_handle_t parser = 0;
pure_simdjson_handle_t doc_a = 0;
pure_simdjson_handle_t doc_b = 0;

pure_simdjson_parser_new(&parser);
pure_simdjson_parser_parse(parser, json_a_ptr, json_a_len, &doc_a);   /* returns 0 */
pure_simdjson_parser_parse(parser, json_b_ptr, json_b_len, &doc_b);   /* returns PURE_SIMDJSON_ERR_PARSER_BUSY */
pure_simdjson_doc_free(doc_a);                                         /* clears busy state */
pure_simdjson_parser_parse(parser, json_b_ptr, json_b_len, &doc_b);   /* may now proceed */
pure_simdjson_doc_free(doc_b);
pure_simdjson_parser_free(parser);
```

# Deferred items

- On-Demand traversal and consumption tracking remain a later-version concern.
- Borrowed string-view APIs remain deferred; `v0.1` uses copy-out plus `pure_simdjson_bytes_free`.
- Pinned-input parse variants are deferred until a later design explicitly models pin lifetimes.
