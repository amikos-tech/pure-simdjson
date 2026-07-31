use pure_simdjson::{
    pure_simdjson_array_at, pure_simdjson_array_len, pure_simdjson_doc_free,
    pure_simdjson_doc_root, pure_simdjson_doc_t, pure_simdjson_element_at_path,
    pure_simdjson_element_at_pointer, pure_simdjson_element_get_int64,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE, PURE_SIMDJSON_ERR_INVALID_PATH,
        PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_object_size, pure_simdjson_parser_free, pure_simdjson_parser_new,
    pure_simdjson_parser_parse, pure_simdjson_parser_t, pure_simdjson_value_view_t,
};

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

fn read_int64(view: &pure_simdjson_value_view_t) -> i64 {
    let mut value = 0_i64;
    let rc = unsafe { pure_simdjson_element_get_int64(view, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    value
}

#[test]
fn element_at_pointer_resolves_nested_field() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"a":{"b":42}}"#);
    let root = doc_root(doc);
    let pointer = b"/a/b";
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe {
        pure_simdjson_element_at_pointer(&root, pointer.as_ptr(), pointer.len(), &mut resolved)
    };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(read_int64(&resolved), 42);
    cleanup(parser, doc);
}

#[test]
fn element_at_pointer_rejects_malformed_pointer() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"a":{"b":42}}"#);
    let root = doc_root(doc);
    let pointer = b"a/b";
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe {
        pure_simdjson_element_at_pointer(&root, pointer.as_ptr(), pointer.len(), &mut resolved)
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_PATH);
    cleanup(parser, doc);
}

#[test]
fn element_at_pointer_out_of_range_array_index_returns_index_out_of_range() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[1,2,3]");
    let root = doc_root(doc);
    let pointer = b"/5";
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe {
        pure_simdjson_element_at_pointer(&root, pointer.as_ptr(), pointer.len(), &mut resolved)
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE);
    cleanup(parser, doc);
}

#[test]
fn element_at_path_resolves_dot_notation() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"a":{"b":42}}"#);
    let root = doc_root(doc);
    let path = b".a.b";
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc =
        unsafe { pure_simdjson_element_at_path(&root, path.as_ptr(), path.len(), &mut resolved) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(read_int64(&resolved), 42);
    cleanup(parser, doc);
}

#[test]
fn element_at_path_rejects_bare_field_name() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"a":42}"#);
    let root = doc_root(doc);
    let path = b"a";
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc =
        unsafe { pure_simdjson_element_at_path(&root, path.as_ptr(), path.len(), &mut resolved) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_PATH);
    cleanup(parser, doc);
}

#[test]
fn array_at_returns_indexed_element() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[10,20,30]");
    let root = doc_root(doc);
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe { pure_simdjson_array_at(&root, 1, &mut resolved) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(read_int64(&resolved), 20);
    cleanup(parser, doc);
}

#[test]
fn array_at_out_of_range_returns_index_out_of_range() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[10,20,30]");
    let root = doc_root(doc);
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe { pure_simdjson_array_at(&root, 5, &mut resolved) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE);
    cleanup(parser, doc);
}

#[test]
fn array_at_wrong_type_returns_wrong_type() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);
    let mut resolved = pure_simdjson_value_view_t::default();

    let rc = unsafe { pure_simdjson_array_at(&root, 0, &mut resolved) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_WRONG_TYPE);
    cleanup(parser, doc);
}

#[test]
fn array_len_reports_direct_child_count() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[1,2,3,4]");
    let root = doc_root(doc);
    let mut len = 0_u64;

    let rc = unsafe { pure_simdjson_array_len(&root, &mut len) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(len, 4);
    cleanup(parser, doc);
}

#[test]
fn empty_array_len_is_zero() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[]");
    let root = doc_root(doc);
    let mut len = u64::MAX;

    let rc = unsafe { pure_simdjson_array_len(&root, &mut len) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(len, 0);
    cleanup(parser, doc);
}

#[test]
fn array_len_wrong_type_returns_wrong_type() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"{}");
    let root = doc_root(doc);
    let mut len = 0_u64;

    let rc = unsafe { pure_simdjson_array_len(&root, &mut len) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_WRONG_TYPE);
    cleanup(parser, doc);
}

#[test]
fn object_size_reports_direct_field_count() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"a":1,"b":2}"#);
    let root = doc_root(doc);
    let mut size = 0_u64;

    let rc = unsafe { pure_simdjson_object_size(&root, &mut size) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(size, 2);
    cleanup(parser, doc);
}

#[test]
fn empty_object_size_is_zero() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"{}");
    let root = doc_root(doc);
    let mut size = u64::MAX;

    let rc = unsafe { pure_simdjson_object_size(&root, &mut size) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(size, 0);
    cleanup(parser, doc);
}

#[test]
fn object_size_wrong_type_returns_wrong_type() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[]");
    let root = doc_root(doc);
    let mut size = 0_u64;

    let rc = unsafe { pure_simdjson_object_size(&root, &mut size) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_WRONG_TYPE);
    cleanup(parser, doc);
}
