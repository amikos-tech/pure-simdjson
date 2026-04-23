use pure_simdjson::{
    pure_simdjson_doc_t,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INVALID_HANDLE, PURE_SIMDJSON_ERR_INVALID_JSON,
        PURE_SIMDJSON_ERR_PARSER_BUSY, PURE_SIMDJSON_OK,
    },
    pure_simdjson_object_get_field, pure_simdjson_doc_free, pure_simdjson_doc_root,
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_parse,
    pure_simdjson_parser_t,
    pure_simdjson_value_kind_t::{
        PURE_SIMDJSON_VALUE_KIND_ARRAY, PURE_SIMDJSON_VALUE_KIND_BOOL,
        PURE_SIMDJSON_VALUE_KIND_INT64, PURE_SIMDJSON_VALUE_KIND_NULL,
        PURE_SIMDJSON_VALUE_KIND_OBJECT, PURE_SIMDJSON_VALUE_KIND_STRING,
        PURE_SIMDJSON_VALUE_KIND_UINT64,
    },
    pure_simdjson_value_view_t,
};
use std::{ptr, slice};

#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
struct psdj_internal_frame_t {
    kind: u32,
    flags: u32,
    child_count: u32,
    reserved: u32,
    key_ptr: *const u8,
    key_len: usize,
    string_ptr: *const u8,
    string_len: usize,
    int64_value: i64,
    uint64_value: u64,
    float64_value: f64,
}

unsafe extern "C" {
    fn psdj_internal_materialize_build(
        view: *const pure_simdjson_value_view_t,
        out_frames: *mut *const psdj_internal_frame_t,
        out_frame_count: *mut usize,
    ) -> pure_simdjson::pure_simdjson_error_code_t;
    fn psdj_internal_test_hold_materialize_guard(
        view: *const pure_simdjson_value_view_t,
    ) -> pure_simdjson::pure_simdjson_error_code_t;
}

fn parser_new() -> pure_simdjson_parser_t {
    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    parser
}

fn parse_root(json: &[u8]) -> (pure_simdjson_parser_t, pure_simdjson_doc_t, pure_simdjson_value_view_t) {
    let parser = parser_new();
    let mut doc = 0_u64;
    let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };
    assert_eq!(rc, PURE_SIMDJSON_OK, "parse should succeed for {:?}", json);

    let mut root = pure_simdjson_value_view_t::default();
    let root_rc = unsafe { pure_simdjson_doc_root(doc, &mut root) };
    assert_eq!(root_rc, PURE_SIMDJSON_OK);
    (parser, doc, root)
}

