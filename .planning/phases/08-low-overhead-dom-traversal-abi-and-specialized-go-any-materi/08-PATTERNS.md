# Phase 8: Low-overhead DOM traversal ABI and specialized Go any materializer - Pattern Map

**Mapped:** 2026-04-23
**Files analyzed:** 24
**Analogs found:** 24 / 24

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `src/lib.rs` | controller / FFI export | request-response, transform | `src/lib.rs` | exact |
| `src/runtime/mod.rs` | service / native bridge declaration | request-response | `src/runtime/mod.rs` | exact |
| `src/runtime/registry.rs` | service / registry | CRUD, request-response, transform | `src/runtime/registry.rs` | exact |
| `src/native/simdjson_bridge.h` | config / private C++ ABI | request-response, transform | `src/native/simdjson_bridge.h` | exact |
| `src/native/simdjson_bridge.cpp` | service / C++ bridge | transform, traversal | `src/native/simdjson_bridge.cpp` | exact |
| `internal/ffi/types.go` | model / FFI mirror | transform | `internal/ffi/types.go` | exact |
| `internal/ffi/bindings.go` | service / purego binding | request-response | `internal/ffi/bindings.go` | exact |
| `materializer_fastpath.go` (suggested new file) | utility / materializer | transform, traversal | `benchmark_comparators_test.go` | role-match |
| `parser.go` | public wrapper | request-response, lifecycle | `parser.go` | exact |
| `doc.go` | model / public wrapper | lifecycle, request-response | `doc.go` | exact |
| `element.go` | public wrapper | request-response, transform | `element.go` | exact |
| `iterator.go` | public wrapper / iterator | event-driven traversal | `iterator.go` | exact |
| `benchmark_comparators_test.go` | benchmark helper | batch, transform | `benchmark_comparators_test.go` | exact |
| `benchmark_diagnostics_test.go` | benchmark | batch diagnostics | `benchmark_diagnostics_test.go` | exact |
| `benchmark_native_alloc_test.go` | benchmark utility | batch telemetry | `benchmark_native_alloc_test.go` | exact |
| `materializer_fastpath_test.go` (new) | test | request-response, traversal, lifetime | `element_scalar_test.go`, `iterator_test.go` | role-match |
| `internal/ffi/types_test.go` (new if fixed frame ABI lands) | test | transform, ABI layout | `tests/abi/check_header.py`, `tests/smoke/ffi_export_surface.c` | partial |
| `element_scalar_test.go` | test | request-response | `element_scalar_test.go` | exact |
| `iterator_test.go` | test | event-driven traversal | `iterator_test.go` | exact |
| `tests/abi/check_header.py` | test utility | batch, ABI validation | `tests/abi/check_header.py` | exact |
| `cbindgen.toml` | config | codegen, ABI validation | `cbindgen.toml` | exact |
| `include/pure_simdjson.h` | generated config / public ABI | request-response | `include/pure_simdjson.h` | exact |
| `tests/smoke/ffi_export_surface.c` | smoke test | request-response, ABI validation | `tests/smoke/ffi_export_surface.c` | exact |
| `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` (new artifact) | testdata | batch benchmark evidence | `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt` | exact |

## Pattern Assignments

### `src/lib.rs` (controller / FFI export, request-response)

**Analog:** `src/lib.rs`

**Imports and ABI type pattern** (lines 4-11, 81-90):
```rust
mod runtime;

use core::ptr;
use std::{
    any::Any,
    panic::{catch_unwind, AssertUnwindSafe},
    slice,
};

#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_value_view_t {
    pub doc: pure_simdjson_doc_t,
    pub state0: u64,
    pub state1: u64,
    pub kind_hint: u32,
    pub reserved: u32,
}
```

**FFI panic/status wrapper** (lines 177-204):
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

unsafe fn write_out<T>(out: *mut T, value: T) -> pure_simdjson_error_code_t {
    if out.is_null() {
        return err_invalid_argument();
    }
    unsafe {
        ptr::write(out, value);
    }
    err_ok()
}
```

**Export shape to copy for internal functions** (lines 399-425):
```rust
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_parse(
    parser: pure_simdjson_parser_t,
    input_ptr: *const u8,
    input_len: usize,
    out_doc: *mut pure_simdjson_doc_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_parse", || unsafe {
        if out_doc.is_null() {
            return err_invalid_argument();
        }
        if input_len != 0 && input_ptr.is_null() {
            return err_invalid_argument();
        }

        let input = if input_len == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(input_ptr, input_len)
        };

        match runtime::registry::parser_parse(parser, input) {
            Ok(doc) => write_out(out_doc, doc),
            Err(rc) => rc,
        }
    })
}
```

**Apply:** New internal DOM traversal exports should keep numeric status returns plus pointer/integer out-params. If exported from Rust with `#[no_mangle]`, add explicit cbindgen/header guards so they do not become public header contract in Phase 8.

---

### `src/runtime/mod.rs` (service / native bridge declaration, request-response)

**Analog:** `src/runtime/mod.rs`

**Private C++ declaration pattern** (lines 29-64):
```rust
#[repr(C)]
pub(crate) struct psimdjson_parser {
    _private: [u8; 0],
}

#[repr(C)]
pub(crate) struct psimdjson_doc {
    _private: [u8; 0],
}

unsafe extern "C" {
    fn psimdjson_parser_new(out_parser: *mut *mut psimdjson_parser) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_free(parser: *mut psimdjson_parser) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_parse(
        parser: *mut psimdjson_parser,
        input_ptr: *const u8,
        input_len: usize,
        out_doc: *mut *mut psimdjson_doc,
    ) -> pure_simdjson_error_code_t;
}
```

