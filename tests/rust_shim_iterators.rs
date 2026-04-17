use std::{ptr, slice};

use pure_simdjson::{
    pure_simdjson_array_iter_new, pure_simdjson_array_iter_next, pure_simdjson_array_iter_t,
    pure_simdjson_bytes_free, pure_simdjson_doc_free, pure_simdjson_doc_root, pure_simdjson_doc_t,
    pure_simdjson_element_get_int64, pure_simdjson_element_get_string,
    pure_simdjson_element_is_null,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND, PURE_SIMDJSON_ERR_INVALID_HANDLE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_object_get_field, pure_simdjson_object_iter_new, pure_simdjson_object_iter_next,
    pure_simdjson_object_iter_t, pure_simdjson_parser_free, pure_simdjson_parser_new,
    pure_simdjson_parser_parse, pure_simdjson_parser_t, pure_simdjson_value_view_t,
};

const ARRAY_ITER_TAG: u16 = u16::from_le_bytes(*b"AR");
const OBJECT_ITER_TAG: u16 = u16::from_le_bytes(*b"OB");
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

fn read_string(view: &pure_simdjson_value_view_t) -> String {
    let mut ptr = ptr::null_mut();
    let mut len = 0_usize;
    let rc = unsafe { pure_simdjson_element_get_string(view, &mut ptr, &mut len) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    let bytes = if len == 0 {
        Vec::new()
    } else {
        unsafe { slice::from_raw_parts(ptr, len) }.to_vec()
    };
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(ptr, len) },
        PURE_SIMDJSON_OK
    );
    String::from_utf8(bytes).expect("valid utf-8 key")
}

fn read_int64(view: &pure_simdjson_value_view_t) -> i64 {
    let mut value = 0_i64;
    let rc = unsafe { pure_simdjson_element_get_int64(view, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    value
}

#[test]
fn array_iteration_preserves_order_and_iterator_next_after_done() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"[1,2,3]"#);
    let root = doc_root(doc);

    let mut iter = pure_simdjson_array_iter_t::default();
    let rc = unsafe { pure_simdjson_array_iter_new(&root, &mut iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(iter.tag, ARRAY_ITER_TAG);

    let mut value = pure_simdjson_value_view_t::default();
    let mut done = 1_u8;
    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_int64(&value), 1);

    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_int64(&value), 2);

    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_int64(&value), 3);

    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 1);

    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK, "next after done must stay clean");
    assert_eq!(done, 1);

    cleanup(parser, doc);
}

#[test]
fn empty_array_and_empty_object_return_done_immediately() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"[]"#);
    let root = doc_root(doc);

    let mut iter = pure_simdjson_array_iter_t::default();
    let rc = unsafe { pure_simdjson_array_iter_new(&root, &mut iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);

    let mut value = pure_simdjson_value_view_t::default();
    let mut done = 0_u8;
    let rc = unsafe { pure_simdjson_array_iter_next(&mut iter, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 1, "empty array should report done immediately");
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{}"#);
    let root = doc_root(doc);

    let mut iter = pure_simdjson_object_iter_t::default();
    let rc = unsafe { pure_simdjson_object_iter_new(&root, &mut iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);

    let mut key = pure_simdjson_value_view_t::default();
    let mut obj_value = pure_simdjson_value_view_t::default();
    let mut obj_done = 0_u8;
    let rc = unsafe {
        pure_simdjson_object_iter_next(&mut iter, &mut key, &mut obj_value, &mut obj_done)
    };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(obj_done, 1, "empty object should report done immediately");
    cleanup(parser, doc);
}

#[test]
fn object_iteration_and_field_lookup_distinguish_missing_and_null() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"":7,"a":1,"b":null}"#);
    let root = doc_root(doc);

    let mut iter = pure_simdjson_object_iter_t::default();
    let rc = unsafe { pure_simdjson_object_iter_new(&root, &mut iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(iter.tag, OBJECT_ITER_TAG);

    let mut key = pure_simdjson_value_view_t::default();
    let mut value = pure_simdjson_value_view_t::default();
    let mut done = 1_u8;
    let rc = unsafe { pure_simdjson_object_iter_next(&mut iter, &mut key, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_string(&key), "");
    assert_eq!(read_int64(&value), 7);

    let rc = unsafe { pure_simdjson_object_iter_next(&mut iter, &mut key, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_string(&key), "a");
    assert_eq!(read_int64(&value), 1);

    let rc = unsafe { pure_simdjson_object_iter_next(&mut iter, &mut key, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 0);
    assert_eq!(read_string(&key), "b");
    let mut is_null = 0_u8;
    let rc = unsafe { pure_simdjson_element_is_null(&value, &mut is_null) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(is_null, 1);

    let rc = unsafe { pure_simdjson_object_iter_next(&mut iter, &mut key, &mut value, &mut done) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(done, 1);

    let mut field = pure_simdjson_value_view_t::default();
    let rc = unsafe { pure_simdjson_object_get_field(&root, b"".as_ptr(), 0, &mut field) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(read_int64(&field), 7);

    let rc = unsafe { pure_simdjson_object_get_field(&root, b"b".as_ptr(), 1, &mut field) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    let rc = unsafe { pure_simdjson_element_is_null(&field, &mut is_null) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(is_null, 1);

    let rc = unsafe { pure_simdjson_object_get_field(&root, b"missing".as_ptr(), 7, &mut field) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND);

    cleanup(parser, doc);
}

#[test]
fn invalid_handle_reserved_and_tag_bits_are_rejected() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"[1]"#);
    let root = doc_root(doc);

    let mut array_iter = pure_simdjson_array_iter_t::default();
    let rc = unsafe { pure_simdjson_array_iter_new(&root, &mut array_iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);

    let mut value = pure_simdjson_value_view_t::default();
    let mut done = 0_u8;
    array_iter.reserved = 1;
    let rc = unsafe { pure_simdjson_array_iter_next(&mut array_iter, &mut value, &mut done) };
    assert_eq!(
        rc, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "reserved bits must invalidate array iterators"
    );

    let mut object_root = doc_root(doc);
    object_root.state0 = 1;
    object_root.state1 = DESC_VIEW_TAG;
    let rc = unsafe { pure_simdjson_array_iter_new(&object_root, &mut array_iter) };
    assert_eq!(
        rc, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "forged descendant-tag view must fail"
    );
    cleanup(parser, doc);

    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"k":1}"#);
    let root = doc_root(doc);

    let mut object_iter = pure_simdjson_object_iter_t::default();
    let rc = unsafe { pure_simdjson_object_iter_new(&root, &mut object_iter) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    object_iter.tag ^= 0xFFFF;

    let mut key = pure_simdjson_value_view_t::default();
    let mut obj_value = pure_simdjson_value_view_t::default();
    let mut obj_done = 0_u8;
    let rc = unsafe {
        pure_simdjson_object_iter_next(&mut object_iter, &mut key, &mut obj_value, &mut obj_done)
    };
    assert_eq!(
        rc, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "invalid tag must invalidate object iterators"
    );

    cleanup(parser, doc);
}