fn object_get_field_view(
    object_view: &pure_simdjson_value_view_t,
    key: &[u8],
) -> pure_simdjson_value_view_t {
    let mut value = pure_simdjson_value_view_t::default();
    let rc =
        unsafe { pure_simdjson_object_get_field(object_view, key.as_ptr(), key.len(), &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    value
}

fn cleanup(parser: pure_simdjson_parser_t, doc: pure_simdjson_doc_t) {
    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(unsafe { pure_simdjson_parser_free(parser) }, PURE_SIMDJSON_OK);
}

fn build_frames(view: &pure_simdjson_value_view_t) -> Vec<psdj_internal_frame_t> {
    let mut frames = ptr::null();
    let mut frame_count = 0_usize;
    let rc = unsafe { psdj_internal_materialize_build(view, &mut frames, &mut frame_count) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(!frames.is_null());
    assert_ne!(frame_count, 0);

    // SAFETY: the internal export returns a borrowed frame span that remains valid while the
    // owning document is live; tests copy it before freeing the document.
    unsafe { slice::from_raw_parts(frames, frame_count).to_vec() }
}

fn frame_key(frame: &psdj_internal_frame_t) -> &[u8] {
    if frame.key_len == 0 {
        assert!(frame.key_ptr.is_null());
        return &[];
    }
    assert!(!frame.key_ptr.is_null());
    // SAFETY: the frame key span is borrowed from the live document for the duration of the call.
    unsafe { slice::from_raw_parts(frame.key_ptr, frame.key_len) }
}

fn frame_string(frame: &psdj_internal_frame_t) -> &[u8] {
    if frame.string_len == 0 {
        assert!(frame.string_ptr.is_null());
        return &[];
    }
    assert!(!frame.string_ptr.is_null());
    // SAFETY: the frame string span is borrowed from the live document for the duration of the call.
    unsafe { slice::from_raw_parts(frame.string_ptr, frame.string_len) }
}

#[test]
fn psdj_internal_materialize_build_root_frames_match_expected_preorder() {
    let (parser, doc, root) =
        parse_root(br#"{"a":[1,true,null,"x"],"n":18446744073709551615}"#);

    let frames = build_frames(&root);

    assert_eq!(frames.len(), 7);
    assert_eq!(frames[0].kind, PURE_SIMDJSON_VALUE_KIND_OBJECT as u32);
    assert_eq!(frames[0].child_count, 2);
    assert_eq!(frames[0].reserved, 0);

    assert_eq!(frames[1].kind, PURE_SIMDJSON_VALUE_KIND_ARRAY as u32);
    assert_eq!(frame_key(&frames[1]), b"a");
    assert_eq!(frames[1].child_count, 4);
    assert_eq!(frames[1].key_len, 1);

    assert_eq!(frames[2].kind, PURE_SIMDJSON_VALUE_KIND_INT64 as u32);
    assert_eq!(frames[2].int64_value, 1);
    assert_eq!(frames[3].kind, PURE_SIMDJSON_VALUE_KIND_BOOL as u32);
    assert_eq!(frames[3].flags, 1);
    assert_eq!(frames[4].kind, PURE_SIMDJSON_VALUE_KIND_NULL as u32);
    assert_eq!(frames[5].kind, PURE_SIMDJSON_VALUE_KIND_STRING as u32);
    assert_eq!(frames[5].string_len, 1);
    assert_eq!(frame_string(&frames[5]), b"x");

    assert_eq!(frames[6].kind, PURE_SIMDJSON_VALUE_KIND_UINT64 as u32);
    assert_eq!(frame_key(&frames[6]), b"n");
    assert_eq!(frames[6].uint64_value, 18446744073709551615);

    cleanup(parser, doc);
}

#[test]
fn psdj_internal_materialize_build_subtree_frames_match_expected_preorder() {
    // Subtree coverage proves descendant ValueView transport reaches the same internal builder.
    let (parser, doc, root) =
        parse_root(br#"{"a":[1,true,null,"x"],"n":18446744073709551615}"#);
    let array_view = object_get_field_view(&root, b"a");

    let frames = build_frames(&array_view);

    assert_eq!(frames.len(), 5);
    assert_eq!(frames[0].kind, PURE_SIMDJSON_VALUE_KIND_ARRAY as u32);
    assert_eq!(frames[0].child_count, 4);
    assert_eq!(frame_key(&frames[0]), b"");
    assert_eq!(frames[4].kind, PURE_SIMDJSON_VALUE_KIND_STRING as u32);
    assert_eq!(frame_string(&frames[4]), b"x");

    cleanup(parser, doc);
}

#[test]
fn psdj_internal_materialize_build_on_simple_object_returns_exactly_two_frames() {
    let (parser, doc, root) = parse_root(br#"{"ok":1}"#);

    let frames = build_frames(&root);

    assert_eq!(frames.len(), 2);
    assert_eq!(frames[0].kind, PURE_SIMDJSON_VALUE_KIND_OBJECT as u32);
    assert_eq!(frames[0].child_count, 1);
    assert_eq!(frames[1].kind, PURE_SIMDJSON_VALUE_KIND_INT64 as u32);
    assert_eq!(frame_key(&frames[1]), b"ok");
    assert_eq!(frames[1].int64_value, 1);

    cleanup(parser, doc);
}

#[test]
fn psdj_internal_materialize_build_propagates_invalid_handle_after_doc_close() {
    let (parser, doc, root) = parse_root(br#"{"ok":1}"#);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);

    let mut frames = ptr::null();
    let mut frame_count = 0_usize;
    let rc = unsafe { psdj_internal_materialize_build(&root, &mut frames, &mut frame_count) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_HANDLE);
    assert!(frames.is_null());
    assert_eq!(frame_count, 0);

    assert_eq!(unsafe { pure_simdjson_parser_free(parser) }, PURE_SIMDJSON_OK);
}

#[test]
fn oversized_literal_parse_rejected_before_materialize() {
    let parser = parser_new();
    let mut doc: pure_simdjson_doc_t = 0;
    let json = br#"{"ok":1,"big":99999999999999999999999}"#;

    let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(doc, 0);
    assert_eq!(unsafe { pure_simdjson_parser_free(parser) }, PURE_SIMDJSON_OK);
}

#[test]
fn materialize_build_reentrant_guard_is_present() {
    let (parser, doc, root) = parse_root(br#"{"ok":1}"#);

    let rc = unsafe { psdj_internal_test_hold_materialize_guard(&root) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_PARSER_BUSY);
    cleanup(parser, doc);
}
