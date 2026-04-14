#![allow(non_camel_case_types)]
#![deny(clippy::missing_safety_doc)]

use core::ptr;
use std::{
    any::Any,
    panic::{catch_unwind, AssertUnwindSafe},
};

/// Stable packed ABI version for the Phase 1 contract.
///
/// This constant is part of the public C header and stays numerically pinned alongside
/// `pure_simdjson_get_abi_version`.
pub const PURE_SIMDJSON_ABI_VERSION: u32 = 0x0001_0000;

/// Public error codes for the stable Phase 1 ABI.
#[repr(i32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
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
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
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

/// Generic packed handle transport for the public ABI.
///
/// The numeric value `0` is reserved as the invalid sentinel and is never produced by
/// successful constructors.
pub type pure_simdjson_handle_t = u64;

/// Opaque parser handle packed as `slot:u32 | generation:u32`.
///
/// Parsers are thread-compatible, not thread-safe: one live parser/document graph must be
/// confined to one thread at a time.
pub type pure_simdjson_parser_t = pure_simdjson_handle_t;

/// Opaque document handle packed as `slot:u32 | generation:u32`.
///
/// The numeric value `0` is reserved as the invalid sentinel. Documents inherit the
/// thread-affinity of their owning parser.
pub type pure_simdjson_doc_t = pure_simdjson_handle_t;

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
    pub doc: pure_simdjson_doc_t,
    pub state0: u64,
    pub state1: u64,
    pub kind_hint: u32,
    pub reserved: u32,
}

/// Stateful array iterator tied to a live document handle.
///
/// `state0`, `state1`, and `tag` are implementation-owned. `index` stays `u32` because the
/// Phase 1 contract only admits documents below the 4 GiB simdjson ceiling. `reserved` stays
/// pinned for future contract growth and callers must leave it untouched.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_array_iter_t {
    pub doc: pure_simdjson_doc_t,
    pub state0: u64,
    pub state1: u64,
    pub index: u32,
    pub tag: u16,
    pub reserved: u16,
}

/// Stateful object iterator tied to a live document handle.
///
/// `state0`, `state1`, and `tag` are implementation-owned. `index` stays `u32` because the
/// Phase 1 contract only admits documents below the 4 GiB simdjson ceiling. `reserved` stays
/// pinned for future contract growth and callers must leave it untouched.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
pub struct pure_simdjson_object_iter_t {
    pub doc: pure_simdjson_doc_t,
    pub state0: u64,
    pub state1: u64,
    pub index: u32,
    pub tag: u16,
    pub reserved: u16,
}

#[inline]
fn contract_only_implementation_name() -> &'static [u8] {
    b"contract-only"
}

#[inline]
const fn err_ok() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_OK
}

#[inline]
const fn err_invalid_argument() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_ARGUMENT
}

#[inline]
const fn err_buffer_too_small() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL
}

#[inline]
#[cfg_attr(debug_assertions, allow(dead_code))]
const fn err_internal() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL
}

#[inline]
const fn err_panic() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_PANIC
}

#[inline]
fn panic_payload_message(payload: &(dyn Any + Send)) -> String {
    if let Some(message) = payload.downcast_ref::<&str>() {
        (*message).to_owned()
    } else if let Some(message) = payload.downcast_ref::<String>() {
        message.clone()
    } else {
        format!("non-string panic payload ({:?})", payload.type_id())
    }
}

#[inline]
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

unsafe fn copy_out_bytes(
    src: &[u8],
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    if out_written.is_null() {
        return err_invalid_argument();
    }

    unsafe {
        ptr::write(out_written, src.len());
    }

    if src.len() > dst_cap {
        return err_buffer_too_small();
    }

    if !src.is_empty() {
        if dst.is_null() {
            return err_invalid_argument();
        }

        unsafe {
            ptr::copy_nonoverlapping(src.as_ptr(), dst, src.len());
        }
    }

    err_ok()
}