**Bridge cleanup pattern** (lines 254-302):
```rust
pub(crate) fn native_parser_parse(
    parser_ptr: usize,
    input: &[u8],
    input_len: usize,
) -> Result<NativeParsedDoc, pure_simdjson_error_code_t> {
    let mut doc = ptr::null_mut();
    let rc = unsafe {
        psimdjson_parser_parse(
            parser_ptr as *mut psimdjson_parser,
            input.as_ptr(),
            input_len,
            &mut doc,
        )
    };
    if rc != err_ok() {
        return Err(rc);
    }
    if doc.is_null() {
        return Err(err_internal());
    }

    let mut root = ptr::null();
    let root_rc = unsafe { psimdjson_doc_root(doc, &mut root) };
    if root_rc != err_ok() {
        let free_rc = unsafe { psimdjson_doc_free(doc) };
        if free_rc != err_ok() {
            eprintln!(
                "pure_simdjson cleanup failure in native_parser_parse/doc_root: {:?}",
                free_rc
            );
        }
        return Err(root_rc);
    }
    if root.is_null() {
        let free_rc = unsafe { psimdjson_doc_free(doc) };
        if free_rc != err_ok() {
            eprintln!(
                "pure_simdjson cleanup failure in native_parser_parse/null_root: {:?}",
                free_rc
            );
        }
        return Err(err_internal());
    }

    Ok(NativeParsedDoc {
        doc_ptr: doc as usize,
        root_ptr: root as usize,
    })
}
```

**Apply:** Add private traversal builder declarations beside `psimdjson_*` declarations. Return `Result<..., pure_simdjson_error_code_t>` in Rust wrappers and free/rollback native resources on partial failure before returning errors.

---

### `src/runtime/registry.rs` (service / registry, CRUD + traversal validation)

**Analog:** `src/runtime/registry.rs`

**Doc-owned lifetime pattern** (lines 43-57):
```rust
#[derive(Clone, Debug)]
struct DocEntry {
    generation: u32,
    native_ptr: usize,
    root_ptr: usize,
    root_after_index: u64,
    owner_slot: u32,
    owner_generation: u32,
    #[allow(dead_code)]
    // Pinned: simdjson's parsed tape and borrowed string views remain tied to this owned buffer
    // for the lifetime of the document entry, even though Rust never reads the field directly.
    input_storage: Vec<u8>,
    descendant_indices: HashSet<u64>,
    iter_leases: HashMap<u32, IteratorLease>,
    next_iter_lease: u32,
}
```

**Copy-in parse and rollback pattern** (lines 432-483):
```rust
pub(crate) fn parser_parse(
    handle: pure_simdjson_parser_t,
    input: &[u8],
) -> Result<pure_simdjson_doc_t, pure_simdjson_error_code_t> {
    let mut registry = registry_guard();
    let (index, slot, generation) = unpack_handle(handle)?;

    let (native_ptr, mut owned_input) = match registry.parsers.get_mut(index) {
        Some(Slot::Occupied(entry)) if entry.generation == generation => {
            if !matches!(entry.state, ParserState::Idle) {
                return Err(err_parser_busy());
            }
            (entry.native_ptr, mem::take(&mut entry.reusable_input))
        }
        _ => return Err(err_invalid_handle()),
    };

    let padding = super::padding_bytes()?;
    let total_len = input
        .len()
        .checked_add(padding)
        .ok_or_else(err_invalid_argument)?;
    owned_input.resize(total_len, 0);
    owned_input[..input.len()].copy_from_slice(input);
    owned_input[input.len()..].fill(0);

    let parsed = match super::native_parser_parse(native_ptr, &owned_input[..], input.len()) {
        Ok(parsed) => parsed,
        Err(rc) => {
            restore_parser_input_buffer(&mut registry, index, generation, owned_input);
            return Err(rc);
        }
    };
    let root_after_index = match super::native_element_after_index(parsed.doc_ptr, ROOT_JSON_INDEX)
    {
        Ok(root_after_index) => root_after_index,
        Err(rc) => {
            let free_rc = super::native_doc_free(parsed.doc_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_parse/root_after_index: {:?}",
                    free_rc
                );
            }
            restore_parser_input_buffer(&mut registry, index, generation, owned_input);
            return Err(rc);
        }
    };
```

**View validation pattern** (lines 683-718):
```rust
fn with_resolved_view<T, F>(
    view: *const pure_simdjson_value_view_t,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(&mut DocEntry, u64, pure_simdjson_doc_t) -> Result<T, pure_simdjson_error_code_t>,
{
    if view.is_null() {
        return Err(err_invalid_argument());
    }

    let view = unsafe { ptr::read_unaligned(view) };
    if view.state0 == 0 || view.reserved != 0 {
        return Err(err_invalid_handle());
    }

    let (doc_index, _, doc_generation) = unpack_handle(view.doc)?;
    let mut registry = registry_guard();
    let entry = match registry.docs.get_mut(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => entry,
        _ => return Err(err_invalid_handle()),
    };
    let json_index = match view.state1 {
        ROOT_VIEW_TAG => {
            if entry.root_ptr != view.state0 as usize {
                return Err(err_invalid_handle());
            }
            ROOT_JSON_INDEX
        }
        DESC_VIEW_TAG => validate_descendant(&view, entry)?,
        _ => return Err(err_invalid_handle()),
    };
    action(entry, json_index, view.doc)
}
```

**Existing copy-out string pattern to avoid in fast path but preserve publicly** (lines 793-837):
```rust
pub(crate) fn element_get_string(
    view: *const pure_simdjson_value_view_t,
) -> Result<(*mut u8, usize), pure_simdjson_error_code_t> {
    let (ptr, len) = with_resolved_view(view, |entry, json_index, _| {
        let (borrowed_ptr, len) =
            super::native_element_get_string_view(entry.native_ptr, json_index)?;
        if len == 0 {
            return Ok((ptr::null_mut(), 0));
        }
        if borrowed_ptr == 0 {
            return Err(err_internal());
        }

        let bytes = unsafe { slice::from_raw_parts(borrowed_ptr as *const u8, len) };
        let mut owned = bytes.to_vec().into_boxed_slice().into_vec();
        let ptr = owned.as_mut_ptr();
        let len = owned.len();
        debug_assert_eq!(owned.len(), owned.capacity());
        mem::forget(owned);
        Ok((ptr, len))
    })?;
```

