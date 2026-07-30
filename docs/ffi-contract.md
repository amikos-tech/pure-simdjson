# Scope

This document is the normative FFI contract for `pure-simdjson` ABI 1.2 (`0x00010002`). It defines the public C ABI exported in [include/pure_simdjson.h](../include/pure_simdjson.h).

The `v0.1.x` product line remains DOM-only. On-Demand APIs, pinned-input parsing, and borrowed string-view APIs remain deferred work and are not part of this contract.

The generated header is authoritative for exact symbol names, field names, and C types. This document is authoritative for lifecycle, ownership, limits, kernel selection, diagnostics, panic/exception policy, and compatibility rules. `^0.1.x` describes the Go module's semantic-version family; native compatibility is the stricter ABI handshake defined below.

ABI 1.2 extends the original typed DOM surface with configured parser construction, exact BigInt copy-out, explicit known-offset state, and process-global implementation control. It is append-only: every ABI 1.1 symbol, signature, numeric value, and public layout remains intact.

# ABI invariants

- Every exported `pure_simdjson_*` function returns `pure_simdjson_error_code_t`, with `int32_t` wire representation.
- Success is `PURE_SIMDJSON_OK == 0`. All other return values are stable numeric error codes.
- Multi-value results always flow through pointer out-params. The ABI does not return structs by value.
- Public signatures must not mix floating-point and integer scalar parameters in the same argument list. Numeric results are returned through out-params such as `int64_t *`, `uint64_t *`, or `double *`.
- `pure_simdjson_handle_t` is the generic packed transport type. Public signatures use the source-level aliases `pure_simdjson_parser_t` and `pure_simdjson_doc_t` to distinguish parser/document roles without changing the underlying wire representation.
- `pure_simdjson_value_view_t`, `pure_simdjson_array_iter_t`, and `pure_simdjson_object_iter_t` are lightweight document-tied view/iterator structs.
- Control flow is driven by numeric status codes, not by diagnostic strings.
- Boolean out-params typed as `uint8_t *` are written as exactly `0` for false and `1` for true. Callers may rely on strict `0`/`1` rather than accepting any non-zero value.
- ABI growth within major 1 is append-only. Existing enum values are never renumbered, existing function signatures are never replaced, and existing public structs are never resized or reinterpreted.
- The pinned sizes remain 8 bytes for handles and `pure_simdjson_handle_parts_t`, 32 bytes for `pure_simdjson_value_view_t` and both iterator structs, and 48 bytes for `pure_simdjson_native_alloc_stats_t`.
- Internal `psdj_internal_*` and `psimdjson_*` bridge/test symbols are not part of the public header or compatibility surface.

# Out-param semantics

- Unless documented otherwise, output pointers are written only on `PURE_SIMDJSON_OK`.
- Size-reporting outputs named `out_len` or `out_written` may also be written on `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL` so callers can learn the required capacity.
- For bounded copy helpers such as `pure_simdjson_copy_implementation_name`, `dst = NULL` with `dst_cap = 0` is a valid size probe. A null destination with sufficient capacity to perform the copy remains `PURE_SIMDJSON_ERR_INVALID_ARGUMENT`.

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
| `PURE_SIMDJSON_ERR_NOT_IMPLEMENTED` | 7 | Optional diagnostic surface is unavailable |
| `PURE_SIMDJSON_ERR_DEPTH_LIMIT` | 8 | Input exceeds the parser's immutable maximum depth |
| `PURE_SIMDJSON_ERR_CAPACITY_LIMIT` | 9 | Input exceeds the parser's immutable maximum capacity |
| `PURE_SIMDJSON_ERR_KERNEL_LOCKED` | 10 | Process-global implementation selection is permanently locked |
| `PURE_SIMDJSON_ERR_INVALID_JSON` | 32 | Parse failure, including malformed JSON and invalid UTF-8 in DOM mode |
| `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE` | 33 | Numeric value cannot fit the requested integer domain |
| `PURE_SIMDJSON_ERR_PRECISION_LOSS` | 34 | Requested numeric conversion would lose precision |
| `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` | 64 | simdjson selected an unsupported kernel for this product policy |
| `PURE_SIMDJSON_ERR_ABI_MISMATCH` | 65 | Loader and library ABI versions are incompatible |
| `PURE_SIMDJSON_ERR_PANIC` | 96 | Rust panic trapped at the FFI boundary in unwind-enabled builds |
| `PURE_SIMDJSON_ERR_CPP_EXCEPTION` | 97 | C++ exception trapped before it could cross into Rust |
| `PURE_SIMDJSON_ERR_INTERNAL` | 127 | Internal failure or contract-only stub path |

These values are part of the public ABI. Downstream wrappers may map them to richer errors, but they must preserve the numeric meaning.

