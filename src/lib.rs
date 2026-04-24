#![allow(non_camel_case_types)]
#![deny(clippy::missing_safety_doc)]

mod runtime;

use core::ptr;
use std::{
    any::Any,
    panic::{catch_unwind, AssertUnwindSafe},
    slice,
};

/// Stable packed ABI version for the ABI v0.1 contract.
///
/// This constant is part of the public C header and stays numerically pinned alongside
/// `pure_simdjson_get_abi_version`.
pub const PURE_SIMDJSON_ABI_VERSION: u32 = 0x0001_0001;

/// Public error codes for the stable ABI v0.1 surface.
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
    /// Optional diagnostic/export surface is absent from the loaded artifact.
    PURE_SIMDJSON_ERR_NOT_IMPLEMENTED = 7,
    /// JSON nesting exceeds the parser/materializer depth contract.
    PURE_SIMDJSON_ERR_DEPTH_LIMIT = 8,
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
/// `state0`, `state1`, `index`, and `tag` are implementation-owned. `index` stays `u32`
/// because the ABI v0.1 layout only admits documents below the 4 GiB simdjson ceiling.
/// `reserved` stays pinned for future contract growth and callers must leave it untouched.
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
/// `state0`, `state1`, `index`, and `tag` are implementation-owned. `index` stays `u32`
/// because the ABI v0.1 layout only admits documents below the 4 GiB simdjson ceiling.
/// `reserved` stays pinned for future contract growth and callers must leave it untouched.
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

/// Diagnostic native allocator counters for the current telemetry epoch.
///
/// This surface reports allocations routed through the native shim/simdjson cdylib path only.
/// It does not claim process-wide totals or Go heap activity.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct pure_simdjson_native_alloc_stats_t {
    pub epoch: u64,
    pub live_bytes: u64,
    pub total_alloc_bytes: u64,
    pub alloc_count: u64,
    pub free_count: u64,
    pub untracked_free_count: u64,
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
#[cfg_attr(not(test), allow(dead_code))]
const fn err_buffer_too_small() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL
}

#[inline]
const fn err_cpu_unsupported() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_CPU_UNSUPPORTED
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

#[cfg_attr(not(test), allow(dead_code))]
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
fn reject_fallback_implementation() -> Result<(), pure_simdjson_error_code_t> {
    let implementation_name = runtime::selected_implementation_name_for_parser_new()?;
    if implementation_name.as_slice() == b"fallback" && !runtime::fallback_allowed_for_tests() {
        return Err(err_cpu_unsupported());
    }

    Ok(())
}

#[doc(hidden)]
pub fn pure_simdjson_test_force_cpp_exception_for_tests() -> pure_simdjson_error_code_t {
    runtime::test_force_cpp_exception_for_tests()
}

#[doc(hidden)]
pub fn pure_simdjson_test_set_forced_implementation_for_tests(value: Option<&[u8]>) {
    runtime::test_set_forced_implementation_override(value);
}

#[doc(hidden)]
pub fn pure_simdjson_test_set_allow_fallback_for_tests(value: Option<bool>) {
    runtime::test_set_fallback_allowed_override(value);
}

#[allow(private_interfaces)]
#[no_mangle]
/// # Safety
///
/// `view` must be a valid value view produced by this library. `out_frames`
/// and `out_frame_count` must be valid writable pointers. On success, the
/// returned frame span is borrowed from the owning document and is invalidated
/// by the next materialize-build call on that same document.
pub unsafe extern "C" fn psdj_internal_materialize_build(
    view: *const pure_simdjson_value_view_t,
    out_frames: *mut *const runtime::psdj_internal_frame_t,
    out_frame_count: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("psdj_internal_materialize_build", || unsafe {
        if out_frames.is_null() || out_frame_count.is_null() {
            return err_invalid_argument();
        }

        let (frames, frame_count) = match runtime::registry::materialize_build(view) {
            Ok(result) => result,
            Err(rc) => return rc,
        };

        ptr::write(out_frames, frames);
        ptr::write(out_frame_count, frame_count);
        err_ok()
    })
}

#[no_mangle]
pub unsafe extern "C" fn psdj_internal_test_hold_materialize_guard(
    view: *const pure_simdjson_value_view_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("psdj_internal_test_hold_materialize_guard", || {
        runtime::registry::test_hold_materialize_guard(view)
    })
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
        match runtime::implementation_name_len() {
            Ok(len) => write_out(out_len, len),
            Err(rc) => rc,
        }
    })
}