**Iterator lease pattern** (lines 916-944):
```rust
fn with_iter_doc<T, F>(
    doc: pure_simdjson_doc_t,
    state0: u64,
    state1: u64,
    lease_id: u32,
    tag: u16,
    reserved: u16,
    expected_tag: u16,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(&mut DocEntry) -> Result<T, pure_simdjson_error_code_t>,
{
    if reserved != 0 || tag != expected_tag || state0 > state1 {
        return Err(err_invalid_handle());
    }

    let (doc_index, _, doc_generation) = unpack_handle(doc)?;
    let mut registry = registry_guard();
    let entry = match registry.docs.get_mut(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => entry,
        _ => return Err(err_invalid_handle()),
    };
    entry.validate_iter_lease(lease_id, state0, state1, expected_tag)?;
    validate_iter_index(state0, entry.root_after_index)?;
    validate_iter_index(state1, entry.root_after_index)?;
    action(entry)
}
```

**Apply:** The frame/tape builder should validate the doc/view once using this same handle/generation/tag discipline. If borrowed frame storage is doc-owned, store it in `DocEntry` so Rust/native bytes cannot outlive the document.

---

### `src/native/simdjson_bridge.h` (private C++ ABI, request-response)

**Analog:** `src/native/simdjson_bridge.h`

**Header pattern** (lines 1-18, 86-125):
```c
#ifndef PSIMDJSON_BRIDGE_H
#define PSIMDJSON_BRIDGE_H

#include "../../include/pure_simdjson.h"
#include "simdjson.h"

#ifdef __cplusplus
extern "C" {
#define PSIMDJSON_NOEXCEPT noexcept
#else
#define PSIMDJSON_NOEXCEPT
#endif

typedef struct psimdjson_parser psimdjson_parser;
typedef struct psimdjson_doc psimdjson_doc;
typedef struct psimdjson_element psimdjson_element;

pure_simdjson_error_code_t psimdjson_element_get_string_view(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t **out_ptr,
    size_t *out_len
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_object_get_field_index(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t *key_ptr,
    size_t key_len,
    uint64_t *out_value_json_index
) PSIMDJSON_NOEXCEPT;
```

**Apply:** Add new private traversal structs/functions here, not to `include/pure_simdjson.h`. Keep `PSIMDJSON_NOEXCEPT`, pointer out-params, and opaque private C++ types.

---

### `src/native/simdjson_bridge.cpp` (C++ bridge, traversal transform)

**Analog:** `src/native/simdjson_bridge.cpp`

**Includes and private storage pattern** (lines 1-24):
```cpp
#include "simdjson_bridge.h"
#include "native_alloc_telemetry.h"

#include <cstdio>
#include <cstring>
#include <memory>
#include <stdexcept>
#include <string>
#include <type_traits>

struct psimdjson_doc {
  simdjson::dom::document document{};
  psimdjson_element root{};
};
```

**Error mapping and exception boundary** (lines 59-90, 183-206):
```cpp
pure_simdjson_error_code_t map_error(simdjson::error_code error) noexcept {
  switch (error) {
    case simdjson::SUCCESS:
      return PURE_SIMDJSON_OK;
    case simdjson::NO_SUCH_FIELD:
      return PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND;
    case simdjson::INCORRECT_TYPE:
      return PURE_SIMDJSON_ERR_WRONG_TYPE;
    case simdjson::NUMBER_OUT_OF_RANGE:
      return PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE;
    case simdjson::BIGINT_ERROR:
      return PURE_SIMDJSON_ERR_PRECISION_LOSS;
    case simdjson::TAPE_ERROR:
    case simdjson::DEPTH_ERROR:
    case simdjson::STRING_ERROR:
      return PURE_SIMDJSON_ERR_INVALID_JSON;
```

```cpp
#define PSIMDJSON_CATCH_CPP_EXCEPTIONS(function_name)                    \
  catch (const std::bad_alloc &error) {                                  \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (const std::exception &error) {                                   \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (...) {                                                          \
    return map_cpp_exception(function_name);                              \
  }
```

**Tape reconstruction pattern** (lines 212-249):
```cpp
simdjson::dom::element element_at(const psimdjson_doc *doc, uint64_t json_index) noexcept {
  static_assert(
      sizeof(simdjson::dom::element) == sizeof(simdjson::internal::tape_ref),
      "dom::element layout must stay tape_ref-sized for descendant reconstruction"
  );
  static_assert(
      std::is_trivially_copyable_v<simdjson::internal::tape_ref>,
      "tape_ref must remain trivially copyable for descendant reconstruction"
  );

  simdjson::dom::element element;
  auto *tape = reinterpret_cast<simdjson::internal::tape_ref *>(&element);
  *tape = simdjson::internal::tape_ref(&doc->document, size_t(json_index));
  return element;
}

simdjson::internal::tape_ref tape_ref_at(const psimdjson_doc *doc, uint64_t json_index) noexcept {
  return simdjson::internal::tape_ref(&doc->document, size_t(json_index));
}
```

**Borrowed string view pattern** (lines 510-530):
```cpp
pure_simdjson_error_code_t psimdjson_element_get_string_view(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t **out_ptr,
    size_t *out_len
) noexcept {
  try {
    if (doc == nullptr || out_ptr == nullptr || out_len == nullptr) {
      return invalid_argument();
    }

    std::string_view value;
    const auto error = element_at(doc, json_index).get_string().get(value);
    if (error != simdjson::SUCCESS) {
      return map_error(error);
    }

    *out_len = value.size();
    *out_ptr = value.empty() ? nullptr : reinterpret_cast<const uint8_t *>(value.data());
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}
```

