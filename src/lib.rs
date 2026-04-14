#![allow(non_camel_case_types)]
#![deny(clippy::missing_safety_doc)]

use core::ptr;

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
/// `state0`, `state1`, and `tag` are implementation-owned. `reserved` stays pinned for future
/// contract growth and callers must leave it untouched.
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
/// `state0`, `state1`, and `tag` are implementation-owned. `reserved` stays pinned for future
/// contract growth and callers must leave it untouched.
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
const fn err_ok() -> i32 {
    pure_simdjson_error_code_t::PURE_SIMDJSON_OK as i32
}

#[inline]
const fn err_invalid_argument() -> i32 {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_ARGUMENT as i32
}

#[inline]
const fn err_buffer_too_small() -> i32 {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL as i32
}

#[inline]
const fn err_internal() -> i32 {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL as i32
}

fn write_out<T>(out: *mut T, value: T) -> i32 {
    if out.is_null() {
        return err_invalid_argument();
    }

    unsafe {
        ptr::write(out, value);
    }

    err_ok()
}

fn copy_out_bytes(src: &[u8], dst: *mut u8, dst_cap: usize, out_written: *mut usize) -> i32 {
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
fn phase1_contract_stub() -> i32 {
    err_internal()
}

/// Write the packed ABI version expected by Go-side compatibility checks.
///
/// # Safety
/// `out_version` must be a valid writable pointer to a `u32`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_abi_version(out_version: *mut u32) -> i32 {
    write_out(out_version, PURE_SIMDJSON_ABI_VERSION)
}

/// Report the byte length of the active implementation name.
///
/// # Safety
/// `out_len` must be a valid writable pointer to a `usize`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_implementation_name_len(out_len: *mut usize) -> i32 {
    write_out(out_len, contract_only_implementation_name().len())
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
) -> i32 {
    copy_out_bytes(
        contract_only_implementation_name(),
        dst,
        dst_cap,
        out_written,
    )
}

/// Allocate a parser handle.
///
/// Phase 1 status: contract-only stub. This export is present to lock the ABI surface and
/// currently returns `PURE_SIMDJSON_ERR_INTERNAL`.
///
/// # Safety
/// `out_parser` must be a valid writable pointer to a `pure_simdjson_parser_t`.
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_new(out_parser: *mut pure_simdjson_parser_t) -> i32 {
    let _ = out_parser;
    phase1_contract_stub()
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
pub unsafe extern "C" fn pure_simdjson_parser_free(parser: pure_simdjson_parser_t) -> i32 {
    let _ = parser;
    phase1_contract_stub()
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
) -> i32 {
    let _ = (parser, input_ptr, input_len, out_doc);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (parser, out_len);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (parser, dst, dst_cap, out_written);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (parser, out_offset);
    phase1_contract_stub()
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
pub unsafe extern "C" fn pure_simdjson_doc_free(doc: pure_simdjson_doc_t) -> i32 {
    let _ = doc;
    phase1_contract_stub()
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
) -> i32 {
    let _ = (doc, out_root);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_type);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_value);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_value);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_value);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_ptr, out_len);
    phase1_contract_stub()
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
pub unsafe extern "C" fn pure_simdjson_bytes_free(ptr: *mut u8, len: usize) -> i32 {
    let _ = (ptr, len);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_value);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (view, out_is_null);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (array_view, out_iter);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (iter, out_value, out_done);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (object_view, out_iter);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (iter, out_key, out_value, out_done);
    phase1_contract_stub()
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
) -> i32 {
    let _ = (object_view, key_ptr, key_len, out_value);
    phase1_contract_stub()
}

#[cfg(test)]
mod tests {
    use super::*;

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
}