The numeric gaps between assigned values (11–31, 35–63, 66–95, 98–126) are reserved for future additive ABI values. Consumers that range-check, bucket, or exhaustively map these codes must tolerate new values appearing in reserved bands.

# Handle format

`pure_simdjson_handle_t` is a packed `uint64_t` with `slot:u32 | generation:u32`.

- `slot` identifies a registry entry.
- `generation` increments when the slot is freed and reused.
- `pure_simdjson_parser_t` and `pure_simdjson_doc_t` are source-level aliases over the same packed `uint64_t`.
- The numeric value `0` is reserved as the invalid sentinel and must never be returned by successful constructors.
- `pure_simdjson_handle_parts_t` is the explicit split view:

```c
typedef struct pure_simdjson_handle_parts_t {
  uint32_t slot;
  uint32_t generation;
} pure_simdjson_handle_parts_t;
```

Bit layout (pinned by the static assertions in `tests/abi/handle_layout.c`):

- `slot` occupies memory bytes `0..4`; `generation` occupies bytes `4..8`.
- On supported little-endian targets this means `slot` is the low 32 bits of the packed `uint64_t` and `generation` the high 32 bits.
- Callers may `memcpy` between an 8-byte handle value and `pure_simdjson_handle_parts_t`, or equivalently extract fields as `(uint32_t)(h)` for `slot` and `(uint32_t)(h >> 32)` for `generation`.

Handles are never raw pointers in the public ABI. Any stale, double-freed, or mismatched generation must fail with `PURE_SIMDJSON_ERR_INVALID_HANDLE` rather than producing undefined behavior.

# Thread safety

- Parsers, documents, value views, and iterators are thread-compatible, not thread-safe.
- One live parser/document graph must be confined to one thread at a time.
- Distinct parsers may be used concurrently as independent graphs.

# Value and iterator model

`pure_simdjson_value_view_t` is a lightweight document-tied view:

