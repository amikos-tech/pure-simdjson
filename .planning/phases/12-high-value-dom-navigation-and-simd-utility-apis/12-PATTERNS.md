# Phase 12: High-value DOM navigation and SIMD utility APIs - Pattern Map

**Mapped:** 2026-07-30
**Files analyzed:** 20 (11 modified + 9 new, across Go/Rust/C++/contract/test layers)
**Analogs found:** 18 / 20 (2 files — the Go `AtPathAll` bulk-`[]Element` transport and the C++
wildcard scratch-vector transport — have no exact existing analog; closest partial analogs given)

Every new capability in this phase is a variation on a pattern that already exists and works in
this exact codebase. Nothing here needs external research — the job is "which existing function do
I clone." All excerpts below are read directly from the current repo (not RESEARCH.md's illustrative
pseudo-code, except where explicitly marked, since RESEARCH.md's own examples are themselves already
*modeled on* the excerpts given here).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `element.go` — `Element.AtPointer`/`AtPath` | controller (Go public accessor) | CRUD (single lookup) | `element.go` `Object.GetField` (L369-392) | exact |
| `element.go` — `Element.AtPathAll` | controller (Go public accessor) | batch | `element.go` `Object.GetField` (single) + `iterator.go` `ArrayIter.Next` (ordered sequence) — no exact `[]Element`-returning precedent | partial (see note) |
| `element.go` — `Array.At` | controller (Go public accessor) | CRUD (single lookup) | `element.go` `Object.GetField` (L369-392) — same "no panic-safe twin" shape | exact |
| `element.go` — `Array.Len`/`LenErr`, `Object.Size`/`SizeErr` | controller (Go public accessor) | CRUD (scalar read) | `element.go` `Type()`/`TypeErr()` (L101-148), `IsNull()`/`IsNullErr()` (L261-285) | exact |
| `errors.go` — `ErrInvalidPath`, `ErrIndexOutOfRange` sentinels + `sentinelForStatus` cases | utility (error mapping) | transform | `errors.go` `sentinelForStatus` (L222-259), existing sentinel block (L11-54) | exact |
| `minify.go` (NEW) — `Minify`, `MinifyInto` | utility (standalone entry point) | transform (buffer-in/buffer-out) | `kernel.go` `SetKernel` (L36-69) + `parser.go` `newParserWithConfig` (L40-62) | exact |
| `utf8.go` (NEW) — `ValidateUTF8` | utility (standalone entry point) | request-response | `kernel.go` `SetKernel` (L36-69) — `activeLibrary()`-first, error-returning standalone function | exact |
| `internal/ffi/types.go` — 2 new `ErrorCode` consts, ABI bump | config | transform | `internal/ffi/types.go` existing `ErrorCode` block (L11-38) | exact |
| `internal/ffi/bindings.go` — 9 new binding fields + required registration + typed wrapper methods | service (symbol-binding adapter) | request-response | `internal/ffi/bindings.go` `elementGetBigInt`/`ElementGetBigInt` (Phase 11 addition, L42,111,381-383) + `ObjectGetField` (L500-513) | exact |
| `src/lib.rs` — 9 new `pure_simdjson_*` `extern "C"` exports | controller (FFI entry point) | request-response (batch for wildcard, transform for minify) | `src/lib.rs` `pure_simdjson_object_get_field` (L966-998) + `pure_simdjson_bytes_free` (L826-834) | exact |
| `src/runtime/mod.rs` — 9 new `psimdjson_*` extern declarations + thin wrappers | service (thin native-call wrapper) | request-response | `src/runtime/mod.rs` `native_object_get_field_index` (L666-689) | exact |
| `src/runtime/registry.rs` — `element_at_pointer`/`at_path`/`at_path_wildcard`, `array_at`, `array_len`, `object_size` | service (handle validation + business logic) | CRUD (single) / batch (wildcard) | `src/runtime/registry.rs` `object_get_field` (L1154-1168), `with_resolved_view` (L731-766), `encode_descendant_view_locked` (L768-786) | exact |
| `src/runtime/registry.rs` — new `view_array_allocations` map + free fn | service (tracked-allocation ownership) | batch | `src/runtime/registry.rs` `byte_allocations` + `bytes_free` (L86, L895-927) | exact |
| `src/native/simdjson_bridge.h` — 9 new `psimdjson_*` declarations | config (bridge header) | transform | `src/native/simdjson_bridge.h` `psimdjson_object_get_field_index` (L189-195) | exact |
| `src/native/simdjson_bridge.cpp` — new bridge fns; `map_error()` +2 cases; `dst_cap` pre-check for minify; CPU-gate for minify/validate_utf8 | service (native implementation) | CRUD/batch/transform (per fn) | `src/native/simdjson_bridge.cpp` `psimdjson_object_get_field_index` (L1508-1541), `copy_bytes` (L125-150), `parser_new_configured_with_selection_lock` (L716-758) | exact |
| `src/native/simdjson_bridge.cpp` — `psimdjson_doc` gains `wildcard_indices`/`wildcard_in_progress` | model + service (scratch-buffer struct + guard) | batch | `src/native/simdjson_bridge.cpp` `psimdjson_doc.materialize_frames`/`materialize_in_progress` (L68-78) + `materialize_build_guard` (L799-824) | exact |
| `include/pure_simdjson.h` (regenerated) | config (generated header) | transform | existing `pure_simdjson_object_get_field`/`pure_simdjson_element_get_bigint` entries (L454-456, L540-543) | exact (no manual edits — `cbindgen` regenerates) |
| `docs/ffi-contract.md` — error codes 11/12, ABI 1.3 section, worked-call-sequence for view-array free | config (normative contract doc) | transform | `docs/ffi-contract.md` "String copy and free" worked sequence (L296-315), Error code space table (L31-57) | exact |
| `tests/abi/check_header.py` — `REQUIRED_SYMBOLS` +9 names | test (ABI allowlist) | transform | `tests/abi/check_header.py` `REQUIRED_SYMBOLS` tuple (L48-80) | exact |
| `tests/rust_shim_navigation.rs` (NEW) | test (Rust integration) | request-response/batch | `tests/rust_shim_accessors.rs` (L1-49 harness) + `tests/rust_shim_iterators.rs` (object-field-lookup tests) | exact |
| `tests/rust_shim_minify.rs` (NEW) | test (Rust integration) | transform | `tests/rust_shim_accessors.rs` (L1-49 harness — same parser/doc setup, swap accessor under test) | exact |
| Go `*_test.go` (extend `element_scalar_test.go` or new `element_pointer_test.go`/`array_test.go`/`minify_test.go`/`utf8_test.go`) | test (Go table test) | request-response | `element_scalar_test.go` `TestElementTypeClassification` (L40-85) | exact |
| `.github/workflows/phase2-rust-shim-smoke.yml` — durable D-14 x86-64 gate | config (CI) | n/a | existing `linux-smoke` job steps (not excerpted; add path filters and a step invoking `scripts/ci/verify_minify_buffer_safety.sh`, whose compiled probe lives under `tests/native/`) | exact |

