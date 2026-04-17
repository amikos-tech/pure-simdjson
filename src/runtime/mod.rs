use std::{
    ptr,
    sync::{Mutex, OnceLock},
};

use crate::{pure_simdjson_error_code_t, pure_simdjson_value_kind_t};

pub(crate) mod registry;

#[allow(unused_imports)]
pub(crate) use registry::ParserState;

pub(crate) const UNKNOWN_ERROR_OFFSET: u64 = u64::MAX;
/// Marker stored in `pure_simdjson_value_view_t.state1` for root views returned by this runtime.
pub(crate) const ROOT_VIEW_TAG: u64 = u64::from_le_bytes(*b"PSDJROOT");
/// Marker stored in `pure_simdjson_value_view_t.state1` for descendant views returned by this runtime.
pub(crate) const DESC_VIEW_TAG: u64 = u64::from_le_bytes(*b"PSDJDESC");
static FORCED_IMPLEMENTATION_NAME: OnceLock<Option<Vec<u8>>> = OnceLock::new();
static FALLBACK_ALLOWED: OnceLock<bool> = OnceLock::new();
static TEST_FORCED_IMPLEMENTATION_OVERRIDE: OnceLock<Mutex<Option<Vec<u8>>>> = OnceLock::new();
static TEST_FALLBACK_ALLOWED_OVERRIDE: OnceLock<Mutex<Option<bool>>> = OnceLock::new();

#[repr(C)]
pub(crate) struct psimdjson_parser {
    _private: [u8; 0],
}

#[repr(C)]
pub(crate) struct psimdjson_doc {
    _private: [u8; 0],
}

#[repr(C)]
pub(crate) struct psimdjson_element {
    _private: [u8; 0],
}