#[inline]
fn phase1_contract_stub(function_name: &'static str) -> pure_simdjson_error_code_t {
    #[cfg(debug_assertions)]
    {
        eprintln!("phase-1 stub reached: {}", function_name);
        std::process::abort();
    }

    #[cfg(not(debug_assertions))]
    {
        let _ = function_name;
        err_internal()
    }
}

/// Write the packed ABI version expected by Go-side compatibility checks.
///
/// # Safety
/// `out_version` must be a valid writable pointer to a `u32`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_abi_version(
    out_version: *mut u32,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_get_abi_version", || unsafe {
        write_out(out_version, PURE_SIMDJSON_ABI_VERSION)
    })
}

/// Report the byte length of the active implementation name.
///
/// # Safety
/// `out_len` must be a valid writable pointer to a `usize`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_implementation_name_len(
    out_len: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_get_implementation_name_len", || unsafe {
        write_out(out_len, contract_only_implementation_name().len())
    })
}

/// Copy the active implementation name into caller-owned storage.
///
/// `*out_written` receives the required byte count on both success and
/// `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`.
///
/// # Safety
/// `out_written` must be a valid writable pointer to a `usize`. When `dst_cap` is large enough
/// to copy the implementation name, `dst` must point to writable storage for at least `dst_cap`
/// bytes.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_copy_implementation_name(
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_copy_implementation_name", || unsafe {
        copy_out_bytes(
            contract_only_implementation_name(),
            dst,
            dst_cap,
            out_written,
        )
    })
}

/// Allocate a parser handle.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `out_parser` must be a valid writable pointer to a `pure_simdjson_parser_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_new(
    out_parser: *mut pure_simdjson_parser_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_new", || {
        let _ = out_parser;
        phase1_contract_stub("pure_simdjson_parser_new")
    })
}

/// Release a parser handle after all associated documents have been freed.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `parser` must be a parser handle previously returned by this library. The sentinel `0` and
/// forged values are invalid.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_free(
    parser: pure_simdjson_parser_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_free", || {
        let _ = parser;
        phase1_contract_stub("pure_simdjson_parser_free")
    })
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
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `parser` must be a live parser handle from this library. When `input_len` is non-zero,
/// `input_ptr` must be readable for `input_len` bytes. `out_doc` must be a valid writable pointer
/// to a `pure_simdjson_doc_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_parse(
    parser: pure_simdjson_parser_t,
    input_ptr: *const u8,
    input_len: usize,
    out_doc: *mut pure_simdjson_doc_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_parse", || {
        let _ = (parser, input_ptr, input_len, out_doc);
        phase1_contract_stub("pure_simdjson_parser_parse")
    })
}

/// Report the byte length of the parser's last diagnostic message.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `parser` must be a live parser handle from this library. `out_len` must be a valid writable
/// pointer to a `usize`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_len(
    parser: pure_simdjson_parser_t,
    out_len: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_get_last_error_len", || {
        let _ = (parser, out_len);
        phase1_contract_stub("pure_simdjson_parser_get_last_error_len")
    })
}

/// Copy the parser's last diagnostic message into caller-owned storage.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `parser` must be a live parser handle from this library. `out_written` must be a valid
/// writable pointer to a `usize`. When `dst_cap` is large enough to copy the active diagnostic,
/// `dst` must point to writable storage for at least `dst_cap` bytes.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_copy_last_error(
    parser: pure_simdjson_parser_t,
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_copy_last_error", || {
        let _ = (parser, dst, dst_cap, out_written);
        phase1_contract_stub("pure_simdjson_parser_copy_last_error")
    })
}

/// Report the byte offset associated with the parser's last failure.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `parser` must be a live parser handle from this library. `out_offset` must be a valid writable
/// pointer to a `u64`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_offset(
    parser: pure_simdjson_parser_t,
    out_offset: *mut u64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_get_last_error_offset", || {
        let _ = (parser, out_offset);
        phase1_contract_stub("pure_simdjson_parser_get_last_error_offset")
    })
}

