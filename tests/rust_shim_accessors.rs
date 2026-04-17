use std::{ptr, slice};

use pure_simdjson::{
    pure_simdjson_bytes_free, pure_simdjson_doc_free, pure_simdjson_doc_root, pure_simdjson_doc_t,
    pure_simdjson_element_get_bool, pure_simdjson_element_get_float64,
    pure_simdjson_element_get_string, pure_simdjson_element_get_uint64,
    pure_simdjson_element_is_null, pure_simdjson_element_type,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_parse,
    pure_simdjson_parser_t,
    pure_simdjson_value_kind_t::{PURE_SIMDJSON_VALUE_KIND_BOOL, PURE_SIMDJSON_VALUE_KIND_NULL},
    pure_simdjson_value_view_t,
};

const DESC_VIEW_TAG: u64 = u64::from_le_bytes(*b"PSDJDESC");

fn parser_new() -> pure_simdjson_parser_t {
    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    parser
}

fn parser_parse_literal(parser: pure_simdjson_parser_t, json: &[u8]) -> pure_simdjson_doc_t {
    let mut doc = 0_u64;
    let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };
    assert_eq!(rc, PURE_SIMDJSON_OK, "failed to parse {:?}", json);
    assert_ne!(doc, 0);
    doc
}

fn doc_root(doc: pure_simdjson_doc_t) -> pure_simdjson_value_view_t {
    let mut root = pure_simdjson_value_view_t::default();
    let rc = unsafe { pure_simdjson_doc_root(doc, &mut root) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    root
}

fn cleanup(parser: pure_simdjson_parser_t, doc: pure_simdjson_doc_t) {
    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn scalar_accessors_read_uint64_float64_bool_and_null_roots() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"18446744073709551615");
    let root = doc_root(doc);
    let mut value = 0_u64;
    let rc = unsafe { pure_simdjson_element_get_uint64(&root, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(value, u64::MAX);
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"1.5");
    let root = doc_root(doc);
    let mut value = 0_f64;
    let rc = unsafe { pure_simdjson_element_get_float64(&root, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(value, 1.5);
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"true");
    let root = doc_root(doc);
    let mut value = 0_u8;
    let rc = unsafe { pure_simdjson_element_get_bool(&root, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(value, 1);
    let mut kind = 0_u32;
    let rc = unsafe { pure_simdjson_element_type(&root, &mut kind) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(kind, PURE_SIMDJSON_VALUE_KIND_BOOL as u32);
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"null");
    let root = doc_root(doc);
    let mut is_null = 0_u8;
    let rc = unsafe { pure_simdjson_element_is_null(&root, &mut is_null) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(is_null, 1);
    let mut kind = 0_u32;
    let rc = unsafe { pure_simdjson_element_type(&root, &mut kind) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(kind, PURE_SIMDJSON_VALUE_KIND_NULL as u32);
    cleanup(parser, doc);
}

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
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(ptr, len) },
        PURE_SIMDJSON_OK
    );

    cleanup(parser, doc);
}

#[test]
fn empty_string_bytes_free_round_trip_uses_empty_string_sentinel() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#""""#);
    let root = doc_root(doc);

    let mut ptr = ptr::null_mut();
    let mut len = 0_usize;
    let rc = unsafe { pure_simdjson_element_get_string(&root, &mut ptr, &mut len) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(
        ptr.is_null(),
        "empty string should use the null buffer sentinel"
    );
    assert_eq!(len, 0);
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(ptr, len) },
        PURE_SIMDJSON_OK
    );

    cleanup(parser, doc);
}

#[test]
fn typed_getters_report_wrong_type_for_null_and_other_values() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"null");
    let root = doc_root(doc);

    let mut string_ptr = ptr::null_mut();
    let mut string_len = 0_usize;
    let string_rc =
        unsafe { pure_simdjson_element_get_string(&root, &mut string_ptr, &mut string_len) };
    assert_eq!(string_rc, PURE_SIMDJSON_ERR_WRONG_TYPE);

    let mut bool_value = 0_u8;
    let bool_rc = unsafe { pure_simdjson_element_get_bool(&root, &mut bool_value) };
    assert_eq!(bool_rc, PURE_SIMDJSON_ERR_WRONG_TYPE);
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#""not-a-number""#);
    let root = doc_root(doc);
    let mut uint_value = 0_u64;
    let rc = unsafe { pure_simdjson_element_get_uint64(&root, &mut uint_value) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_WRONG_TYPE);
    cleanup(parser, doc);
}

#[test]
fn bytes_free_rejects_null_pointer_with_length() {
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(ptr::null_mut(), 1) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
}

#[test]
fn descendant_tag_and_reserved_bits_return_invalid_handle() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"18446744073709551615");
    let mut root = doc_root(doc);

    root.state0 = 1;
    root.state1 = DESC_VIEW_TAG;
    let mut value = 0_u64;
    let rc = unsafe { pure_simdjson_element_get_uint64(&root, &mut value) };
    assert_eq!(
        rc, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "descendant tag must validate through the registry"
    );

    let mut root = doc_root(doc);
    root.reserved = 1;
    let rc = unsafe { pure_simdjson_element_get_uint64(&root, &mut value) };
    assert_eq!(
        rc, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "reserved bits must invalidate the handle"
    );

    cleanup(parser, doc);
}