```c
typedef struct pure_simdjson_value_view_t {
  pure_simdjson_doc_t doc;
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
- `reserved` must be zero. Non-zero reserved bits are rejected as `PURE_SIMDJSON_ERR_INVALID_HANDLE`.
- `pure_simdjson_array_iter_t` and `pure_simdjson_object_iter_t` are stateful, document-tied iterators driven from Go/C by repeated `*_next` calls.
- Iterator `state0`, `state1`, `index`, and `tag` are implementation-owned state reserved for runtime validation. Iterator `reserved` is pinned for future growth and callers must leave it untouched.
- `pure_simdjson_doc_root`, `pure_simdjson_object_get_field`, `pure_simdjson_array_iter_next`, and `pure_simdjson_object_iter_next` return new view state through out-params rather than allocating child handles.
- `pure_simdjson_object_get_field` returns the first matching field when duplicate keys are present, matching simdjson DOM `object::at_key` semantics.

The value-kind numbers are pinned:

| Kind | Value |
| --- | ---: |
| `PURE_SIMDJSON_VALUE_KIND_INVALID` | 0 |
| `PURE_SIMDJSON_VALUE_KIND_NULL` | 1 |
| `PURE_SIMDJSON_VALUE_KIND_BOOL` | 2 |
| `PURE_SIMDJSON_VALUE_KIND_INT64` | 3 |
| `PURE_SIMDJSON_VALUE_KIND_UINT64` | 4 |
| `PURE_SIMDJSON_VALUE_KIND_FLOAT64` | 5 |
| `PURE_SIMDJSON_VALUE_KIND_STRING` | 6 |
| `PURE_SIMDJSON_VALUE_KIND_ARRAY` | 7 |
| `PURE_SIMDJSON_VALUE_KIND_OBJECT` | 8 |
| `PURE_SIMDJSON_VALUE_KIND_BIGINT` | 9 |

Split numeric access is mandatory:

- `pure_simdjson_element_get_int64` returns a signed integer only.
- `pure_simdjson_element_get_uint64` returns an unsigned integer only.
- `pure_simdjson_element_get_float64` returns a `double` only.
- `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE` covers values that cannot fit the requested integer domain.
- `PURE_SIMDJSON_ERR_PRECISION_LOSS` covers values that cannot be represented without loss in the requested floating/integer conversion path.
- `PURE_SIMDJSON_ERR_WRONG_TYPE` is used when the value kind is not numeric for the requested accessor.

This contract forbids a combined "number union" return or automatic widening that hides overflow or rounding behavior.

BigInt is strict and additive:

- Integer-syntax values below `-9223372036854775808` or above `18446744073709551615` are kind 9. The two boundary values remain int64/uint64, and decimal or exponent syntax remains float64.
- `pure_simdjson_element_get_bigint` accepts only kind 9 and copies the exact decimal spelling.
- `pure_simdjson_element_get_int64`, `pure_simdjson_element_get_uint64`, and `pure_simdjson_element_get_float64` return `PURE_SIMDJSON_ERR_WRONG_TYPE` for BigInt. BigInt never reports `PURE_SIMDJSON_ERR_PRECISION_LOSS`.

# Parser lifecycle

The parser/document state machine is fixed:

1. `pure_simdjson_parser_new` or `pure_simdjson_parser_new_configured` creates a parser handle.
2. `pure_simdjson_parser_parse` parses one input buffer into one `Doc`.
3. `pure_simdjson_doc_root` resolves the root view for that `Doc`.
4. Value accessors and iterators operate while the `Doc` remains live.
5. `pure_simdjson_doc_free` releases the `Doc`.
6. `pure_simdjson_parser_free` releases the parser after all associated documents are gone.

Busy-state rule:

- `pure_simdjson_parser_parse` returns `PURE_SIMDJSON_ERR_PARSER_BUSY` while a live `Doc` exists for that parser.
- `pure_simdjson_parser_free` also returns `PURE_SIMDJSON_ERR_PARSER_BUSY` while a live `Doc` exists for that parser.
- Re-parse never discards, replaces, or mutates the old `Doc` as a side effect.
- Only `pure_simdjson_doc_free` clears the busy state.
- Generation checks remain the mechanism that turns stale parser/doc/view use into `PURE_SIMDJSON_ERR_INVALID_HANDLE`.

The lifecycle is explicit by design so later phases do not introduce hidden reuse semantics.

# Configured parser limits

`pure_simdjson_parser_new` retains the ABI 1.1 behavior and uses the defaults below. `pure_simdjson_parser_new_configured(max_capacity, max_depth, out_parser)` selects immutable limits:

- `max_capacity == 0` means the default `4294967295` (`0xFFFFFFFF`) bytes.
- A non-zero capacity must be in `32..4294967295`; other values return `PURE_SIMDJSON_ERR_INVALID_ARGUMENT`.
- `max_depth == 0` means the default `1024`.
- A non-zero depth must be in `1..1024`; values above `1024` return `PURE_SIMDJSON_ERR_INVALID_ARGUMENT`.
- The effective pair is stored on the parser and reused unchanged by primary DOM parsing and both diagnostic replay passes.
- The same `1024` ceiling bounds the internal fast materializer, so parsing, replay, and materialization share one depth boundary.
- Input longer than the effective capacity returns status 9 before padding arithmetic, arena growth, input allocation, or input copying.
- Input that crosses the effective nesting limit returns status 8.

Limits cannot change after construction. A legacy parser is therefore equivalent to a configured parser constructed with `(0, 0)`.

# Ownership and padding

Every accepted `pure_simdjson_parser_parse` call copies `input_ptr[..input_len]` into Rust-owned padded storage before simdjson sees it.

Rules:

- The public ABI never stores a Go-owned or caller-owned input pointer beyond the parsing call.
- The configured capacity check occurs before the copy or any input-sized allocation.
- The copied buffer must satisfy `SIMDJSON_PADDING` semantics.
- `Doc` lifetime owns the backing parsed storage and any document-tied views/iterators derived from it.
- `Doc` release invalidates all derived `pure_simdjson_value_view_t`, `pure_simdjson_array_iter_t`, and `pure_simdjson_object_iter_t` state.

This choice is part of the ABI 1.2 contract and is not an optimization detail.

# Copied scalars and diagnostics

String and BigInt access are copy-out only.

`pure_simdjson_element_get_string` and `pure_simdjson_element_get_bigint` return:

- `uint8_t **out_ptr`
- `size_t *out_len`

The callee allocates the returned byte buffer, and the caller must release it with `pure_simdjson_bytes_free(ptr, len)`. The bytes remain valid after the document is freed until that explicit release. Borrowed scalar getters are outside this contract.

Diagnostics helpers are part of the ABI surface:

- `pure_simdjson_get_abi_version`
- `pure_simdjson_get_implementation_name_len`
- `pure_simdjson_copy_implementation_name`
- `pure_simdjson_native_alloc_stats_reset`
- `pure_simdjson_native_alloc_stats_snapshot`
- `pure_simdjson_parser_get_last_error_len`
- `pure_simdjson_parser_copy_last_error`
- `pure_simdjson_parser_get_last_error_offset`
- `pure_simdjson_parser_get_last_error_has_offset`

Diagnostics are advisory only:

- Callers must branch on the `pure_simdjson_error_code_t` status code first.
- Diagnostic text and offsets help logging and debugging but do not redefine success/failure.
- `pure_simdjson_parser_copy_last_error` and `pure_simdjson_copy_implementation_name` use bounded caller-provided buffers and may return `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`.
- Callers must read the known bit separately. A proven failure at byte zero is `(offset=0, has_offset=1)`; an unknown location is `(offset=UINT64_MAX, has_offset=0)`. Inferring known/unknown from the numeric offset alone is forbidden.
- A location is known only when upstream provides a non-end pointer proven inside the copied input. No secondary JSON parser, guessed index, or message parsing may manufacture an offset.
- A successful DOM parse performs zero diagnostic replay. An eligible failure performs at most two additional `O(input length)` upstream scans, each using a fresh parser and bounded by the configured capacity and depth. Resource or limit errors stop replay rather than starting an unbounded retry path.

# Process-global implementation selection

Implementation selection is diagnostic and process-global:

- `pure_simdjson_set_implementation(name, name_len)` selects an exact, case-sensitive compiled implementation supported by the current CPU.
- `(name=NULL, name_len=0)` restores upstream automatic selection.
- `pure_simdjson_get_implementation_name_len` and `pure_simdjson_copy_implementation_name` read the selected implementation.
- `pure_simdjson_lock_implementation_selection` irreversibly locks selection and is idempotent.
- The first valid legacy or configured parser construction attempt also locks selection before native allocation.
- Every setter call after the lock returns status 10. There is no per-parser implementation override or unlock operation.

# Native allocator telemetry

The ABI exposes a diagnostic native allocator telemetry surface for benchmark helpers only. It
reports allocations routed through the shim/simdjson cdylib path; it does not claim process-wide
totals or Go heap totals.

```c
typedef struct pure_simdjson_native_alloc_stats_t {
  uint64_t epoch;
  uint64_t live_bytes;
  uint64_t total_alloc_bytes;
  uint64_t alloc_count;
  uint64_t free_count;
  uint64_t untracked_free_count;
} pure_simdjson_native_alloc_stats_t;
```

Rules:

- `pure_simdjson_native_alloc_stats_reset` starts a new telemetry epoch.
- Allocations that were already live at reset time remain valid, but later snapshots exclude them
  from `live_bytes`, `total_alloc_bytes`, `alloc_count`, and `free_count`.
- `epoch` lets callers reject snapshots that straddle a reset, and `untracked_free_count` reports
  frees that did not match the telemetry registry in the current epoch.
- `pure_simdjson_native_alloc_stats_snapshot` writes the current epoch's counters into
  caller-owned `pure_simdjson_native_alloc_stats_t` storage.
- This telemetry surface is diagnostic-only and does not alter parse semantics or the public DOM
  lifecycle contract.

# ABI version handshake

The ABI version export is `pure_simdjson_get_abi_version`.

- The current packed ABI version is `0x00010002`.
- ABI 1.2 is the strict minimum for the Phase 11 wrapper. ABI 1.1 is rejected with an ABI-version mismatch before any ABI 1.2-only lookup.
- ABI 1.2 makes `pure_simdjson_set_implementation`, `pure_simdjson_lock_implementation_selection`, `pure_simdjson_parser_new_configured`, `pure_simdjson_parser_get_last_error_has_offset`, and `pure_simdjson_element_get_bigint` mandatory.
- After the one-symbol version probe succeeds, the loader must bind its complete required surface before cache installation. An artifact claiming compatible ABI 1.2 (or a later additive ABI 1.x) but missing any required symbol is corrupt/incomplete and must fail closed with that symbol named; optional downgrade is forbidden.
- Other ABI majors are incompatible. Later ABI 1.x values may be accepted only when every wrapper-required symbol binds successfully.

The version probe is the only pre-classification symbol. Implementation-name reads occur only after the complete compatible surface binds and must succeed before the loaded library is cached.

# Panic and exception policy

Every exported Rust ABI function must be authored through the `ffi_wrap` helper (`src/lib.rs`), which applies `catch_unwind` in unwind-enabled builds and converts trapped panics into `PURE_SIMDJSON_ERR_PANIC`.

Rules:

- `ffi_wrap` is mandatory for every public export.
- `catch_unwind` is required when unwinding is enabled so Rust panics do not cross the C ABI boundary.
- The ABI 1.2 build policy pins `panic = "abort"` in the dev and release Cargo profiles. Cargo ignores that setting for the `test` profile, so unwind-enabled test builds still require the `ffi_wrap`/`catch_unwind` boundary above to convert internal panics into `PURE_SIMDJSON_ERR_PANIC`.
- The Rust/C++ seam must use non-throwing simdjson access patterns such as `.get(err)`.
- C++ exceptions must be trapped before re-entering Rust and converted into `PURE_SIMDJSON_ERR_CPP_EXCEPTION`.
- No foreign exception or Rust unwind may cross into Go or C callers.

This section is normative even though the full shim implementation lands in later phases.

# Worked call sequences

## String copy and free

```c
pure_simdjson_parser_t parser = 0;
pure_simdjson_doc_t doc = 0;
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
pure_simdjson_parser_t parser = 0;
pure_simdjson_doc_t doc_a = 0;
pure_simdjson_doc_t doc_b = 0;

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