/// Release a live document handle.
///
/// Contract:
/// - `pure_simdjson_doc_free` is the only operation that clears a parser's busy state.
/// - Parser reuse never happens implicitly from `pure_simdjson_parser_parse`.
/// - Generation checks remain the mechanism that turns stale parser/doc/view use into
///   deterministic `PURE_SIMDJSON_ERR_INVALID_HANDLE` failures instead of undefined behavior.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `doc` must be a document handle previously returned by this library. The sentinel `0` and
/// forged values are invalid.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_free(
    doc: pure_simdjson_doc_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_doc_free", || {
        let _ = doc;
        phase1_contract_stub("pure_simdjson_doc_free")
    })
}

/// Resolve the root value view for a live document handle.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `doc` must be a live document handle from this library. `out_root` must be a valid writable
/// pointer to a `pure_simdjson_value_view_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_root(
    doc: pure_simdjson_doc_t,
    out_root: *mut pure_simdjson_value_view_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_doc_root", || {
        let _ = (doc, out_root);
        phase1_contract_stub("pure_simdjson_doc_root")
    })
}

/// Report the value kind for a document-tied view.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_type` must point to writable `u32` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_type(
    view: *const pure_simdjson_value_view_t,
    out_type: *mut u32,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_type", || {
        let _ = (view, out_type);
        phase1_contract_stub("pure_simdjson_element_type")
    })
}

/// Decode the referenced value as `int64_t`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `i64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_int64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut i64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_int64", || {
        let _ = (view, out_value);
        phase1_contract_stub("pure_simdjson_element_get_int64")
    })
}

/// Decode the referenced value as `uint64_t`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `u64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_uint64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_uint64", || {
        let _ = (view, out_value);
        phase1_contract_stub("pure_simdjson_element_get_uint64")
    })
}

/// Decode the referenced value as `double`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `f64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_float64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut f64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_float64", || {
        let _ = (view, out_value);
        phase1_contract_stub("pure_simdjson_element_get_float64")
    })
}

/// Copy the referenced string value into a newly allocated byte buffer.
///
/// The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
/// `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document.
/// `out_ptr` and `out_len` must point to writable storage owned by the caller.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_string(
    view: *const pure_simdjson_value_view_t,
    out_ptr: *mut *mut u8,
    out_len: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_string", || {
        let _ = (view, out_ptr, out_len);
        phase1_contract_stub("pure_simdjson_element_get_string")
    })
}

/// Release memory previously returned by `pure_simdjson_element_get_string`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `ptr` and `len` must describe an allocation previously returned by
/// `pure_simdjson_element_get_string`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_bytes_free(
    ptr: *mut u8,
    len: usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_bytes_free", || {
        let _ = (ptr, len);
        phase1_contract_stub("pure_simdjson_bytes_free")
    })
}

/// Decode the referenced value as a C `uint8_t` boolean.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `u8` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_bool(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_bool", || {
        let _ = (view, out_value);
        phase1_contract_stub("pure_simdjson_element_get_bool")
    })
}

/// Report whether the referenced value is JSON `null`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_is_null` must point to writable `u8` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_is_null(
    view: *const pure_simdjson_value_view_t,
    out_is_null: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_is_null", || {
        let _ = (view, out_is_null);
        phase1_contract_stub("pure_simdjson_element_is_null")
    })
}

/// Initialize array iterator state from an array-valued view.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `array_view` must point to a readable array-valued `pure_simdjson_value_view_t` derived from a
/// live document. `out_iter` must point to writable iterator storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_array_iter_new(
    array_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_array_iter_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_array_iter_new", || {
        let _ = (array_view, out_iter);
        phase1_contract_stub("pure_simdjson_array_iter_new")
    })
}