unsafe extern "C" {
    fn psimdjson_get_implementation_name_len(out_len: *mut usize) -> pure_simdjson_error_code_t;
    fn psimdjson_copy_implementation_name(
        dst: *mut u8,
        dst_cap: usize,
        out_written: *mut usize,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_padding_bytes() -> usize;

    fn psimdjson_parser_new(out_parser: *mut *mut psimdjson_parser) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_free(parser: *mut psimdjson_parser) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_parse(
        parser: *mut psimdjson_parser,
        input_ptr: *const u8,
        input_len: usize,
        out_doc: *mut *mut psimdjson_doc,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_get_last_error_len(
        parser: *const psimdjson_parser,
        out_len: *mut usize,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_copy_last_error(
        parser: *const psimdjson_parser,
        dst: *mut u8,
        dst_cap: usize,
        out_written: *mut usize,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_parser_get_last_error_offset(
        parser: *const psimdjson_parser,
        out_offset: *mut u64,
    ) -> pure_simdjson_error_code_t;

    fn psimdjson_doc_free(doc: *mut psimdjson_doc) -> pure_simdjson_error_code_t;
    fn psimdjson_doc_root(
        doc: *mut psimdjson_doc,
        out_element: *mut *const psimdjson_element,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_type(
        element: *const psimdjson_element,
        out_kind: *mut pure_simdjson_value_kind_t,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_type_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_kind: *mut pure_simdjson_value_kind_t,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_get_int64_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_value: *mut i64,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_get_uint64_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_value: *mut u64,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_get_float64_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_value: *mut f64,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_get_string_view(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_ptr: *mut *const u8,
        out_len: *mut usize,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_get_bool_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_value: *mut u8,
    ) -> pure_simdjson_error_code_t;
    fn psimdjson_element_is_null_at(
        doc: *const psimdjson_doc,
        json_index: u64,
        out_is_null: *mut u8,
    ) -> pure_simdjson_error_code_t;

    fn psimdjson_test_force_cpp_exception() -> pure_simdjson_error_code_t;
}

pub(crate) struct NativeParsedDoc {
    pub(crate) doc_ptr: usize,
    pub(crate) root_ptr: usize,
}

#[inline]
fn lock_poison_tolerant<T>(mutex: &'static Mutex<T>) -> std::sync::MutexGuard<'static, T> {
    mutex
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

#[inline]
fn err_internal() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL
}

#[inline]
fn err_ok() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_OK
}

#[inline]
pub(crate) fn implementation_name_len() -> Result<usize, pure_simdjson_error_code_t> {
    let mut len = 0_usize;
    let rc = unsafe { psimdjson_get_implementation_name_len(&mut len) };
    if rc == err_ok() {
        Ok(len)
    } else {
        Err(rc)
    }
}

#[inline]
pub(crate) fn copy_implementation_name(
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    unsafe { psimdjson_copy_implementation_name(dst, dst_cap, out_written) }
}

pub(crate) fn implementation_name() -> Result<Vec<u8>, pure_simdjson_error_code_t> {
    let len = implementation_name_len()?;
    let mut bytes = vec![0_u8; len];
    let mut written = 0_usize;
    let rc = unsafe {
        psimdjson_copy_implementation_name(bytes.as_mut_ptr(), bytes.len(), &mut written)
    };
    if rc != err_ok() {
        return Err(rc);
    }
    bytes.truncate(written);
    Ok(bytes)
}

pub(crate) fn padding_bytes() -> Result<usize, pure_simdjson_error_code_t> {
    let padding = unsafe { psimdjson_padding_bytes() };
    if padding == 0 {
        Err(err_internal())
    } else {
        Ok(padding)
    }
}

pub(crate) fn native_parser_new() -> Result<usize, pure_simdjson_error_code_t> {
    let mut parser = ptr::null_mut();
    let rc = unsafe { psimdjson_parser_new(&mut parser) };
    if rc != err_ok() {
        return Err(rc);
    }
    if parser.is_null() {
        return Err(err_internal());
    }
    Ok(parser as usize)
}

pub(crate) fn native_parser_free(parser_ptr: usize) -> pure_simdjson_error_code_t {
    unsafe { psimdjson_parser_free(parser_ptr as *mut psimdjson_parser) }
}

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

pub(crate) fn native_parser_get_last_error_len(
    parser_ptr: usize,
) -> Result<usize, pure_simdjson_error_code_t> {
    let mut len = 0_usize;
    let rc = unsafe {
        psimdjson_parser_get_last_error_len(parser_ptr as *const psimdjson_parser, &mut len)
    };
    if rc == err_ok() {
        Ok(len)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_parser_copy_last_error(
    parser_ptr: usize,
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    unsafe {
        psimdjson_parser_copy_last_error(
            parser_ptr as *const psimdjson_parser,
            dst,
            dst_cap,
            out_written,
        )
    }
}

pub(crate) fn native_parser_get_last_error_offset(
    parser_ptr: usize,
) -> Result<u64, pure_simdjson_error_code_t> {
    let mut offset = UNKNOWN_ERROR_OFFSET;
    let rc = unsafe {
        psimdjson_parser_get_last_error_offset(parser_ptr as *const psimdjson_parser, &mut offset)
    };
    if rc == err_ok() {
        Ok(offset)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_doc_free(doc_ptr: usize) -> pure_simdjson_error_code_t {
    unsafe { psimdjson_doc_free(doc_ptr as *mut psimdjson_doc) }
}

pub(crate) fn native_element_type(element_ptr: usize) -> Result<u32, pure_simdjson_error_code_t> {
    let mut kind = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_INVALID;
    let rc = unsafe { psimdjson_element_type(element_ptr as *const psimdjson_element, &mut kind) };
    if rc == err_ok() {
        Ok(kind as u32)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_type_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<u32, pure_simdjson_error_code_t> {
    let mut kind = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_INVALID;
    let rc = unsafe {
        psimdjson_element_type_at(doc_ptr as *const psimdjson_doc, json_index, &mut kind)
    };
    if rc == err_ok() {
        Ok(kind as u32)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_int64_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<i64, pure_simdjson_error_code_t> {
    let mut value = 0_i64;
    let rc = unsafe {
        psimdjson_element_get_int64_at(doc_ptr as *const psimdjson_doc, json_index, &mut value)
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_uint64_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<u64, pure_simdjson_error_code_t> {
    let mut value = 0_u64;
    let rc = unsafe {
        psimdjson_element_get_uint64_at(doc_ptr as *const psimdjson_doc, json_index, &mut value)
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_float64_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<f64, pure_simdjson_error_code_t> {
    let mut value = 0_f64;
    let rc = unsafe {
        psimdjson_element_get_float64_at(doc_ptr as *const psimdjson_doc, json_index, &mut value)
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_string_view(
    doc_ptr: usize,
    json_index: u64,
) -> Result<(usize, usize), pure_simdjson_error_code_t> {
    let mut ptr = ptr::null();
    let mut len = 0_usize;
    let rc = unsafe {
        psimdjson_element_get_string_view(
            doc_ptr as *const psimdjson_doc,
            json_index,
            &mut ptr,
            &mut len,
        )
    };
    if rc == err_ok() {
        Ok((ptr as usize, len))
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_bool_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<u8, pure_simdjson_error_code_t> {
    let mut value = 0_u8;
    let rc = unsafe {
        psimdjson_element_get_bool_at(doc_ptr as *const psimdjson_doc, json_index, &mut value)
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_is_null_at(
    doc_ptr: usize,
    json_index: u64,
) -> Result<u8, pure_simdjson_error_code_t> {
    let mut value = 0_u8;
    let rc = unsafe {
        psimdjson_element_is_null_at(doc_ptr as *const psimdjson_doc, json_index, &mut value)
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn selected_implementation_name_for_parser_new(
) -> Result<Vec<u8>, pure_simdjson_error_code_t> {
    let override_lock = TEST_FORCED_IMPLEMENTATION_OVERRIDE.get_or_init(|| Mutex::new(None));
    if let Some(value) = lock_poison_tolerant(override_lock).clone() {
        return Ok(value);
    }

    if let Some(value) = FORCED_IMPLEMENTATION_NAME
        .get_or_init(|| {
            std::env::var("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION")
                .ok()
                .map(String::into_bytes)
        })
        .clone()
    {
        if value == b"fallback" {
            return Ok(value);
        }
        // Test-only env var; fail loud so a typo (e.g. "fallbck") does not silently no-op.
        panic!(
            "PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION only accepts \"fallback\"; got {:?}",
            String::from_utf8_lossy(&value)
        );
    }
    implementation_name()
}

pub(crate) fn fallback_allowed_for_tests() -> bool {
    let override_lock = TEST_FALLBACK_ALLOWED_OVERRIDE.get_or_init(|| Mutex::new(None));
    if let Some(value) = *lock_poison_tolerant(override_lock) {
        return value;
    }

    *FALLBACK_ALLOWED.get_or_init(|| {
        matches!(
            std::env::var("PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS"),
            Ok(value) if value == "1"
        )
    })
}

#[doc(hidden)]
pub fn test_set_forced_implementation_override(value: Option<&[u8]>) {
    let override_lock = TEST_FORCED_IMPLEMENTATION_OVERRIDE.get_or_init(|| Mutex::new(None));
    *lock_poison_tolerant(override_lock) = value.map(|bytes| bytes.to_vec());
}

#[doc(hidden)]
pub fn test_set_fallback_allowed_override(value: Option<bool>) {
    let override_lock = TEST_FALLBACK_ALLOWED_OVERRIDE.get_or_init(|| Mutex::new(None));
    *lock_poison_tolerant(override_lock) = value;
}

#[doc(hidden)]
pub fn test_force_cpp_exception_for_tests() -> pure_simdjson_error_code_t {
    unsafe { psimdjson_test_force_cpp_exception() }
}
