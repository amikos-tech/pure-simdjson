use std::ptr;

use crate::{pure_simdjson_error_code_t, pure_simdjson_value_kind_t};

pub(crate) mod registry;

#[allow(unused_imports)]
pub(crate) use registry::ParserState;

pub(crate) const UNKNOWN_ERROR_OFFSET: u64 = u64::MAX;
pub(crate) const ROOT_VIEW_TAG: u64 = u64::from_le_bytes(*b"PSDJROOT");

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

    fn psimdjson_parser_new(
        out_parser: *mut *mut psimdjson_parser,
    ) -> pure_simdjson_error_code_t;
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
    fn psimdjson_element_get_int64(
        element: *const psimdjson_element,
        out_value: *mut i64,
    ) -> pure_simdjson_error_code_t;

    fn psimdjson_test_force_cpp_exception() -> pure_simdjson_error_code_t;
}

pub(crate) struct NativeParsedDoc {
    pub(crate) doc_ptr: usize,
    pub(crate) root_ptr: usize,
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
) -> Result<NativeParsedDoc, pure_simdjson_error_code_t> {
    let mut doc = ptr::null_mut();
    let rc = unsafe {
        psimdjson_parser_parse(
            parser_ptr as *mut psimdjson_parser,
            input.as_ptr(),
            input.len(),
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
        let _ = unsafe { psimdjson_doc_free(doc) };
        return Err(root_rc);
    }
    if root.is_null() {
        let _ = unsafe { psimdjson_doc_free(doc) };
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
        psimdjson_parser_get_last_error_offset(
            parser_ptr as *const psimdjson_parser,
            &mut offset,
        )
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

pub(crate) fn native_element_type(
    element_ptr: usize,
) -> Result<u32, pure_simdjson_error_code_t> {
    let mut kind = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_INVALID;
    let rc = unsafe {
        psimdjson_element_type(
            element_ptr as *const psimdjson_element,
            &mut kind,
        )
    };
    if rc == err_ok() {
        Ok(kind as u32)
    } else {
        Err(rc)
    }
}

pub(crate) fn native_element_get_int64(
    element_ptr: usize,
) -> Result<i64, pure_simdjson_error_code_t> {
    let mut value = 0_i64;
    let rc = unsafe {
        psimdjson_element_get_int64(
            element_ptr as *const psimdjson_element,
            &mut value,
        )
    };
    if rc == err_ok() {
        Ok(value)
    } else {
        Err(rc)
    }
}

pub(crate) fn selected_implementation_name_for_parser_new(
) -> Result<Vec<u8>, pure_simdjson_error_code_t> {
    if let Some(value) = std::env::var_os("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION") {
        if value == "fallback" {
            return Ok(b"fallback".to_vec());
        }
    }
    implementation_name()
}

pub(crate) fn fallback_allowed_for_tests() -> bool {
    matches!(
        std::env::var("PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS"),
        Ok(value) if value == "1"
    )
}

#[doc(hidden)]
pub fn test_force_cpp_exception_for_tests() -> pure_simdjson_error_code_t {
    unsafe { psimdjson_test_force_cpp_exception() }
}