**Note on the two "partial" matches:** `AtPathAll`'s `[]Element` bulk return and the C++ wildcard
scratch-vector transport are genuinely new shapes in this codebase (RESEARCH.md's own Architecture
section confirms this explicitly — "the one genuinely new transport shape"). There is no regression
here: RESEARCH.md's "Code Examples" section (lines 354-463 of `12-RESEARCH.md`) already designed the
exact shape to use, modeled directly on the analogs listed below (`materialize_build_guard` +
`bytes_free`/`byte_allocations`). Treat those RESEARCH.md snippets as the wildcard-specific design,
and the excerpts in this file as the concrete existing code they were modeled on.

---

## Pattern Assignments

### Layer 1 — Go public API accessor methods (`element.go`)

**Analog:** `Object.GetField` (no panic-safe twin) and `Type()`/`TypeErr()` (panic-safe dual method)

**The `usableDoc()` guard, used by every accessor including every new one** (`element.go` L69-80):
```go
func (e Element) usableDoc() (*Doc, error) {
	if e.doc == nil {
		return nil, ErrInvalidHandle
	}
	if e.doc.parser == nil || e.doc.parser.library == nil || e.doc.parser.library.bindings == nil {
		return nil, ErrInvalidHandle
	}
	if e.doc.isClosed() {
		return nil, ErrClosed
	}
	return e.doc, nil
}
```

**D-04 precedent — `Array.At` must mirror `Object.GetField`'s no-panic-safe-twin shape exactly**
(`element.go` L369-392):
```go
func (o Object) GetField(key string) (Element, error) {
	doc, err := o.element.usableDoc()
	if err != nil {
		return Element{}, err
	}
	if ffi.ValueKind(o.element.view.KindHint) != ffi.ValueKindObject {
		return Element{}, ErrWrongType
	}

	view, rc := doc.parser.library.bindings.ObjectGetField(&o.element.view, key)
	runtime.KeepAlive(doc)
	if err := normalizeIteratorError(doc, rc); err != nil {
		return Element{}, err
	}

	return Element{doc: doc, view: view}, nil
}
```
`Array.At(index int) (Element, error)` copies this shape 1:1: `usableDoc()` guard, kind-check
(`ffi.ValueKindArray`), call the new `bindings.ArrayAt(&a.element.view, uint64(index))` wrapper,
`wrapStatus(rc)` (plain `wrapStatus`, not `normalizeIteratorError`, since `At` is not an
iterator-lease call), return `Element{doc: doc, view: view}`.

**D-06 precedent — `Array.Len`/`LenErr` and `Object.Size`/`SizeErr` must mirror the
`Type()`/`TypeErr()` dual-method shape** (`element.go` L101-148, abbreviated to the shape that
matters):
```go
func (e Element) Type() ElementType {
	kind, err := e.TypeErr()
	if err != nil {
		return TypeInvalid
	}
	return kind
}

func (e Element) TypeErr() (ElementType, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return TypeInvalid, err
	}
	kind, rc := doc.parser.library.bindings.ElementType(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return TypeInvalid, err
	}
	// ... classify kind ...
}
```
`Array.LenErr() (int, error)` follows the `TypeErr()` shape (usableDoc -> bindings call ->
`wrapStatus` -> cast native `uint64`/`uint32` count to `int`, per D-05's established
"cast native unsigned counts to `int` at Go boundaries" convention — see the `int(frame.ChildCount)`
precedent in `materializer_fastpath.go` L172/187). `Array.Len() int` follows the `Type()` shape:
call `LenErr()`, return `0` on any error.

**AsArray/AsObject kind-check precedent, useful for the `Object.Size` analog** (`element.go`
L287-311) — same `usableDoc()` + `ffi.ValueKind(e.view.KindHint) != ffi.ValueKindX` -> `ErrWrongType`
shape reused by every typed accessor.