/// Copy the active implementation name into caller-owned storage.
///
/// `*out_written` is written with the required byte count whenever `out_written` itself is
/// non-null, regardless of the return code. Callers can read the size report on success, on
/// `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`, and also on `PURE_SIMDJSON_ERR_INVALID_ARGUMENT`
/// caused by a null `dst` with sufficient `dst_cap`.
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
    ffi_wrap("pure_simdjson_copy_implementation_name", || {
        runtime::copy_implementation_name(dst, dst_cap, out_written)
    })
}

/// Reset the diagnostic native allocator telemetry epoch.
///
/// Existing live native allocations remain valid, but future snapshots exclude them from the
/// reported counters until they are reallocated in the new epoch.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_native_alloc_stats_reset() -> pure_simdjson_error_code_t {
    ffi_wrap(
        "pure_simdjson_native_alloc_stats_reset",
        || match runtime::native_alloc_stats_reset() {
            Ok(()) => err_ok(),
            Err(rc) => rc,
        },
    )
}

/// Snapshot the diagnostic native allocator counters for the current telemetry epoch.
///
/// # Safety
/// `out_stats` must point to writable `pure_simdjson_native_alloc_stats_t` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_native_alloc_stats_snapshot(
    out_stats: *mut pure_simdjson_native_alloc_stats_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_native_alloc_stats_snapshot", || unsafe {
        match runtime::native_alloc_stats_snapshot() {
            Ok(stats) => write_out(out_stats, stats),
            Err(rc) => rc,
        }
    })
}

/// Allocate a parser handle.
///
/// # Safety
/// `out_parser` must be a valid writable pointer to a `pure_simdjson_parser_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_new(
    out_parser: *mut pure_simdjson_parser_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_new", || unsafe {
        if out_parser.is_null() {
            return err_invalid_argument();
        }

        if let Err(rc) = reject_fallback_implementation() {
            return rc;
        }

        match runtime::registry::parser_new() {
            Ok(parser) => write_out(out_parser, parser),
            Err(rc) => rc,
        }
    })
}

/// Release a parser handle after all associated documents have been freed.
///
/// Returns `PURE_SIMDJSON_ERR_PARSER_BUSY` while a live document still belongs to `parser`.
///
/// # Safety
/// `parser` must be a parser handle previously returned by this library. The sentinel `0` and
/// forged values are invalid.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_free(
    parser: pure_simdjson_parser_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_free", || {
        runtime::registry::parser_free(parser)
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

/// Report the byte length of the parser's last diagnostic message.
///
/// # Safety
/// `parser` must be a live parser handle from this library. `out_len` must be a valid writable
/// pointer to a `usize`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_len(
    parser: pure_simdjson_parser_t,
    out_len: *mut usize,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_get_last_error_len", || unsafe {
        match runtime::registry::parser_last_error_len(parser) {
            Ok(len) => write_out(out_len, len),
            Err(rc) => rc,
        }
    })
}

/// Copy the parser's last diagnostic message into caller-owned storage.
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
        runtime::registry::parser_copy_last_error(parser, dst, dst_cap, out_written)
    })
}

/// Report the byte offset associated with the parser's last failure.
///
/// # Safety
/// `parser` must be a live parser handle from this library. `out_offset` must be a valid writable
/// pointer to a `u64`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_get_last_error_offset(
    parser: pure_simdjson_parser_t,
    out_offset: *mut u64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_parser_get_last_error_offset", || unsafe {
        match runtime::registry::parser_last_error_offset(parser) {
            Ok(offset) => write_out(out_offset, offset),
            Err(rc) => rc,
        }
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
/// # Safety
/// `doc` must be a document handle previously returned by this library. The sentinel `0` and
/// forged values are invalid.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_free(
    doc: pure_simdjson_doc_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_doc_free", || {
        runtime::registry::doc_free(doc)
    })
}

/// Resolve the root value view for a live document handle.
///
/// The returned view's `kind_hint` is `PURE_SIMDJSON_VALUE_KIND_INVALID` for roots whose value
/// kind cannot be classified (for example, BIGINT). The canonical precision-loss error surfaces
/// at `pure_simdjson_element_type`, not here.
///
/// # Safety
/// `doc` must be a live document handle from this library. `out_root` must be a valid writable
/// pointer to a `pure_simdjson_value_view_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_doc_root(
    doc: pure_simdjson_doc_t,
    out_root: *mut pure_simdjson_value_view_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_doc_root", || unsafe {
        match runtime::registry::doc_root(doc) {
            Ok(root) => write_out(out_root, root),
            Err(rc) => rc,
        }
    })
}