/// Advance an array iterator and return the next value view plus a done flag.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `iter` must point to readable and writable iterator state created by this library. `out_value`
/// and `out_done` must point to writable storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_array_iter_next(
    iter: *mut pure_simdjson_array_iter_t,
    out_value: *mut pure_simdjson_value_view_t,
    out_done: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_array_iter_next", || {
        let _ = (iter, out_value, out_done);
        phase1_contract_stub("pure_simdjson_array_iter_next")
    })
}

/// Initialize object iterator state from an object-valued view.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `object_view` must point to a readable object-valued `pure_simdjson_value_view_t` derived from
/// a live document. `out_iter` must point to writable iterator storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_iter_new(
    object_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_object_iter_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_object_iter_new", || {
        let _ = (object_view, out_iter);
        phase1_contract_stub("pure_simdjson_object_iter_new")
    })
}

/// Advance an object iterator and return the next key/value pair plus a done flag.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `iter` must point to readable and writable iterator state created by this library. `out_key`,
/// `out_value`, and `out_done` must point to writable storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_iter_next(
    iter: *mut pure_simdjson_object_iter_t,
    out_key: *mut pure_simdjson_value_view_t,
    out_value: *mut pure_simdjson_value_view_t,
    out_done: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_object_iter_next", || {
        let _ = (iter, out_key, out_value, out_done);
        phase1_contract_stub("pure_simdjson_object_iter_next")
    })
}