**Apply:** Build the frame stream in one traversal using private tape knowledge, but expose a repo-owned frame layout to Rust/Go. Preserve `map_error`, null-pointer checks, and catch macros on every C++ entry point.

---

### `internal/ffi/types.go` (Go FFI mirror models, transform)

**Analog:** `internal/ffi/types.go`

**Type mirror pattern** (lines 11-84):
```go
type ErrorCode int32

const (
	OK                 ErrorCode = 0
	ErrInvalidArg      ErrorCode = 1
	ErrInvalidHandle   ErrorCode = 2
	ErrParserBusy      ErrorCode = 3
	ErrWrongType       ErrorCode = 4
	ErrElementNotFound ErrorCode = 5
	ErrBufferTooSmall  ErrorCode = 6

	ErrInvalidJSON      ErrorCode = 32
	ErrNumberOutOfRange ErrorCode = 33
	ErrPrecisionLoss    ErrorCode = 34
)

type ValueView struct {
	Doc      DocHandle
	State0   uint64
	State1   uint64
	KindHint uint32
	Reserved uint32
}

type NativeAllocStats struct {
	Epoch              uint64
	LiveBytes          uint64
	TotalAllocBytes    uint64
	AllocCount         uint64
	FreeCount          uint64
	UntrackedFreeCount uint64
}
```

**Apply:** Add internal frame structs here only if Go must mirror a fixed native layout. Keep field order and scalar widths identical to Rust/C++; add a layout test if any frame crosses FFI as a fixed struct.

---

### `internal/ffi/bindings.go` (purego binding service, request-response)

**Analog:** `internal/ffi/bindings.go`

**Binding registry pattern** (lines 15-46, 50-91):
```go
type Bindings struct {
	handle uintptr

	getABIVersion            func(*uint32) int32
	getImplementationNameLen func(*uintptr) int32
	copyImplementationName   func(*byte, uintptr, *uintptr) int32
	nativeAllocStatsReset    func() int32
	nativeAllocStatsSnapshot func(*NativeAllocStats) int32

	parserNew   func(*ParserHandle) int32
	parserFree  func(ParserHandle) int32
	parserParse func(ParserHandle, *byte, uintptr, *DocHandle) int32

	docFree        func(DocHandle) int32
	docRoot        func(DocHandle, *ValueView) int32
	elementGetString func(*ValueView, **byte, *uintptr) int32
}

func Bind(handle uintptr, lookup SymbolLookup) (*Bindings, error) {
	b := &Bindings{handle: handle}

	symbols := []struct {
		name   string
		target any
	}{
		{name: "pure_simdjson_get_abi_version", target: &b.getABIVersion},
		{name: "pure_simdjson_parser_parse", target: &b.parserParse},
		{name: "pure_simdjson_element_get_string", target: &b.elementGetString},
	}

	for _, symbol := range symbols {
		if err := registerFunc(handle, lookup, symbol.name, symbol.target); err != nil {
			return nil, err
		}
	}

	return b, nil
}
```

**KeepAlive and slice pointer pattern** (lines 170-180):
```go
func (b *Bindings) ParserParse(parser ParserHandle, data []byte) (DocHandle, int32) {
	var inputPtr *byte
	if len(data) > 0 {
		inputPtr = unsafe.SliceData(data)
	}

	var doc DocHandle
	rc := b.parserParse(parser, inputPtr, uintptr(len(data)), &doc)
	runtime.KeepAlive(data)
	runtime.KeepAlive(b)
	return doc, rc
}
```

**Current string copy/free pattern** (lines 262-285):
```go
func (b *Bindings) ElementGetString(view *ValueView) (string, int32) {
	var ptr *byte
	var length uintptr
	rc := b.elementGetString(view, &ptr, &length)
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

	if ptr == nil && length == 0 {
		return "", int32(OK)
	}

	return string(unsafe.Slice(ptr, length)), int32(OK)
}
```

**Apply:** Register internal fast-path symbols in the same `symbols` list. Borrowed frame/string pointers must be consumed before `runtime.KeepAlive(doc)` after the last read; do not route internal borrowed strings through `bytes_free`.

---

### `materializer_fastpath.go` (suggested new utility, transform/traversal)

**Analog:** `benchmark_comparators_test.go`

**Current recursive materializer semantics to preserve** (lines 418-482):
```go
func benchmarkMaterializePureElement(element Element) (any, error) {
	kind := ElementType(element.view.KindHint)
	if kind == TypeInvalid {
		resolvedKind, err := element.TypeErr()
		if err != nil {
			return nil, err
		}
		kind = resolvedKind
	}

	switch kind {
	case TypeNull:
		return nil, nil
	case TypeBool:
		return element.GetBool()
	case TypeInt64:
		return element.GetInt64()
	case TypeUint64:
		return element.GetUint64()
	case TypeFloat64:
		return element.GetFloat64()
	case TypeString:
		return element.GetString()
	case TypeArray:
		array, err := element.AsArray()
		if err != nil {
			return nil, err
		}

		values := make([]any, 0, 8)
		iter := array.Iter()
		for iter.Next() {
			value, err := benchmarkMaterializePureElement(iter.Value())
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		return values, nil
	case TypeObject:
		object, err := element.AsObject()
		if err != nil {
			return nil, err
		}

		values := make(map[string]any, 8)
		iter := object.Iter()
		for iter.Next() {
			value, err := benchmarkMaterializePureElement(iter.Value())
			if err != nil {
				return nil, err
			}
			values[iter.Key()] = value
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported pure-simdjson element type %v", kind)
	}
}
```

**Apply:** Replace the per-node accessor/iterator loop in benchmark paths with the internal frame walker. Preserve scalar kind distinctions, map assignment duplicate-key semantics, and fail-fast error returns. Replace fixed capacities `8` with frame child counts.

---

### `parser.go`, `doc.go`, `element.go`, `iterator.go` (public wrapper preservation)

**Analogs:** same files