---

### Layer 1b — `AtPathAll`'s Go public surface (no exact existing analog — nearest building blocks)

**Nearest existing shapes:** `Object.GetField` (single-result call+wrap) and `ArrayIter.Next`
(`iterator.go` L33-61, ordered per-element `doc`-tied results). Neither returns `[]Element`
directly; RESEARCH.md's own designed snippet (`12-RESEARCH.md` lines 441-463) is the shape to
implement, built by composing these two known-good pieces: one bindings call returning a Go slice of
`ffi.ValueView` (analogous to a single `ObjectGetField` call, but N-wide), then a Go-side loop
wrapping each into `Element{doc: doc, view: v}` (the same construction `Object.GetField`'s return
statement already does per-element). D-02's "empty match is `[]Element{}, nil`, never `nil, nil`"
requirement is satisfied by `make([]Element, len(views))` — a zero-length non-nil slice when
`len(views) == 0`, exactly the semantics of `filepath.Glob`/map-lookup idioms the decision cites.

---

### Layer 2 — Standalone Go functions with no receiver (`minify.go`, `utf8.go`)

**Analog:** `kernel.go` `SetKernel` (activeLibrary()-first, error-returning, no Doc/Parser) and
`parser.go` `newParserWithConfig` (CPU-unsupported rejection + kernel-lock-on-success)

**`SetKernel` — the exact `activeLibrary()`-first shape `Minify`/`MinifyInto`/`ValidateUTF8` must
copy** (`kernel.go` L39-69):
```go
func SetKernel(name string) error {
	kernelMu.Lock()
	defer kernelMu.Unlock()

	if kernelSelectionLocked {
		return ErrKernelLocked
	}

	library, err := activeLibrary()
	if err != nil {
		return err
	}

	rc := library.bindings.SetImplementation(name)
	if statusErr := wrapStatus(rc); statusErr != nil {
		if rc == int32(ffi.ErrInvalidArg) {
			return fmt.Errorf("%w: %v", ErrInvalidOption, statusErr)
		}
		return statusErr
	}
	// ...
}
```

**`newParserWithConfig` — the CPU-unsupported rejection + `lockKernelSelection()`-on-success shape
D-15 requires `Minify`/`MinifyInto`/`ValidateUTF8` to replicate** (`parser.go` L40-62):
```go
func newParserWithConfig(config parserConfig) (*Parser, error) {
	kernelMu.Lock()
	defer kernelMu.Unlock()

	library, err := activeLibrary()
	if err != nil {
		return nil, err
	}

	handle, rc := library.bindings.ParserNewConfigured(config.maxCapacity, config.maxDepth)
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}
	kernelSelectionLocked = true
	// ...
}
```
Here the CPU-unsupported check (`PURE_SIMDJSON_ERR_CPU_UNSUPPORTED`) happens natively inside
`ParserNewConfigured`'s C++ implementation (see Layer 5 below) — `wrapStatus(rc)` already turns that
into `ErrCPUUnsupported` via the existing `sentinelForStatus` case (`errors.go` L246-247). The new
`Minify`/`MinifyInto`/`ValidateUTF8` Go functions need the *identical* two-step shape: acquire
`kernelMu`, call `activeLibrary()`, call the new bindings method, and on `PURE_SIMDJSON_OK` set
`kernelSelectionLocked = true` — no new Go-side CPU check needed, since the native layer (Layer 5)
is where D-15's gate actually lives, mirroring `parser_new_configured_with_selection_lock`.

**`activeLibrary()` — the resolution path itself** (`library_loading.go` L57-68), unchanged, reused
verbatim by the three new standalone functions as the first-ever non-`NewParser` caller of this
function.

---

### Layer 3 — purego symbol binding (`internal/ffi/bindings.go`, `internal/ffi/types.go`)

**Analog:** `pure_simdjson_element_get_bigint` / `ElementGetBigInt` — the most recent (Phase 11)
required-symbol addition, structurally identical to what the 9 new Phase 12 exports need.