/// Look up one object field by key and return its value view through `out_value`.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `object_view` must point to a readable object-valued `pure_simdjson_value_view_t` derived from
/// a live document. When `key_len` is non-zero, `key_ptr` must be readable for `key_len` bytes.
/// `out_value` must point to writable storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key_ptr: *const u8,
    key_len: usize,
    out_value: *mut pure_simdjson_value_view_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_object_get_field", || {
        let _ = (object_view, key_ptr, key_len, out_value);
        phase1_contract_stub("pure_simdjson_object_get_field")
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        any::TypeId,
        env,
        process::{Command, Output},
    };

    fn assert_subprocess_ran_exactly_one_test(output: &Output) {
        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(
            stdout.contains("running 1 test"),
            "subprocess filter should execute exactly one test: {stdout}",
        );
    }

    #[test]
    fn abi_version_getter_returns_the_pinned_constant() {
        let mut abi_version = 0_u32;

        let rc = unsafe { pure_simdjson_get_abi_version(&mut abi_version) };

        assert_eq!(rc, err_ok());
        assert_eq!(PURE_SIMDJSON_ABI_VERSION, 0x0001_0000);
        assert_eq!(abi_version, 0x0001_0000);
    }

    #[test]
    fn implementation_name_probe_reports_required_length() {
        let mut written = 0_usize;

        let rc =
            unsafe { pure_simdjson_copy_implementation_name(ptr::null_mut(), 0, &mut written) };

        assert_eq!(rc, err_buffer_too_small());
        assert_eq!(written, contract_only_implementation_name().len());
    }

    #[test]
    fn implementation_name_rejects_null_destination_when_capacity_is_sufficient() {
        let mut written = 0_usize;

        let rc = unsafe {
            pure_simdjson_copy_implementation_name(
                ptr::null_mut(),
                contract_only_implementation_name().len(),
                &mut written,
            )
        };

        assert_eq!(rc, err_invalid_argument());
        assert_eq!(written, contract_only_implementation_name().len());
    }

    #[test]
    fn ffi_exports_use_named_error_code_type() {
        let get_abi_version: unsafe extern "C" fn(*mut u32) -> pure_simdjson_error_code_t =
            pure_simdjson_get_abi_version;
        let get_implementation_name_len: unsafe extern "C" fn(
            *mut usize,
        )
            -> pure_simdjson_error_code_t = pure_simdjson_get_implementation_name_len;
        let copy_implementation_name: unsafe extern "C" fn(
            *mut u8,
            usize,
            *mut usize,
        ) -> pure_simdjson_error_code_t = pure_simdjson_copy_implementation_name;

        let _ = (
            get_abi_version,
            get_implementation_name_len,
            copy_implementation_name,
        );
    }

    #[test]
    fn raw_pointer_helpers_stay_unsafe_and_return_error_codes() {
        let write_u32: unsafe fn(*mut u32, u32) -> pure_simdjson_error_code_t = write_out::<u32>;
        let copy_bytes: unsafe fn(&[u8], *mut u8, usize, *mut usize) -> pure_simdjson_error_code_t =
            copy_out_bytes;

        let _ = (write_u32, copy_bytes);
    }

    #[test]
    fn ffi_wrap_converts_panics_to_err_panic() {
        let rc = ffi_wrap("ffi_wrap_converts_panics_to_err_panic", || {
            panic!("ffi panic sentinel")
        });

        assert_eq!(rc, pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_PANIC);
    }

    #[test]
    fn panic_payload_message_reports_type_id_for_non_string_payloads() {
        let payload: Box<dyn Any + Send> = Box::new(7_u32);
        let expected_type_id = format!("{:?}", TypeId::of::<u32>());
        let message = panic_payload_message(payload.as_ref());

        assert!(
            message.contains(&expected_type_id),
            "non-string panic payload diagnostics should include the concrete TypeId: {message}",
        );
    }

    #[test]
    fn ffi_wrap_reports_panic_payload_to_stderr() {
        if env::var_os("PURE_SIMDJSON_TRIGGER_FFI_PANIC").is_some() {
            let rc = ffi_wrap("ffi_wrap_reports_panic_payload_to_stderr", || {
                panic!("ffi panic sentinel")
            });

            assert_eq!(rc, pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_PANIC);
            return;
        }

        let output = Command::new(env::current_exe().expect("test binary path"))
            .env("PURE_SIMDJSON_TRIGGER_FFI_PANIC", "1")
            .arg("--exact")
            .arg("tests::ffi_wrap_reports_panic_payload_to_stderr")
            .arg("--nocapture")
            .output()
            .expect("spawn ffi panic subprocess");

        assert_subprocess_ran_exactly_one_test(&output);
        assert!(
            output.status.success(),
            "ffi panic conversion subprocess should stay alive"
        );

        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(
            stderr.contains("pure_simdjson panic in ffi_wrap_reports_panic_payload_to_stderr"),
            "ffi panic diagnostics should name the failing entry point: {stderr}",
        );
        assert!(
            stderr.contains("ffi panic sentinel"),
            "ffi panic diagnostics should preserve the panic payload: {stderr}",
        );
    }

    #[test]
    fn phase1_stub_hits_debug_tripwire() {
        if env::var_os("PURE_SIMDJSON_TRIGGER_STUB").is_some() {
            let _ = unsafe { pure_simdjson_parser_new(ptr::null_mut()) };
            return;
        }

        if !cfg!(debug_assertions) {
            return;
        }

        let output = Command::new(env::current_exe().expect("test binary path"))
            .env("PURE_SIMDJSON_TRIGGER_STUB", "1")
            .arg("--exact")
            .arg("tests::phase1_stub_hits_debug_tripwire")
            .arg("--nocapture")
            .output()
            .expect("spawn stub tripwire subprocess");

        assert_subprocess_ran_exactly_one_test(&output);
        assert!(
            !output.status.success(),
            "phase-1 stubs should fail fast in debug builds"
        );

        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(
            stderr.contains("phase-1 stub reached: pure_simdjson_parser_new"),
            "debug stub tripwire should emit the stub marker before aborting: {stderr}",
        );
    }

    #[cfg(not(debug_assertions))]
    #[test]
    fn phase1_stub_returns_err_internal_in_release_builds() {
        let mut parser = 0_u64;

        assert_eq!(
            unsafe { pure_simdjson_parser_new(&mut parser) },
            pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL
        );
    }
}