**Parser lifecycle pattern** (`parser.go` lines 64-105):
```go
func (p *Parser) Parse(data []byte) (*Doc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.library == nil {
		return nil, ErrInvalidHandle
	}
	if p.closed {
		return nil, ErrClosed
	}
	if p.liveDoc != 0 {
		return nil, ErrParserBusy
	}

	handle := p.handle
	library := p.library

	docHandle, rc := library.bindings.ParserParse(handle, data)
	runtime.KeepAlive(data)
	runtime.KeepAlive(p)
	if err := wrapParserStatus(library.bindings, handle, rc); err != nil {
		return nil, err
	}

	root, rc := library.bindings.DocRoot(docHandle)
	runtime.KeepAlive(p)
	if err := wrapStatus(rc); err != nil {
		if freeErr := wrapStatus(library.bindings.DocFree(docHandle)); freeErr != nil {
			err = errors.Join(err, freeErr)
		}
		return nil, err
	}

	doc := &Doc{
		parser: p,
		handle: docHandle,
		root:   root,
	}
	attachDocFinalizer(doc)

	p.liveDoc = docHandle
	return doc, nil
}
```

**Doc close/finalizer pattern** (`doc.go` lines 28-57):
```go
func (d *Doc) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}

	handle := d.handle
	parser := d.parser
	library := parser.library

	clearDocFinalizer(d)
	rc := library.bindings.DocFree(handle)
	runtime.KeepAlive(d)
	runtime.KeepAlive(d.parser)
	if err := wrapStatus(rc); err != nil {
		attachDocFinalizer(d)
		d.mu.Unlock()
		return err
	}

	d.closed = true
	d.handle = 0
	d.mu.Unlock()

	parser.clearLiveDoc(handle)
	return nil
}
```

**Element validation + numeric precision pattern** (`element.go` lines 67-78, 168-207):
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