**Struct field + required-symbol table entry** (`internal/ffi/bindings.go` L42, L111):
```go
elementGetBigInt         func(*ValueView, **byte, *uintptr) int32
// ...
{name: "pure_simdjson_element_get_bigint", target: &b.elementGetBigInt},
```
This confirms the required (not optional) registration path — DOM-01..04/UTIL-01/02 are public
documented API, so all 9 new symbols go in the mandatory `symbols := []struct{...}{...}` slice
(L87-120), never through `registerOptionalFuncWithRegistrar` (that path is reserved for
`pure_simdjson_native_alloc_stats_*`/`psdj_internal_materialize_build` — internal diagnostics
allowed to be missing from released artifacts, per RESEARCH.md's explicit Anti-Pattern warning).

**Typed wrapper method — byte-copy-returning shape (`ElementGetBigInt`, `GetString`'s sibling)**
(`internal/ffi/bindings.go` L381-409):
```go
func (b *Bindings) ElementGetBigInt(view *ValueView) (string, int32) {
	return b.copyElementBytes(view, b.elementGetBigInt)
}

func (b *Bindings) copyElementBytes(view *ValueView, getter func(*ValueView, **byte, *uintptr) int32) (string, int32) {
	var ptr *byte
	var length uintptr
	rc := getter(view, &ptr, &length)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	if rc != int32(OK) {
		return "", rc
	}
	defer func() {
		if ptr == nil {
			return
		}
		if freeRC := b.BytesFree(ptr, length); freeRC != int32(OK) {
			emitBytesFreeFailureWarning(freeRC, length)
		}
	}()
	// ...
}
```

**Typed wrapper method — view-returning shape with a string key argument (`ObjectGetField`, the
direct analog for `AtPointer`/`AtPath`)** (`internal/ffi/bindings.go` L500-513):
```go
func (b *Bindings) ObjectGetField(view *ValueView, key string) (ValueView, int32) {
	var keyBytes []byte
	if key != "" {
		keyBytes = []byte(key)
	}
	var keyPtr *byte
	if len(keyBytes) != 0 {
		keyPtr = unsafe.SliceData(keyBytes)
	}

	var value ValueView
	rc := b.objectGetField(view, keyPtr, uintptr(len(keyBytes)), &value)
	runtime.KeepAlive(keyBytes)
	// ...
}
```
`ElementAtPointer(view *ValueView, pointer string) (ValueView, int32)` and
`ElementAtPath(view *ValueView, path string) (ValueView, int32)` copy this exact shape (string ->
byte-slice -> pointer/len pair -> out-param `ValueView`). `ArrayAt(view *ValueView, index uint64)
(ValueView, int32)` is simpler still (no string marshaling, just an extra scalar argument like
`ParserNewConfigured(uint64, uint32, *ParserHandle)`, `internal/ffi/bindings.go` L27).

**`internal/ffi/types.go` — additive status-code and ABI-version block** (L3-9, L13-38): add
`ErrInvalidPath ErrorCode = 11` and `ErrIndexOutOfRange ErrorCode = 12` immediately after
`ErrKernelLocked ErrorCode = 10` (matching RESEARCH.md Assumption A1 and the file's own comment
"Values in this block must stay in lockstep with pure_simdjson.h and src/lib.rs"); bump
`ABIVersion uint32 = 0x00010003` (was `0x00010002`), matching the exact `0x00010001 ->
0x00010002` precedent this same constant went through in Phase 11.

---

### Layer 4 — Rust `extern "C"` export shape (`src/lib.rs`)

**Analog:** `pure_simdjson_object_get_field` (single-result, out-param) and `pure_simdjson_bytes_free`
(tracked-allocation free)

**`ffi_wrap`/`catch_unwind` shape, mandatory for every export** (`src/lib.rs` L193-209):
```rust
fn ffi_wrap<F>(function_name: &'static str, body: F) -> pure_simdjson_error_code_t
where
    F: FnOnce() -> pure_simdjson_error_code_t,
{
    match catch_unwind(AssertUnwindSafe(body)) {
        Ok(rc) => rc,
        Err(payload) => {
            eprintln!(
                "pure_simdjson panic in {}: {}",
                function_name,
                panic_payload_message(payload.as_ref())
            );
            err_panic()
        }
    }
}
```

**Out-parameter + int32-status-code convention, single-result shape to copy for
`pure_simdjson_element_at_pointer`/`_at_path`/`_array_at`** (`src/lib.rs` L966-998):
```rust
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key_ptr: *const u8,
    key_len: usize,
    out_value: *mut pure_simdjson_value_view_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_object_get_field", || unsafe {
        if out_value.is_null() {
            return err_invalid_argument();
        }
        if key_len != 0 && key_ptr.is_null() {
            return err_invalid_argument();
        }
        let key = if key_len == 0 { &[][..] } else { slice::from_raw_parts(key_ptr, key_len) };
        match runtime::registry::object_get_field(object_view, key) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
    })
}
```

**CPU-unsupported gate + first-call registration prologue** (`src/lib.rs` L255-265, used by
`pure_simdjson_parser_new` L448-465):
```rust
#[inline]
fn reject_fallback_implementation() -> Result<(), pure_simdjson_error_code_t> {
    let forced_implementation = runtime::forced_implementation_name_for_parser_new();
    if matches!(forced_implementation.as_deref(), Some(b"fallback"))
        && !runtime::fallback_allowed_for_tests()
    {
        return Err(err_cpu_unsupported());
    }
    Ok(())
}
// usage:
if let Err(rc) = reject_fallback_implementation() {
    return rc;
}
```
D-15's `pure_simdjson_minify`/`pure_simdjson_validate_utf8` need this same
`reject_fallback_implementation()` call inserted before the registry/bridge call — see Layer 5 for
where the *authoritative* CPU-unsupported check actually lives (the C++
`ImplementationSelectionState`), since this Rust-side helper is a test-forcing wrapper around the
same policy, not a second independent check.

**Tracked-allocation free pair — the analog for the new `pure_simdjson_value_views_free`**
(`src/lib.rs` L826-834):
```rust
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_bytes_free(
    ptr: *mut u8,
    len: usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_bytes_free", || {
        runtime::registry::bytes_free(ptr, len)
    })
}
```
paired with the registry-side tracked-allocation table (`src/runtime/registry.rs` L86, L895-927):
```rust
struct Registry {
    parsers: Vec<Slot<ParserEntry>>,
    docs: Vec<Slot<DocEntry>>,
    byte_allocations: HashMap<usize, usize>,   // NEW: add view_array_allocations: HashMap<usize, usize>
}

pub(crate) fn bytes_free(ptr: *mut u8, len: usize) -> pure_simdjson_error_code_t {
    if ptr.is_null() {
        return if len == 0 { err_ok() } else { err_invalid_argument() };
    }
    if len == 0 {
        return err_invalid_argument();
    }
    {
        let mut registry = registry_guard();
        match registry.byte_allocations.remove(&(ptr as usize)) {
            Some(registered_len) if registered_len == len => {}
            Some(registered_len) => {
                registry.byte_allocations.insert(ptr as usize, registered_len);
                return err_invalid_handle();
            }
            None => return err_invalid_handle(),
        }
    }
    unsafe { drop(Vec::from_raw_parts(ptr, len, len)); }
    err_ok()
}
```
`pure_simdjson_value_views_free(ptr: *mut pure_simdjson_value_view_t, len: usize)` copies this
exactly, against a new `view_array_allocations: HashMap<usize, usize>` map (storing element *count*,
not byte length, per RESEARCH.md Open Question 2's recommendation) rather than overloading
`byte_allocations`' units.

---

### Layer 5 — C++ bridge (`src/native/simdjson_bridge.h` / `.cpp`)

**Analog:** `copy_bytes` (dst_cap pre-check — the exact model RESEARCH.md's Pattern 2 already
names for `psimdjson_minify`) and `psimdjson_object_get_field_index` (resolve-to-`json_index`
shape) and `parser_new_configured_with_selection_lock` (CPU-unsupported gate + lock)

**`copy_bytes` — the dst_cap pre-check `psimdjson_minify` must replicate, since
`simdjson::minify(buf, len, dst, dst_len)` has no `dst_cap` parameter of its own**
(`src/native/simdjson_bridge.cpp` L125-150):
```cpp
pure_simdjson_error_code_t copy_bytes(
    std::string_view src, uint8_t *dst, size_t dst_cap, size_t *out_written
) noexcept {
  if (out_written == nullptr) return invalid_argument();
  *out_written = src.size();
  if (src.size() > dst_cap) return PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL;
  if (!src.empty() && dst == nullptr) return invalid_argument();
  if (!src.empty()) std::memcpy(dst, src.data(), src.size());
  return PURE_SIMDJSON_OK;
}
```
`psimdjson_minify(src_ptr, src_len, dst_ptr, dst_cap, out_written)` must pre-check
`dst_cap >= src_len` **before** calling `simdjson::minify` at all (RESEARCH.md Pattern 2 spells out
the full function; `copy_bytes` above is the existing pattern it copies).

**`psimdjson_object_get_field_index` — resolve-to-`json_index` shape, the model for
`psimdjson_element_at_pointer_index`/`_at_path_index`/`_array_at_index`**
(`src/native/simdjson_bridge.cpp` L1508-1541):
```cpp
pure_simdjson_error_code_t psimdjson_object_get_field_index(
    const psimdjson_doc *doc, uint64_t json_index,
    const uint8_t *key_ptr, size_t key_len,
    uint64_t *out_value_json_index
) noexcept {
  try {
    if (doc == nullptr || out_value_json_index == nullptr) return invalid_argument();
    if (key_len != 0 && key_ptr == nullptr) return invalid_argument();

    simdjson::dom::object object;
    const auto object_error = element_at(doc, json_index).get_object().get(object);
    if (object_error != simdjson::SUCCESS) return map_error(object_error);

    const auto key = key_len == 0 ? std::string_view{}
        : std::string_view(reinterpret_cast<const char *>(key_ptr), key_len);
    simdjson::dom::element value;
    const auto field_error = object.at_key(key).get(value);
    if (field_error != simdjson::SUCCESS) return map_error(field_error);

    *out_value_json_index = element_json_index(value);
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}
```
`psimdjson_element_at_pointer_index` swaps `object.at_key(key)` for
`element_at(doc, json_index).at_pointer(pointer)`; `psimdjson_array_at_index` swaps it for
`element_at(doc, json_index).get_array()` + `array.at(index)`. `map_error()` (`simdjson_bridge.cpp`
L152 onward) needs 2 new `switch` cases: `simdjson::INVALID_JSON_POINTER ->
PURE_SIMDJSON_ERR_INVALID_PATH` and `simdjson::INDEX_OUT_OF_BOUNDS ->
PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE` (both currently fall through to
`PURE_SIMDJSON_ERR_INTERNAL`, per RESEARCH.md's direct confirmation).

**CPU-unsupported gate + selection-lock — the authoritative check `psimdjson_minify`/
`psimdjson_validate_utf8` must call, mirroring parser construction exactly (D-15)**
(`src/native/simdjson_bridge.cpp` L110-119, L716-758):
```cpp
struct ImplementationSelectionState {
  std::mutex mutex{};
  bool locked{false};
  bool explicit_selection{false};
};
ImplementationSelectionState &implementation_selection_state() {
  static ImplementationSelectionState state;
  return state;
}

pure_simdjson_error_code_t parser_new_configured_with_selection_lock(/* ... */) {
  // ...
  auto &selection = implementation_selection_state();
  const std::lock_guard<std::mutex> lock(selection.mutex);
  selection.locked = true;

  const simdjson::implementation *implementation = simdjson::get_active_implementation();
  if (!selection.explicit_selection && implementation->name() == "fallback") {
    return PURE_SIMDJSON_ERR_CPU_UNSUPPORTED;
  }
  // ... proceed ...
}
```
`psimdjson_minify`/`psimdjson_validate_utf8` open with the identical
`implementation_selection_state()` lock-and-check block (locking selection and rejecting on
unsupported fallback) before doing any real work — this is the literal mechanism behind D-15's
"call the `reject_fallback_implementation()`-equivalent check and lock kernel selection."

**`psimdjson_doc` scratch-buffer + guard pattern — the analog for the wildcard result transport**
(`src/native/simdjson_bridge.cpp` L68-78, L799-824):
```cpp
struct psimdjson_doc {
  simdjson::dom::document document{};
  psimdjson_element root{};
  std::vector<psdj_internal_frame_t> materialize_frames{};   // NEW: std::vector<uint64_t> wildcard_indices{};
  bool materialize_in_progress{false};                        // NEW: bool wildcard_in_progress{false};
};

class materialize_build_guard {
 public:
  explicit materialize_build_guard(psimdjson_doc *doc) noexcept : doc_(doc) {
    if (doc_ != nullptr && !doc_->materialize_in_progress) {
      doc_->materialize_in_progress = true;
      acquired_ = true;
    }
  }
  ~materialize_build_guard() noexcept {
    if (acquired_) { doc_->materialize_in_progress = false; }
  }
  bool acquired() const noexcept { return acquired_; }
 private:
  psimdjson_doc *doc_{nullptr};
  bool acquired_{false};
};
```
`psimdjson_element_at_path_wildcard_indices` adds a `wildcard_build_guard` of the identical shape
guarding `doc->wildcard_indices` instead of `doc->materialize_frames` — see RESEARCH.md's full
worked function (`12-RESEARCH.md` L363-398) for the complete new-function body; this excerpt is the
*existing* struct/guard it is modeled on.

---

### Layer 6 — Descendant `Element` construction (`src/runtime/registry.rs`)

**Analog:** `object_get_field` + `with_resolved_view` + `encode_descendant_view_locked` — this
triple is the exact "resolve-then-register" pattern every single-result navigation method
(`AtPointer`, `AtPath`, `Array.At`) must replicate, and `encode_descendant_view_locked` specifically
is RESEARCH.md's named reuse target for turning each wildcard match index into a registered
`ValueView`.

**`with_resolved_view` — validates the input handle (generation, state tag, descendant-index
membership) before any native call; every new registry function wraps its body in this**
(`src/runtime/registry.rs` L731-766, full function already read — key excerpt):
```rust
fn with_resolved_view<T, F>(
    view: *const pure_simdjson_value_view_t,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(&mut DocEntry, u64, pure_simdjson_doc_t) -> Result<T, pure_simdjson_error_code_t>,
{
    if view.is_null() { return Err(err_invalid_argument()); }
    let view = unsafe { ptr::read_unaligned(view) };
    if view.state0 == 0 || view.reserved != 0 { return Err(err_invalid_handle()); }

    let (doc_index, _, doc_generation) = unpack_handle(view.doc)?;
    let mut registry = registry_guard();
    let entry = match registry.docs.get_mut(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => entry,
        _ => return Err(err_invalid_handle()),
    };
    let json_index = match view.state1 {
        ROOT_VIEW_TAG => { /* ... */ ROOT_JSON_INDEX }
        DESC_VIEW_TAG => validate_descendant(&view, entry)?,
        _ => return Err(err_invalid_handle()),
    };
    action(entry, json_index, view.doc)
}
```

**`encode_descendant_view_locked` — registers a new native `json_index` as a lifetime-tracked
descendant view; this is the exact function RESEARCH.md names for wrapping each `AtPathAll` match**
(`src/runtime/registry.rs` L768-786):
```rust
fn encode_descendant_view_locked(
    entry: &mut DocEntry,
    handle: pure_simdjson_doc_t,
    json_index: u64,
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    if json_index == 0 || json_index >= entry.root_after_index {
        return Err(err_invalid_handle());
    }
    let kind_hint = super::native_element_type_at(entry.native_ptr, json_index)?;
    entry.descendant_indices.insert(json_index);

    Ok(pure_simdjson_value_view_t {
        doc: handle,
        state0: json_index,
        state1: DESC_VIEW_TAG,
        kind_hint,
        reserved: 0,
    })
}
```

**`object_get_field` — the complete single-result composition to copy verbatim for
`element_at_pointer`/`element_at_path`/`array_at`** (`src/runtime/registry.rs` L1154-1168):
```rust
pub(crate) fn object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key: &[u8],
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    with_resolved_view(object_view, |entry, json_index, doc| {
        let kind = super::native_element_type_at(entry.native_ptr, json_index)?;
        if kind != KIND_HINT_OBJECT {
            return Err(err_wrong_type());
        }
        let value_json_index =
            super::native_object_get_field_index(entry.native_ptr, json_index, key)?;
        encode_descendant_view_locked(entry, doc, value_json_index)
    })
}
```
`element_at_pointer`/`element_at_path` drop the `kind != KIND_HINT_OBJECT` pre-check (upstream
`at_pointer`/`at_path` dispatch on kind internally and work on scalars when the pointer/path is
empty, per RESEARCH.md Pattern 1); `array_at` keeps an analogous `kind != KIND_HINT_ARRAY` pre-check
(compare `array_iter_new`'s existing `KIND_HINT_ARRAY` check, `src/runtime/registry.rs` L1003-1008).

**Thin native-call wrapper — `native_object_get_field_index`, the model for
`native_element_at_pointer_index`/`native_array_at_index`** (`src/runtime/mod.rs` L666-689):
```rust
pub(crate) fn native_object_get_field_index(
    doc_ptr: usize, json_index: u64, key: &[u8],
) -> Result<u64, pure_simdjson_error_code_t> {
    let key_ptr = key.as_ptr();
    let mut value_json_index = 0_u64;
    let rc = unsafe {
        psimdjson_object_get_field_index(
            doc_ptr as *const psimdjson_doc, json_index, key_ptr, key.len(), &mut value_json_index,
        )
    };
    if rc != err_ok() { return Err(rc); }
    if value_json_index == 0 { return Err(err_internal()); }
    Ok(value_json_index)
}
```

---

### Layer 7 — Tests

**Analog for `tests/rust_shim_navigation.rs`/`tests/rust_shim_minify.rs`:** `tests/rust_shim_accessors.rs`
and `tests/rust_shim_iterators.rs` — both establish the identical harness shape: import the specific
`pure_simdjson_*` symbols under test from the crate root, define `parser_new()` /
`parser_parse_literal()` / `doc_root()` / `cleanup()` helpers, then one `#[test] fn` per behavior
asserting on the raw `pure_simdjson_error_code_t` (`tests/rust_shim_accessors.rs` L1-49, L98-116):
```rust
use pure_simdjson::{
    pure_simdjson_bytes_free, pure_simdjson_doc_free, pure_simdjson_doc_root, pure_simdjson_doc_t,
    pure_simdjson_element_get_bool, /* ... */
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        PURE_SIMDJSON_ERR_PRECISION_LOSS, PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_parse,
    pure_simdjson_parser_t, pure_simdjson_value_view_t,
};

fn parser_new() -> pure_simdjson_parser_t { /* alloc + assert PURE_SIMDJSON_OK */ }
fn parser_parse_literal(parser: pure_simdjson_parser_t, json: &[u8]) -> pure_simdjson_doc_t { /* ... */ }
fn doc_root(doc: pure_simdjson_doc_t) -> pure_simdjson_value_view_t { /* ... */ }
fn cleanup(parser: pure_simdjson_parser_t, doc: pure_simdjson_doc_t) { /* ... */ }

#[test]
fn bytes_free_round_trip_releases_string_buffer() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#""hello""#);
    let root = doc_root(doc);
    let mut ptr = ptr::null_mut();
    let mut len = 0_usize;
    let rc = unsafe { pure_simdjson_element_get_string(&root, &mut ptr, &mut len) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(!ptr.is_null());
    assert_eq!(unsafe { slice::from_raw_parts(ptr, len) }, b"hello");
    assert_eq!(unsafe { pure_simdjson_bytes_free(ptr, len) }, PURE_SIMDJSON_OK);
    cleanup(parser, doc);
}
```
`rust_shim_navigation.rs` follows this exactly for `pure_simdjson_object_get_field`-style calls
against the 6 new nav/index/size exports (swap in `PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE`/
`PURE_SIMDJSON_ERR_INVALID_PATH` where relevant). `rust_shim_minify.rs` follows the same harness for
overlap (`dst == src`), empty-input, malformed (`UNCLOSED_STRING`), and undersized-`dst_cap` cases
against `pure_simdjson_minify`.

**Analog for Go table tests** (`element_scalar_test.go` `TestElementTypeClassification`, L40-65):
```go
func TestElementTypeClassification(t *testing.T) {
	testCases := []struct {
		name string
		json string
		want ElementType
	}{
		{name: "int64", json: "42", want: TypeInt64},
		// ...
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)
			if got := doc.Root().Type(); got != tc.want {
				t.Fatalf("Type() = %v, want %v", got, tc.want)
			}
		})
	}
}
```
Every new Go-level test (`TestElement_AtPointer`, `TestElement_AtPath`, `TestElement_AtPathAll`,
`TestArray_At`, `TestArray_Len`, `TestObject_Size`, `TestMinify`, `TestMinifyInto`,
`TestValidateUTF8`) follows this `testCases := []struct{...}{}` + `t.Run(tc.name, ...)` shape, using
the existing `mustParseDoc` (`element_scalar_test.go` L17) / `mustNewParser` (`parser_test.go` L19) helpers already defined in this package's test files.

---

### Layer 8 — ABI/contract plumbing

**`tests/abi/check_header.py` `REQUIRED_SYMBOLS` — closed allowlist, must gain exactly the 9 new
names or `make verify-contract` fails closed** (`tests/abi/check_header.py` L48-80):
```python
REQUIRED_SYMBOLS = (
    "pure_simdjson_get_abi_version",
    # ...
    "pure_simdjson_element_get_bigint",
    "pure_simdjson_bytes_free",
    # ...
    "pure_simdjson_object_get_field",
)
```
Append `pure_simdjson_element_at_pointer`, `pure_simdjson_element_at_path`,
`pure_simdjson_element_at_path_wildcard`, `pure_simdjson_value_views_free`,
`pure_simdjson_array_at`, `pure_simdjson_array_len`, `pure_simdjson_object_size`,
`pure_simdjson_minify`, `pure_simdjson_validate_utf8` (exact final names are Claude's discretion per
CONTEXT.md, but each new required Rust export must land here in lockstep).

**`include/pure_simdjson.h` — generated, do not hand-edit; shape to expect after
`make generate-header`** (existing entries, L454-456, L540-543):
```c
pure_simdjson_error_code_t pure_simdjson_element_get_bigint(const struct pure_simdjson_value_view_t *view,
                                                            uint8_t **out_ptr,
                                                            size_t *out_len);
/* ... */
pure_simdjson_error_code_t pure_simdjson_object_get_field(const struct pure_simdjson_value_view_t *object_view,
                                                          const uint8_t *key_ptr,
                                                          size_t key_len,
                                                          struct pure_simdjson_value_view_t *out_value);
```
`cbindgen` will emit the 9 new declarations in this identical style purely from the `/// # Safety`
doc comments + signatures written in `src/lib.rs` (Layer 4) — no `cbindgen.toml` change expected.

**`docs/ffi-contract.md` — Error code space table addition** (L31-57, current tail):
```
| `PURE_SIMDJSON_ERR_KERNEL_LOCKED` | 10 | Process-global implementation selection is permanently locked |
| `PURE_SIMDJSON_ERR_INVALID_JSON` | 32 | ... |
```
Insert `PURE_SIMDJSON_ERR_INVALID_PATH = 11` and `PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE = 12`
between the `10` row and the `32` row, consuming 2 of the documented "11–31 reserved" gap (L57).

**`docs/ffi-contract.md` — "Worked call sequences" section, needs a new subsection for the view-array
free, modeled on the existing "String copy and free" sequence** (L296-315):
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
New "Wildcard match array copy and free" subsection follows this exact call-then-free-then-doc-free
shape: `pure_simdjson_element_at_path_wildcard(&root, path_ptr, path_len, &views_ptr, &views_len)`
-> caller copies `views_ptr[0..views_len]` -> `pure_simdjson_value_views_free(views_ptr, views_len)`
-> `pure_simdjson_doc_free(doc)` -> `pure_simdjson_parser_free(parser)`.

---

## Shared Patterns

### `usableDoc()` handle-validity guard
**Source:** `element.go` L69-80
**Apply to:** Every new `Element`/`Array`/`Object` method (`AtPointer`, `AtPath`, `AtPathAll`,
`Array.At`, `Array.Len`/`LenErr`, `Object.Size`/`SizeErr`) — call first, before touching
`e.view`/native state, identical to every existing accessor.

### `wrapStatus(rc)` / `sentinelForStatus(code)` native-status-to-Go-error mapping
**Source:** `errors.go` L177-182, L222-259
**Apply to:** Every new Go method and standalone function. Extend `sentinelForStatus`'s `switch` with
two new `case ffi.ErrInvalidPath: return ErrInvalidPath` / `case ffi.ErrIndexOutOfRange: return
ErrIndexOutOfRange` arms — do not invent a parallel error-mapping path.

### `ffi_wrap`/`catch_unwind` FFI boundary (mandatory per `docs/ffi-contract.md`)
**Source:** `src/lib.rs` L193-209
**Apply to:** All 9 new `pure_simdjson_*` Rust exports, with zero exceptions — this is a normative,
not optional, project-wide rule (`docs/ffi-contract.md` "Panic and exception policy" section).

### `with_resolved_view` handle validation (generation + state-tag + descendant-membership check)
**Source:** `src/runtime/registry.rs` L731-766
**Apply to:** Every new registry function that takes a `*const pure_simdjson_value_view_t` —
`element_at_pointer`, `element_at_path`, `element_at_path_wildcard`, `array_at`, `array_len`,
`object_size`. This is the "handle-validation prologue" that must run before any native call, per
the phase-specific guidance.

### `PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)` try/catch wrapper
**Source:** every function in `src/native/simdjson_bridge.cpp` (e.g. L1514, L1578) uses `try { ... }
PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)`.
**Apply to:** All new C++ bridge functions — no exceptions cross into Rust, per
`docs/ffi-contract.md`'s panic/exception policy.

### `map_error(simdjson::error_code)` central translation table
**Source:** `src/native/simdjson_bridge.cpp` L152 onward
**Apply to:** Every new bridge function that calls into upstream `at_pointer`/`at_path`/
`at_path_with_wildcard`/`array::at`/`minify` — route the returned `simdjson::error_code` through
`map_error()` rather than hand-mapping status codes per call site; add the 2 new cases here once
(`INVALID_JSON_POINTER`, `INDEX_OUT_OF_BOUNDS`).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `element.go` — `Element.AtPathAll` `[]Element` construction loop | controller | batch | No existing Go function returns `[]Element` from one native call; nearest building blocks (`Object.GetField` single-result + `ArrayIter.Next` ordered-stream) are composed per RESEARCH.md's own designed snippet (`12-RESEARCH.md` L441-463) rather than cloned from one existing function |
| `src/native/simdjson_bridge.cpp` — `psimdjson_element_at_path_wildcard_indices`'s doc-owned scratch `std::vector<uint64_t>` + `wildcard_build_guard` | service | batch | Structurally new (bulk transport of N indices from one call), though every *piece* it's built from (`materialize_frames`/`materialize_in_progress`/`materialize_build_guard`, `psimdjson_doc` struct shape) already exists verbatim — see Layer 5 above; RESEARCH.md L356-398 has the full worked function |

## Metadata

**Analog search scope:** `element.go`, `errors.go`, `doc.go`, `kernel.go`, `library_loading.go`,
`parser.go`, `iterator.go`, `materializer_fastpath.go`, `element_scalar_test.go`,
`internal/ffi/bindings.go`, `internal/ffi/types.go`, `src/lib.rs`, `src/runtime/mod.rs`,
`src/runtime/registry.rs`, `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp`,
`tests/rust_shim_accessors.rs`, `tests/rust_shim_iterators.rs`, `tests/abi/check_header.py`,
`include/pure_simdjson.h`, `docs/ffi-contract.md`
**Files scanned:** 20
**Pattern extraction date:** 2026-07-30
