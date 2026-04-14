#![allow(non_camel_case_types)]

use core::ptr;

/// Stable packed ABI version for the Phase 1 contract.
pub const PURE_SIMDJSON_ABI_VERSION: u32 = 0x0001_0000;

/// Public error codes for the stable Phase 1 ABI.
#[repr(i32)]
pub enum pure_simdjson_error_code_t {
    PURE_SIMDJSON_OK = 0,
    PURE_SIMDJSON_ERR_INVALID_ARGUMENT = 1,
    PURE_SIMDJSON_ERR_INVALID_HANDLE = 2,
    PURE_SIMDJSON_ERR_PARSER_BUSY = 3,
    PURE_SIMDJSON_ERR_WRONG_TYPE = 4,
    PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND = 5,
    PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL = 6,
    PURE_SIMDJSON_ERR_INVALID_JSON = 32,
    PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE = 33,
    PURE_SIMDJSON_ERR_PRECISION_LOSS = 34,
    PURE_SIMDJSON_ERR_CPU_UNSUPPORTED = 64,
    PURE_SIMDJSON_ERR_ABI_MISMATCH = 65,
    PURE_SIMDJSON_ERR_PANIC = 96,
    PURE_SIMDJSON_ERR_CPP_EXCEPTION = 97,
    PURE_SIMDJSON_ERR_INTERNAL = 127,
}

/// Coarse value kind tags used by `pure_simdjson_value_view_t.kind_hint`.
#[repr(u32)]
pub enum pure_simdjson_value_kind_t {
    PURE_SIMDJSON_VALUE_KIND_INVALID = 0,
    PURE_SIMDJSON_VALUE_KIND_NULL = 1,
    PURE_SIMDJSON_VALUE_KIND_BOOL = 2,
    PURE_SIMDJSON_VALUE_KIND_INT64 = 3,
    PURE_SIMDJSON_VALUE_KIND_UINT64 = 4,
    PURE_SIMDJSON_VALUE_KIND_FLOAT64 = 5,
    PURE_SIMDJSON_VALUE_KIND_STRING = 6,
    PURE_SIMDJSON_VALUE_KIND_ARRAY = 7,
    PURE_SIMDJSON_VALUE_KIND_OBJECT = 8,
}

/// Opaque parser and document handles are packed `slot:u32 | generation:u32`.
pub type pure_simdjson_handle_t = u64;

/// Split view of a packed `pure_simdjson_handle_t`.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_handle_parts_t {
    pub slot: u32,
    pub generation: u32,
}

/// Lightweight document-tied node view used for roots, fields, and iterator results.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_value_view_t {
    pub doc: pure_simdjson_handle_t,
    pub state0: u64,
    pub state1: u64,
    pub kind_hint: u32,
    pub reserved: u32,
}

/// Stateful array iterator tied to a live document handle.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_array_iter_t {
    pub doc: pure_simdjson_handle_t,
    pub state0: u64,
    pub state1: u64,
    pub index: u64,
}

/// Stateful object iterator tied to a live document handle.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_object_iter_t {
    pub doc: pure_simdjson_handle_t,
    pub state0: u64,
    pub state1: u64,
    pub index: u64,
}

const CONTRACT_ONLY_IMPLEMENTATION_NAME: &[u8] = b"contract-only";

const PURE_SIMDJSON_OK: i32 = pure_simdjson_error_code_t::PURE_SIMDJSON_OK as i32;
const PURE_SIMDJSON_ERR_INVALID_ARGUMENT: i32 =
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_ARGUMENT as i32;
const PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL: i32 =
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL as i32;
const PURE_SIMDJSON_ERR_INTERNAL: i32 =
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL as i32;

unsafe fn write_out<T>(out: *mut T, value: T) -> i32 {
    if out.is_null() {
        return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
    }

    unsafe {
        ptr::write(out, value);
    }

    PURE_SIMDJSON_OK
}

unsafe fn copy_out_bytes(src: &[u8], dst: *mut u8, dst_cap: usize, out_written: *mut usize) -> i32 {
    if out_written.is_null() {
        return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
    }

    unsafe {
        ptr::write(out_written, src.len());
    }

    if src.len() > dst_cap {
        return PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL;
    }

    if !src.is_empty() {
        if dst.is_null() {
            return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
        }

        unsafe {
            ptr::copy_nonoverlapping(src.as_ptr(), dst, src.len());
        }
    }

    PURE_SIMDJSON_OK
}

/// Write the packed ABI version expected by Go-side compatibility checks.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_abi_version(out_version: *mut u32) -> i32 {
    unsafe { write_out(out_version, PURE_SIMDJSON_ABI_VERSION) }
}

/// Report the byte length of the active implementation name.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_implementation_name_len(out_len: *mut usize) -> i32 {
    unsafe { write_out(out_len, CONTRACT_ONLY_IMPLEMENTATION_NAME.len()) }
}

/// Copy the active implementation name into caller-owned storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_copy_implementation_name(
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> i32 {
    unsafe { copy_out_bytes(CONTRACT_ONLY_IMPLEMENTATION_NAME, dst, dst_cap, out_written) }
}