/// Report the value kind for a document-tied view.
///
/// Returns `PURE_SIMDJSON_ERR_PRECISION_LOSS` for BIGINT values and
/// `PURE_SIMDJSON_ERR_INVALID_HANDLE` when reserved bits are non-zero or the root tag is invalid.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_type` must point to writable `u32` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_type(
    view: *const pure_simdjson_value_view_t,
    out_type: *mut u32,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_type", || unsafe {
        match runtime::registry::element_type(view) {
            Ok(value_kind) => write_out(out_type, value_kind),
            Err(rc) => rc,
        }
    })
}

/// Decode the referenced value as `int64_t`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `i64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_int64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut i64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_int64", || unsafe {
        match runtime::registry::element_get_int64(view) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
    })
}

/// Decode the referenced value as `uint64_t`.
///
/// Negative integers return `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE`; non-uint64 kinds return
/// `PURE_SIMDJSON_ERR_WRONG_TYPE`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `u64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_uint64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_uint64", || unsafe {
        match runtime::registry::element_get_uint64(view) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
    })
}

/// Decode the referenced value as `double`.
///
/// Integral values that cannot be represented exactly as `double` return
/// `PURE_SIMDJSON_ERR_PRECISION_LOSS`; non-numeric kinds return `PURE_SIMDJSON_ERR_WRONG_TYPE`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `f64` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_float64(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut f64,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_float64", || unsafe {
        match runtime::registry::element_get_float64(view) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
    })
}

/// Copy the referenced string value into a newly allocated byte buffer.
///
/// The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
/// `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
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
    ffi_wrap("pure_simdjson_element_get_string", || unsafe {
        if out_ptr.is_null() || out_len.is_null() {
            return err_invalid_argument();
        }

        match runtime::registry::element_get_string(view) {
            Ok((ptr_value, len)) => {
                ptr::write(out_ptr, ptr_value);
                ptr::write(out_len, len);
                err_ok()
            }
            Err(rc) => rc,
        }
    })
}

/// Release memory previously returned by `pure_simdjson_element_get_string`.
/// The empty-string sentinel is `ptr == NULL && len == 0`.
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
        runtime::registry::bytes_free(ptr, len)
    })
}

/// Decode the referenced value as a C `uint8_t` boolean.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_value` must point to writable `u8` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_get_bool(
    view: *const pure_simdjson_value_view_t,
    out_value: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_get_bool", || unsafe {
        match runtime::registry::element_get_bool(view) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
    })
}

/// Report whether the referenced value is JSON `null`.
///
/// # Safety
/// `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
/// `out_is_null` must point to writable `u8` storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_element_is_null(
    view: *const pure_simdjson_value_view_t,
    out_is_null: *mut u8,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_element_is_null", || unsafe {
        match runtime::registry::element_is_null(view) {
            Ok(value) => write_out(out_is_null, value),
            Err(rc) => rc,
        }
    })
}

/// Initialize array iterator state from an array-valued view.
///
/// # Safety
/// `array_view` must point to a readable array-valued `pure_simdjson_value_view_t` derived from a
/// live document. `out_iter` must point to writable iterator storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_array_iter_new(
    array_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_array_iter_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_array_iter_new", || unsafe {
        match runtime::registry::array_iter_new(array_view) {
            Ok(iter) => write_out(out_iter, iter),
            Err(rc) => rc,
        }
    })
}

/// Advance an array iterator and return the next value view plus a done flag.
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
    ffi_wrap("pure_simdjson_array_iter_next", || unsafe {
        if iter.is_null() || out_value.is_null() || out_done.is_null() {
            return err_invalid_argument();
        }

        match runtime::registry::array_iter_next(iter) {
            Ok(step) => {
                ptr::write(iter, step.iter);
                ptr::write(out_value, step.value);
                ptr::write(out_done, step.done);
                err_ok()
            }
            Err(rc) => rc,
        }
    })
}

/// Initialize object iterator state from an object-valued view.
///
/// # Safety
/// `object_view` must point to a readable object-valued `pure_simdjson_value_view_t` derived from
/// a live document. `out_iter` must point to writable iterator storage.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_object_iter_new(
    object_view: *const pure_simdjson_value_view_t,
    out_iter: *mut pure_simdjson_object_iter_t,
) -> pure_simdjson_error_code_t {
    ffi_wrap("pure_simdjson_object_iter_new", || unsafe {
        match runtime::registry::object_iter_new(object_view) {
            Ok(iter) => write_out(out_iter, iter),
            Err(rc) => rc,
        }
    })
}