func (e Element) GetFloat64() (float64, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return 0, err
	}

	switch ffi.ValueKind(e.view.KindHint) {
	case ffi.ValueKindInt64:
		value, rc := doc.parser.library.bindings.ElementGetInt64(&e.view)
		runtime.KeepAlive(doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if !exactFloat64Int64(value) {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
	case ffi.ValueKindUint64:
		value, rc := doc.parser.library.bindings.ElementGetUint64(&e.view)
		runtime.KeepAlive(doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if !exactFloat64Uint64(value) {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
	}

	value, rc := doc.parser.library.bindings.ElementGetFloat64(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}
```

**Iterator hot path to bypass internally but preserve publicly** (`iterator.go` lines 79-123):
```go
func (it *ObjectIter) Next() bool {
	if it == nil || it.done || it.err != nil {
		return false
	}
	bindings, err := usableIteratorBindings(it.doc)
	if err != nil {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = err
		return false
	}

	keyView, valueView, done, rc := bindings.ObjectIterNext(&it.iter)
	runtime.KeepAlive(it.doc)
	if err := normalizeIteratorError(it.doc, rc); err != nil {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = err
		return false
	}
	if done {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		return false
	}

	key, rc := bindings.ElementGetString(&keyView)
	runtime.KeepAlive(it.doc)
	if err := normalizeIteratorError(it.doc, rc); err != nil {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = err
		return false
	}

	it.currentKey = key
	it.currentValue = Element{doc: it.doc, view: valueView}
	return true
}
```

**Apply:** Do not add public `Interface` methods in Phase 8. Any fast materializer entry should be unexported and should reuse `usableDoc`, `wrapStatus`, `normalizeIteratorError`, `runtime.KeepAlive`, and existing close/finalizer discipline.

---

### `benchmark_comparators_test.go` and `benchmark_diagnostics_test.go` (benchmark paths)

**Analogs:** same files

**Comparator parse/materialize pattern** (`benchmark_comparators_test.go` lines 356-380):
```go
func benchmarkMaterializePureSimdjson(_ string, data []byte) (result any, err error) {
	parser, err := NewParser()
	if err != nil {
		return nil, err
	}
	defer func() {
		err = benchmarkCloseMaterializeResources(err, nil, parser)
	}()

	return benchmarkMaterializePureSimdjsonWithParser(parser, data)
}

func benchmarkMaterializePureSimdjsonWithParser(parser *Parser, data []byte) (result any, err error) {
	var doc *Doc
	defer func() {
		err = benchmarkCloseMaterializeResources(err, doc, nil)
	}()

	doc, err = parser.Parse(data)
	if err != nil {
		return nil, err
	}

	result, err = benchmarkMaterializePureElement(doc.Root())
	return result, err
}
```

**Diagnostics split pattern** (`benchmark_diagnostics_test.go` lines 22-47, 98-128):
```go
func runTier1DiagnosticsBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	b.Run(benchmarkComparatorPureSimdjson+"-full", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsFullPureSimdjson(b, fixtureName, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-parse-only", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsParseOnly(b, fixtureName, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-materialize-only", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsMaterializeOnly(b, fixtureName, data)
	})
}

func benchmarkRunTier1DiagnosticsMaterializeOnly(b *testing.B, fixtureName string, data []byte) {
	parser := benchmarkWarmPureParser(b, fixtureName, data)
	defer func() {
		if err := parser.Close(); err != nil {
			b.Fatalf("parser.Close(%s): %v", fixtureName, err)
		}
	}()

	doc, err := parser.Parse(data)
	if err != nil {
		b.Fatalf("%s materialize-only Parse(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
	}
	defer func() {
		if err := doc.Close(); err != nil {
			b.Fatalf("doc.Close(%s): %v", fixtureName, err)
		}
	}()

	root := doc.Root()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	benchmarkRunWithNativeAllocMetrics(b, false, func() {
		for i := 0; i < b.N; i++ {
			value, err := benchmarkMaterializePureElement(root)
			if err != nil {
				b.Fatalf("%s materialize-only(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}
			benchmarkTier1Result = value
		}
	})
}
```

**Native allocation metric pattern** (`benchmark_native_alloc_test.go` lines 11-45):
```go
func benchmarkRunWithNativeAllocMetrics(b *testing.B, requireNativeAllocs bool, run func()) {
	b.Helper()

	library, err := activeLibrary()
	if err != nil {
		b.Fatalf("activeLibrary(): %v", err)
	}
	if err := wrapStatus(library.bindings.NativeAllocStatsReset()); err != nil {
		b.Fatalf("NativeAllocStatsReset(): %v", err)
	}

	b.ResetTimer()
	run()
	b.StopTimer()

	stats, rc := library.bindings.NativeAllocStatsSnapshot()
	if err := wrapStatus(rc); err != nil {
		b.Fatalf("NativeAllocStatsSnapshot(): %v", err)
	}
	if requireNativeAllocs && b.N > 0 && stats.AllocCount == 0 {
		b.Fatalf("NativeAllocStatsSnapshot(): alloc_count = 0, want native allocation telemetry for this path")
	}

	if b.N > 0 {
		perOp := float64(b.N)
		b.ReportMetric(float64(stats.TotalAllocBytes)/perOp, benchmarkMetricNativeBytesPerOp)
		b.ReportMetric(float64(stats.AllocCount)/perOp, benchmarkMetricNativeAllocsPerOp)
	}
	b.ReportMetric(float64(stats.LiveBytes), benchmarkMetricNativeLiveBytes)
}
```

**Apply:** Update only the pure-simdjson Tier 1 materialization/full paths to use the fast materializer. Keep parse-only and comparator names stable so Phase 8 can compare against Phase 7 raw diagnostics.

---

### `materializer_fastpath_test.go`, `element_scalar_test.go`, `iterator_test.go` (correctness tests)

**Analogs:** `element_scalar_test.go`, `iterator_test.go`, `benchmark_oracle_test.go`

**Test helper pattern** (`element_scalar_test.go` lines 17-38):
```go
func mustParseDoc(t *testing.T, json string) (*Parser, *Doc) {
	t.Helper()

	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(json))
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", json, err)
	}
	t.Cleanup(func() {
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() cleanup error = %v", err)
		}
	})

	return parser, doc
}
```

**Scalar/numeric error pattern** (`element_scalar_test.go` lines 412-459):
```go
func checkGetFloat64Int64Contract(parser *Parser, value int64) error {
	json := strconv.FormatInt(value, 10)
	got, err := parseRootFloat64(parser, json)
	if exactFloat64Int64(value) {
		if err != nil {
			return fmt.Errorf("GetFloat64(%s) error = %v, want nil", json, err)
		}
		if got != float64(value) {
			return fmt.Errorf("GetFloat64(%s) = %.0f, want %.0f", json, got, float64(value))
		}
		return nil
	}
	if !errors.Is(err, ErrPrecisionLoss) {
		return fmt.Errorf("GetFloat64(%s) error = %v, want ErrPrecisionLoss", json, err)
	}
	return nil
}

func parseRootFloat64(parser *Parser, json string) (float64, error) {
	doc, err := parser.Parse([]byte(json))
	if err != nil {
		return 0, fmt.Errorf("Parse(%q): %w", json, err)
	}

	value, getErr := doc.Root().GetFloat64()
	if closeErr := doc.Close(); closeErr != nil {
		return 0, fmt.Errorf("doc.Close(%q): %w", json, closeErr)
	}
	return value, getErr
}
```

**Duplicate-key public accessor preservation** (`iterator_test.go` lines 311-330):
```go
func TestObjectGetFieldDuplicateKeySemantics(t *testing.T) {
	_, doc := mustParseDoc(t, `{"dup":1,"dup":2}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	field, err := object.GetField("dup")
	if err != nil {
		t.Fatalf("GetField(\"dup\") error = %v", err)
	}
	value, err := field.GetInt64()
	if err != nil {
		t.Fatalf("GetInt64() error = %v", err)
	}
	if value != 1 {
		t.Fatalf("GetField(\"dup\").GetInt64() = %d, want first duplicate field", value)
	}
}
```

**Lifetime-after-close pattern** (`element_scalar_test.go` lines 490-512; `iterator_test.go` lines 582-628):
```go
func TestGetString(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(`"hello"`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	value, err := doc.Root().GetString()
	if err != nil {
		t.Fatalf("GetString() error = %v", err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if value != "hello" {
		t.Fatalf("GetString() copied value = %q, want %q", value, "hello")
	}
}
```

**Oracle loop pattern** (`benchmark_oracle_test.go` lines 13-75):
```go
func TestJSONTestSuiteOracle(t *testing.T) {
	manifest := loadOracleManifest(t)
	caseFiles := loadOracleCaseFiles(t)

	accepted := 0
	rejected := 0
	for _, oracleCase := range manifest {
		data, err := os.ReadFile(caseFiles[oracleCase.relativePath])
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", oracleCase.relativePath, err)
		}

		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser() for %q: %v", oracleCase.relativePath, err)
		}

		doc, parseErr := parser.Parse(data)
		closeOracleResources(t, oracleCase.relativePath, doc, parser)

		switch oracleCase.expect {
		case oracleExpectAccept:
			if parseErr != nil {
				t.Fatalf("Parse(%q) error = %v, want success (%s)", oracleCase.relativePath, parseErr, oracleCase.note)
			}
			accepted++
		case oracleExpectReject:
			if parseErr == nil {
				t.Fatalf("Parse(%q) unexpectedly succeeded, want reject (%s)", oracleCase.relativePath, oracleCase.note)
			}
			rejected++
		}
	}
}
```

**Apply:** Fast materializer tests should compare fast output to the existing accessor materializer or `encoding/json` where numeric semantics align. Add explicit cases for duplicate-key last-wins materialization while preserving `Object.GetField` first-match behavior.

---

### `internal/ffi/types_test.go` (new layout test if fixed frame ABI lands)

**Analog:** `tests/abi/check_header.py`, `tests/smoke/ffi_export_surface.c`

**Existing ABI shape assertions are in Python/C, not Go** (`tests/abi/check_header.py` lines 140-154):
```python
def rule_error_code_outparams(
    prototypes: dict[str, tuple[str, list[str]]], _: str
) -> None:
    for name, (return_type, params) in prototypes.items():
        if return_type not in (
            "pure_simdjson_error_code_t",
            "enum pure_simdjson_error_code_t",
        ):
            fail(
                f"{name}: expected pure_simdjson_error_code_t return, found {return_type}"
            )
        for param in params:
            if any(struct_type in param for struct_type in STRUCT_TYPES) and "*" not in param:
                fail(f"{name}: struct transport must use pointer out-params, found by-value parameter {param}")
```

**Existing C smoke fixed typedef usage** (`tests/smoke/ffi_export_surface.c` lines 81-126):
```c
typedef pure_simdjson_error_code_t (*fn_parser_parse)(pure_simdjson_parser_t,
                                                      const uint8_t *,
                                                      size_t,
                                                      pure_simdjson_doc_t *);
typedef pure_simdjson_error_code_t (*fn_element_get_string)(const pure_simdjson_value_view_t *,
                                                            uint8_t **,
                                                            size_t *);
typedef pure_simdjson_error_code_t (*fn_object_iter_next)(pure_simdjson_object_iter_t *,
                                                          pure_simdjson_value_view_t *,
                                                          pure_simdjson_value_view_t *,
                                                          uint8_t *);
```

**Apply:** If Go mirrors a native frame struct, add a Go layout test with `unsafe.Sizeof`/`unsafe.Offsetof` constants or native exported layout constants. There is no existing Go layout test; keep it package-local under `internal/ffi`.

---

### `tests/abi/check_header.py`, `cbindgen.toml`, `include/pure_simdjson.h`, `tests/smoke/ffi_export_surface.c` (ABI public-header guards)

**Analogs:** same files

**cbindgen public include/exclude pattern** (`cbindgen.toml` lines 9-53):
```toml
[export]
include = [
  "pure_simdjson_error_code_t",
  "pure_simdjson_value_kind_t",
  "pure_simdjson_handle_t",
  "pure_simdjson_parser_t",
  "pure_simdjson_doc_t",
  "pure_simdjson_handle_parts_t",
  "pure_simdjson_value_view_t",
  "pure_simdjson_array_iter_t",
  "pure_simdjson_object_iter_t",
  "pure_simdjson_native_alloc_stats_t",
]
exclude = [
  "psimdjson_parser",
  "psimdjson_doc",
  "psimdjson_element",
  "psimdjson_get_implementation_name_len",
  "psimdjson_copy_implementation_name",
  "psimdjson_parser_new",
  "psimdjson_parser_free",
  "psimdjson_parser_parse",
  "psimdjson_object_get_field_index",
  "psimdjson_test_force_cpp_exception",
]
```

**Required-symbol and unexpected-public-symbol guard** (`tests/abi/check_header.py` lines 46-73, 164-175):
```python
REQUIRED_SYMBOLS = (
    "pure_simdjson_get_abi_version",
    "pure_simdjson_get_implementation_name_len",
    "pure_simdjson_copy_implementation_name",
    "pure_simdjson_native_alloc_stats_reset",
    "pure_simdjson_native_alloc_stats_snapshot",
    "pure_simdjson_parser_new",
    "pure_simdjson_parser_free",
    "pure_simdjson_parser_parse",
    "pure_simdjson_parser_get_last_error_len",
    "pure_simdjson_parser_copy_last_error",
    "pure_simdjson_parser_get_last_error_offset",
    "pure_simdjson_doc_free",
    "pure_simdjson_doc_root",
    "pure_simdjson_element_type",
    "pure_simdjson_element_get_int64",
    "pure_simdjson_element_get_uint64",
    "pure_simdjson_element_get_float64",
    "pure_simdjson_element_get_string",
    "pure_simdjson_bytes_free",
    "pure_simdjson_element_get_bool",
    "pure_simdjson_element_is_null",
    "pure_simdjson_array_iter_new",
    "pure_simdjson_array_iter_next",
    "pure_simdjson_object_iter_new",
    "pure_simdjson_object_iter_next",
    "pure_simdjson_object_get_field",
)

def rule_required_symbols(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    missing = [symbol for symbol in REQUIRED_SYMBOLS if symbol not in prototypes]
    if missing:
        fail("missing required symbols: " + ", ".join(missing))
    unexpected = sorted(
        name
        for name in prototypes
        if name.startswith("pure_simdjson_") and name not in REQUIRED_SYMBOLS
    )
    if unexpected:
        fail("unexpected exported symbols: " + ", ".join(unexpected))
```

**Public header ownership rule to preserve** (`include/pure_simdjson.h` lines 354-376):
```c
/**
 * Copy the referenced string value into a newly allocated byte buffer.
 *
 * The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
 * `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_string(const struct pure_simdjson_value_view_t *view,
                                                            uint8_t **out_ptr,
                                                            size_t *out_len);

pure_simdjson_error_code_t pure_simdjson_bytes_free(uint8_t *ptr, size_t len);
```

**Smoke export surface pattern** (`tests/smoke/ffi_export_surface.c` lines 52-79, 245-292):
```c
static const char *EXPORT_NAMES[EXPORT_COUNT] = {
    "pure_simdjson_get_abi_version",
    "pure_simdjson_get_implementation_name_len",
    "pure_simdjson_copy_implementation_name",
    "pure_simdjson_native_alloc_stats_reset",
    "pure_simdjson_native_alloc_stats_snapshot",
    "pure_simdjson_parser_new",
    "pure_simdjson_parser_free",
    "pure_simdjson_parser_parse",
    "pure_simdjson_parser_get_last_error_len",
    "pure_simdjson_parser_copy_last_error",
    "pure_simdjson_parser_get_last_error_offset",
    "pure_simdjson_doc_free",
    "pure_simdjson_doc_root",
    "pure_simdjson_element_type",
    "pure_simdjson_element_get_int64",
    "pure_simdjson_element_get_uint64",
    "pure_simdjson_element_get_float64",
    "pure_simdjson_element_get_string",
    "pure_simdjson_bytes_free",
    "pure_simdjson_element_get_bool",
    "pure_simdjson_element_is_null",
    "pure_simdjson_array_iter_new",
    "pure_simdjson_array_iter_next",
    "pure_simdjson_object_iter_new",
    "pure_simdjson_object_iter_next",
    "pure_simdjson_object_get_field",
};

#define RESOLVE(field, index, type)                                                            \
  do {                                                                                         \
    void *symbol_ptr = lookup_symbol(exports, EXPORT_NAMES[index]);                            \
    if (symbol_ptr == NULL) {                                                                  \
      return failf("resolve", "failed to resolve %s", EXPORT_NAMES[index]);                    \
    }                                                                                          \
    exports->field = (type)symbol_ptr;                                                         \
  } while (0)
```

**Apply:** Internal fast-path symbols must be excluded from `include/pure_simdjson.h`. Extend `check_header.py` with an explicit forbidden internal-symbol check if the new Rust export names do not start with `pure_simdjson_`.

---

### `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` (benchmark artifact)

**Analog:** `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`

**Pattern:** Capture raw output from:
```bash
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5 > testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt
```

**Apply:** Keep the Phase 8 artifact separate from v0.1.1 baseline. Use `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` for closeout evidence.

## Shared Patterns

### Status And Error Handling

**Source:** `src/lib.rs`, `errors.go`, `src/native/simdjson_bridge.cpp`

**Apply to:** Rust exports, Rust registry calls, C++ bridge functions, Go wrappers, Go materializer

```go
func wrapStatus(code int32) error {
	if code == int32(ffi.OK) {
		return nil
	}
	return newError(code, nativeDetails{}, sentinelForStatus(code))
}

func sentinelForStatus(code int32) error {
	switch ffi.ErrorCode(code) {
	case ffi.ErrInvalidHandle:
		return ErrInvalidHandle
	case ffi.ErrParserBusy:
		return ErrParserBusy
	case ffi.ErrWrongType:
		return ErrWrongType
	case ffi.ErrElementNotFound:
		return ErrElementNotFound
	case ffi.ErrInvalidJSON:
		return ErrInvalidJSON
	case ffi.ErrNumberOutOfRange:
		return ErrNumberOutOfRange
	case ffi.ErrPrecisionLoss:
		return ErrPrecisionLoss
	default:
		return ErrInternal
	}
}
```

### Lifetime And KeepAlive

**Source:** `parser.go`, `doc.go`, `internal/ffi/bindings.go`, `src/runtime/registry.rs`

**Apply to:** Go binding wrappers, fast materializer, borrowed frame/string reads

```go
docHandle, rc := library.bindings.ParserParse(handle, data)
runtime.KeepAlive(data)
runtime.KeepAlive(p)

rc := library.bindings.DocFree(handle)
runtime.KeepAlive(d)
runtime.KeepAlive(d.parser)
```

Borrowed frame/key/string memory must be read and copied into Go-owned values before returning. Keep the owning `Doc` alive through the final borrowed read.

### Public Wrapper Stability

**Source:** `element.go`, `iterator.go`

**Apply to:** Any `parser.go`, `doc.go`, `element.go`, `iterator.go` edits

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

Do not change public accessor, iterator, or `Object.GetField` semantics while adding the internal materializer.

### Benchmark Evidence

**Source:** `benchmark_diagnostics_test.go`, `benchmark_native_alloc_test.go`

**Apply to:** Phase 8 performance validation

```go
b.ReportAllocs()
b.SetBytes(int64(len(data)))
benchmarkRunWithNativeAllocMetrics(b, false, func() {
	for i := 0; i < b.N; i++ {
		value, err := benchmarkMaterializePureElement(root)
		if err != nil {
			b.Fatalf("%s materialize-only(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
		}
		benchmarkTier1Result = value
	}
})
```

Keep diagnostic names and benchmark fixtures stable. The closeout proof is materialize-only and full Tier 1 improvement over the Phase 7 baseline, not public benchmark repositioning.

### Artifact Hygiene

**Source:** user/project instruction

**Apply to:** `08-PATTERNS.md`, future plans, commits, PR text, closeout docs

Do not include private/internal corporate repository hostnames or private repository details in generated artifacts.

## No Analog Found

All files have an exact, role-match, or partial analog. The only partial gap is Go-side fixed-layout struct testing under `internal/ffi`: no current Go `unsafe.Sizeof`/`unsafe.Offsetof` test exists, so use the ABI guard style from `tests/abi/check_header.py` and `tests/smoke/ffi_export_surface.c`.

## Metadata

**Analog search scope:** `src/`, `src/runtime/`, `src/native/`, `internal/ffi/`, root Go wrappers, root benchmark/test files, `tests/abi/`, `tests/smoke/`, `include/`, `cbindgen.toml`, `testdata/benchmark-results/`

**Files scanned:** 1,145 repo files listed; 38 source/test files in primary `src`, `internal`, and `tests` search scope; 24 phase-relevant files classified.

**Project context:** No repository-local `CLAUDE.md` content was present. Repo-local `pure-simdjson-release` skill is release-only and is not applicable to this Phase 8 planning artifact.

**Pattern extraction date:** 2026-04-23

## PATTERN MAPPING COMPLETE

**Phase:** 08 - Low-overhead DOM traversal ABI and specialized Go any materializer
**Files classified:** 24
**Analogs found:** 24 / 24

Planner can now reference the analog patterns above in Phase 8 PLAN.md files.