/// Allocate a parser handle.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_new(out_parser: *mut pure_simdjson_handle_t) -> i32 {
    let _ = out_parser;
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Release a parser handle after all associated documents have been freed.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_free(parser: pure_simdjson_handle_t) -> i32 {
    let _ = parser;
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Parse one JSON buffer into a new document handle.
///
/// Contract:
/// - Every call copies `input_ptr[..input_len]` into Rust-owned padded storage before simdjson
///   sees it, with enough trailing capacity for `SIMDJSON_PADDING`.
/// - A parser owns at most one live document at a time. If `parser` already owns a live document,
///   this function returns `PURE_SIMDJSON_ERR_PARSER_BUSY`.
/// - Re-parse never implicitly invalidates an existing document. The busy state remains until
///   `pure_simdjson_doc_free` succeeds for that document.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_parse(
    parser: pure_simdjson_handle_t,
    input_ptr: *const u8,
    input_len: usize,
    out_doc: *mut pure_simdjson_handle_t,
) -> i32 {
    let _ = (parser, input_ptr, input_len, out_doc);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Report the byte length of the parser's last diagnostic message.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_len(
    parser: pure_simdjson_handle_t,
    out_len: *mut usize,
) -> i32 {
    let _ = (parser, out_len);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Copy the parser's last diagnostic message into caller-owned storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_copy_last_error(
    parser: pure_simdjson_handle_t,
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> i32 {
    let _ = (parser, dst, dst_cap, out_written);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Report the byte offset associated with the parser's last failure.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_offset(
    parser: pure_simdjson_handle_t,
    out_offset: *mut u64,
) -> i32 {
    let _ = (parser, out_offset);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Release a live document handle.
///
/// Contract:
/// - `pure_simdjson_doc_free` is the only operation that clears a parser's busy state.
/// - Parser reuse never happens implicitly from `pure_simdjson_parser_parse`.
/// - Generation checks remain the mechanism that turns stale parser/doc/view use into
///   deterministic `PURE_SIMDJSON_ERR_INVALID_HANDLE` failures instead of undefined behavior.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_free(doc: pure_simdjson_handle_t) -> i32 {
    let _ = doc;
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Resolve the root value view for a live document handle.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_root(
    doc: pure_simdjson_handle_t,
    out_root: *mut pure_simdjson_value_view_t,
) -> i32 {
    let _ = (doc, out_root);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Report the value kind for a document-tied view.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_type(
    view: *const pure_simdjson_value_view_t,
    out_type: *mut u32,
) -> i32 {
    let _ = (view, out_type);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Decode the referenced value as `int64_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_int64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut i64,
) -> i32 {
    let _ = (view, out_value);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Decode the referenced value as `uint64_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_uint64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u64,
) -> i32 {
    let _ = (view, out_value);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Decode the referenced value as `double`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_float64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut f64,
) -> i32 {
    let _ = (view, out_value);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Copy the referenced string value into a newly allocated byte buffer.
///
/// The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
/// `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_string(
    view: *const pure_simdjson_value_view_t,
    out_ptr: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    let _ = (view, out_ptr, out_len);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Release memory previously returned by `pure_simdjson_element_get_string`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_bytes_free(ptr: *mut u8, len: usize) -> i32 {
    let _ = (ptr, len);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Decode the referenced value as a C `uint8_t` boolean.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_bool(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u8,
) -> i32 {
    let _ = (view, out_value);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Report whether the referenced value is JSON `null`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_is_null(
    view: *const pure_simdjson_value_view_t,
    out_is_null: *mut u8,
) -> i32 {
    let _ = (view, out_is_null);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Initialize array iterator state from an array-valued view.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_array_iter_new(
    array_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_array_iter_t,
) -> i32 {
    let _ = (array_view, out_iter);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Advance an array iterator and return the next value view plus a done flag.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_array_iter_next(
    iter: *mut pure_simdjson_array_iter_t,
    out_value: *mut pure_simdjson_value_view_t,
    out_done: *mut u8,
) -> i32 {
    let _ = (iter, out_value, out_done);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Initialize object iterator state from an object-valued view.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_iter_new(
    object_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_object_iter_t,
) -> i32 {
    let _ = (object_view, out_iter);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Advance an object iterator and return the next key/value pair plus a done flag.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_iter_next(
    iter: *mut pure_simdjson_object_iter_t,
    out_key: *mut pure_simdjson_value_view_t,
    out_value: *mut pure_simdjson_value_view_t,
    out_done: *mut u8,
) -> i32 {
    let _ = (iter, out_key, out_value, out_done);
    PURE_SIMDJSON_ERR_INTERNAL
}

/// Look up one object field by key and return its value view through `out_value`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key_ptr: *const u8,
    key_len: usize,
    out_value: *mut pure_simdjson_value_view_t,
) -> i32 {
    let _ = (object_view, key_ptr, key_len, out_value);
    PURE_SIMDJSON_ERR_INTERNAL
}