/// Advance an object iterator and return the next key/value pair plus a done flag.
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
    ffi_wrap("pure_simdjson_object_iter_next", || unsafe {
        if iter.is_null() || out_key.is_null() || out_value.is_null() || out_done.is_null() {
            return err_invalid_argument();
        }

        match runtime::registry::object_iter_next(iter) {
            Ok(step) => {
                ptr::write(iter, step.iter);
                ptr::write(out_key, step.key);
                ptr::write(out_value, step.value);
                ptr::write(out_done, step.done);
                err_ok()
            }
            Err(rc) => rc,
        }
    })
}

/// Look up one object field by key and return its value view through `out_value`.
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
    ffi_wrap("pure_simdjson_object_get_field", || unsafe {
        if out_value.is_null() {
            return err_invalid_argument();
        }
        if key_len != 0 && key_ptr.is_null() {
            return err_invalid_argument();
        }

        let key = if key_len == 0 {
            &[][..]
        } else {
            slice::from_raw_parts(key_ptr, key_len)
        };

        match runtime::registry::object_get_field(object_view, key) {
            Ok(value) => write_out(out_value, value),
            Err(rc) => rc,
        }
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
        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(
            stdout.contains("running 1 test"),
            "subprocess filter should execute exactly one test: status={:?}, stdout={stdout}, stderr={stderr}",
            output.status,
        );
    }

    #[test]
    fn abi_version_getter_returns_the_pinned_constant() {
        let mut abi_version = 0_u32;

        let rc = unsafe { pure_simdjson_get_abi_version(&mut abi_version) };

        assert_eq!(rc, err_ok());
        assert_eq!(PURE_SIMDJSON_ABI_VERSION, 0x0001_0001);
        assert_eq!(abi_version, 0x0001_0001);
    }

    #[test]
    fn implementation_name_probe_reports_required_length() {
        let expected_len = runtime::implementation_name()
            .expect("bridge implementation name should be available")
            .len();
        let mut written = 0_usize;

        let rc =
            unsafe { pure_simdjson_copy_implementation_name(ptr::null_mut(), 0, &mut written) };

        assert_eq!(rc, err_buffer_too_small());
        assert_eq!(written, expected_len);
    }

    #[test]
    fn implementation_name_rejects_null_destination_when_capacity_is_sufficient() {
        let expected_len = runtime::implementation_name()
            .expect("bridge implementation name should be available")
            .len();
        let mut written = 0_usize;

        let rc = unsafe {
            pure_simdjson_copy_implementation_name(ptr::null_mut(), expected_len, &mut written)
        };

        assert_eq!(rc, err_invalid_argument());
        assert_eq!(written, expected_len);
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
    fn assert_subprocess_ran_exactly_one_test_includes_status_and_stderr_on_failure() {
        let output = Command::new(env::current_exe().expect("test binary path"))
            .arg("--pure-simdjson-invalid-libtest-flag")
            .output()
            .expect("spawn invalid libtest subprocess");

        let panic = std::panic::catch_unwind(|| assert_subprocess_ran_exactly_one_test(&output))
            .expect_err("helper should reject subprocesses that do not run exactly one test");
        let message = panic_payload_message(panic.as_ref());

        assert!(
            message.contains("status="),
            "failure context should include the subprocess exit status: {message}",
        );
        assert!(
            message.contains("stderr="),
            "failure context should include subprocess stderr: {message}",
        );
        assert!(
            message.contains("pure-simdjson-invalid-libtest-flag"),
            "failure context should preserve stderr details from libtest: {message}",
        );
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

        assert_eq!(
            message,
            format!("non-string panic payload ({expected_type_id})")
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
    fn bytes_free_rejects_null_pointer_with_nonzero_length() {
        assert_eq!(
            unsafe { pure_simdjson_bytes_free(ptr::null_mut(), 1) },
            pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_ARGUMENT
        );
    }

    #[test]
    fn bytes_free_accepts_empty_string_sentinel() {
        assert_eq!(
            unsafe { pure_simdjson_bytes_free(ptr::null_mut(), 0) },
            pure_simdjson_error_code_t::PURE_SIMDJSON_OK
        );
    }
}
