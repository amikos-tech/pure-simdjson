use std::{ptr, slice};

use pure_simdjson::{
    pure_simdjson_array_iter_new, pure_simdjson_array_iter_next, pure_simdjson_array_iter_t,
    pure_simdjson_bytes_free, pure_simdjson_doc_free, pure_simdjson_doc_root, pure_simdjson_doc_t,
    pure_simdjson_element_get_bigint, pure_simdjson_element_get_float64,
    pure_simdjson_element_get_int64, pure_simdjson_element_get_uint64, pure_simdjson_element_type,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT, PURE_SIMDJSON_ERR_INVALID_HANDLE,
        PURE_SIMDJSON_ERR_INVALID_JSON, PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_object_get_field, pure_simdjson_parser_free, pure_simdjson_parser_new,
    pure_simdjson_parser_parse, pure_simdjson_parser_t, pure_simdjson_value_view_t,
};

const KIND_BIGINT: u32 = 9;

#[derive(Clone, Copy)]
enum BigIntLocation {
    Root,
    ArrayFirst,
    ObjectField,
}

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

fn object_get_field(object: &pure_simdjson_value_view_t, key: &[u8]) -> pure_simdjson_value_view_t {
    let mut value = pure_simdjson_value_view_t::default();
    let rc = unsafe { pure_simdjson_object_get_field(object, key.as_ptr(), key.len(), &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    value
}

fn array_first(array: &pure_simdjson_value_view_t) -> pure_simdjson_value_view_t {
    let mut iter = pure_simdjson_array_iter_t::default();
    assert_eq!(
        unsafe { pure_simdjson_array_iter_new(array, &mut iter) },
        PURE_SIMDJSON_OK
    );
    let mut item = pure_simdjson_value_view_t::default();
    let mut done = 1_u8;
    assert_eq!(
        unsafe { pure_simdjson_array_iter_next(&mut iter, &mut item, &mut done) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(done, 0);
    item
}

fn read_bigint(view: &pure_simdjson_value_view_t) -> String {
    let mut out_ptr: *mut u8 = ptr::null_mut();
    let mut out_len = 0_usize;
    let rc = unsafe { pure_simdjson_element_get_bigint(view, &mut out_ptr, &mut out_len) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(!out_ptr.is_null());
    let value = unsafe { slice::from_raw_parts(out_ptr, out_len) };
    let value = String::from_utf8(value.to_vec()).expect("BigInt text must be UTF-8");
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(out_ptr, out_len) },
        PURE_SIMDJSON_OK
    );
    value
}

fn assert_bigint_case(json: &[u8], expected: &[u8], location: BigIntLocation) {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, json);
    let root = doc_root(doc);
    let bigint = match location {
        BigIntLocation::Root => root,
        BigIntLocation::ArrayFirst => array_first(&root),
        BigIntLocation::ObjectField => object_get_field(&root, b"n"),
    };

    assert_eq!(bigint.kind_hint, KIND_BIGINT, "wrong kind for {json:?}");
    assert_eq!(
        read_bigint(&bigint).as_bytes(),
        expected,
        "wrong exact text for {json:?}"
    );

    cleanup(parser, doc);
}

fn cleanup(parser: pure_simdjson_parser_t, doc: pure_simdjson_doc_t) {
    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn malformed_bigint_suffixes_are_rejected_without_truncation() {
    let cases: &[(&str, &[u8])] = &[
        ("positive root x", b"18446744073709551616x"),
        ("positive root underscore", b"18446744073709551616_"),
        ("positive root plus", b"18446744073709551616+"),
        ("positive root slash", b"18446744073709551616/"),
        ("positive root NUL", b"18446744073709551616\0"),
        ("negative root x", b"-9223372036854775809x"),
        ("negative root underscore", b"-9223372036854775809_"),
        ("negative root plus", b"-9223372036854775809+"),
        ("negative root slash", b"-9223372036854775809/"),
        ("negative root NUL", b"-9223372036854775809\0"),
        ("array value x", b"[123456789012345678901x]"),
        ("array value underscore", b"[123456789012345678901_]"),
        ("array value plus", b"[123456789012345678901+]"),
        ("array value slash", b"[123456789012345678901/]"),
        ("array value NUL", b"[123456789012345678901\0]"),
        ("object value x", br#"{"n":-9223372036854775809x}"#),
        ("object value underscore", br#"{"n":-9223372036854775809_}"#),
        ("object value plus", br#"{"n":-9223372036854775809+}"#),
        ("object value slash", br#"{"n":-9223372036854775809/}"#),
        ("object value NUL", b"{\"n\":-9223372036854775809\0}"),
    ];

    for &(name, json) in cases {
        let parser = parser_new();
        let sentinel = u64::MAX;
        let mut doc = sentinel;
        let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };
        assert_eq!(
            rc, PURE_SIMDJSON_ERR_INVALID_JSON,
            "{name} unexpectedly parsed: {json:?}"
        );
        assert_eq!(doc, sentinel, "{name} overwrote the document output");
        assert_eq!(
            unsafe { pure_simdjson_parser_free(parser) },
            PURE_SIMDJSON_OK
        );
    }
}

#[test]
fn valid_bigint_delimiters_preserve_exact_text() {
    let positive = b"18446744073709551616";
    let negative = b"-9223372036854775809";
    let cases: &[(&[u8], &[u8], BigIntLocation)] = &[
        (positive, positive, BigIntLocation::Root),
        (negative, negative, BigIntLocation::Root),
        (
            b"[18446744073709551616]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
        (
            br#"{"n":18446744073709551616}"#,
            positive,
            BigIntLocation::ObjectField,
        ),
        (
            br#"{"n":-9223372036854775809}"#,
            negative,
            BigIntLocation::ObjectField,
        ),
        (
            b"[18446744073709551616,0]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809,0]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[18446744073709551616 ]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809 ]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[18446744073709551616\t]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809\t]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[18446744073709551616\r]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809\r]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[18446744073709551616\n]",
            positive,
            BigIntLocation::ArrayFirst,
        ),
        (
            b"[-9223372036854775809\n]",
            negative,
            BigIntLocation::ArrayFirst,
        ),
    ];

    for &(json, expected, location) in cases {
        assert_bigint_case(json, expected, location);
    }
}

#[test]
fn bigint_accessor_returns_exact_signed_and_unsigned_text() {
    for literal in [
        b"18446744073709551616".as_slice(),
        b"-9223372036854775809".as_slice(),
        b"1234567890123456789012345678901234567890".as_slice(),
    ] {
        let parser = parser_new();
        let doc = parser_parse_literal(parser, literal);
        let root = doc_root(doc);

        assert_eq!(root.kind_hint, KIND_BIGINT);
        assert_eq!(read_bigint(&root).as_bytes(), literal);

        cleanup(parser, doc);
    }
}

#[test]
fn bigint_copy_survives_document_free_and_rejects_double_free() {
    let parser = parser_new();
    let literal = b"-123456789012345678901234567890";
    let doc = parser_parse_literal(parser, literal);
    let root = doc_root(doc);

    let mut out_ptr: *mut u8 = ptr::null_mut();
    let mut out_len = 0_usize;
    let rc = unsafe { pure_simdjson_element_get_bigint(&root, &mut out_ptr, &mut out_len) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(!out_ptr.is_null());

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(unsafe { slice::from_raw_parts(out_ptr, out_len) }, literal);
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(out_ptr, out_len) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(
        unsafe { pure_simdjson_bytes_free(out_ptr, out_len) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn bigint_accessor_is_strict_for_every_other_json_kind() {
    for literal in [
        b"-1".as_slice(),
        b"18446744073709551615".as_slice(),
        b"1.5".as_slice(),
        br#""text""#.as_slice(),
        b"true".as_slice(),
        b"null".as_slice(),
        b"[]".as_slice(),
        b"{}".as_slice(),
    ] {
        let parser = parser_new();
        let doc = parser_parse_literal(parser, literal);
        let root = doc_root(doc);
        let sentinel = ptr::NonNull::<u8>::dangling().as_ptr();
        let mut out_ptr = sentinel;
        let mut out_len = usize::MAX;

        let rc = unsafe { pure_simdjson_element_get_bigint(&root, &mut out_ptr, &mut out_len) };
        assert_eq!(
            rc, PURE_SIMDJSON_ERR_WRONG_TYPE,
            "unexpected status for {:?}",
            literal
        );
        assert_eq!(out_ptr, sentinel, "error must not overwrite out_ptr");
        assert_eq!(out_len, usize::MAX, "error must not overwrite out_len");

        cleanup(parser, doc);
    }
}

#[test]
fn bigint_accessor_rejects_null_outputs_without_partial_writes() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"18446744073709551616");
    let root = doc_root(doc);

    let mut out_len = 41_usize;
    let rc = unsafe { pure_simdjson_element_get_bigint(&root, ptr::null_mut(), &mut out_len) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_ARGUMENT);
    assert_eq!(out_len, 41);

    let sentinel = ptr::NonNull::<u8>::dangling().as_ptr();
    let mut out_ptr = sentinel;
    let rc = unsafe { pure_simdjson_element_get_bigint(&root, &mut out_ptr, ptr::null_mut()) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_ARGUMENT);
    assert_eq!(out_ptr, sentinel);

    cleanup(parser, doc);
}

#[test]
fn numeric_accessors_reject_bigint_as_wrong_type() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"18446744073709551616");
    let root = doc_root(doc);

    let mut int_value = 17_i64;
    assert_eq!(
        unsafe { pure_simdjson_element_get_int64(&root, &mut int_value) },
        PURE_SIMDJSON_ERR_WRONG_TYPE
    );
    assert_eq!(int_value, 17);

    let mut uint_value = 23_u64;
    assert_eq!(
        unsafe { pure_simdjson_element_get_uint64(&root, &mut uint_value) },
        PURE_SIMDJSON_ERR_WRONG_TYPE
    );
    assert_eq!(uint_value, 23);

    let mut float_value = 1.25_f64;
    assert_eq!(
        unsafe { pure_simdjson_element_get_float64(&root, &mut float_value) },
        PURE_SIMDJSON_ERR_WRONG_TYPE
    );
    assert_eq!(float_value, 1.25);

    cleanup(parser, doc);
}

#[test]
fn root_lookup_and_iterator_views_preserve_bigint_kind_hint() {
    let parser = parser_new();
    let doc = parser_parse_literal(
        parser,
        br#"{"lookup":18446744073709551616,"items":[-9223372036854775809]}"#,
    );
    let root = doc_root(doc);

    let lookup = object_get_field(&root, b"lookup");
    assert_eq!(lookup.kind_hint, KIND_BIGINT);
    let mut kind = 0_u32;
    assert_eq!(
        unsafe { pure_simdjson_element_type(&lookup, &mut kind) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(kind, KIND_BIGINT);
    assert_eq!(read_bigint(&lookup), "18446744073709551616");

    let items = object_get_field(&root, b"items");
    let mut iter = pure_simdjson_array_iter_t::default();
    assert_eq!(
        unsafe { pure_simdjson_array_iter_new(&items, &mut iter) },
        PURE_SIMDJSON_OK
    );
    let mut item = pure_simdjson_value_view_t::default();
    let mut done = 1_u8;
    assert_eq!(
        unsafe { pure_simdjson_array_iter_next(&mut iter, &mut item, &mut done) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(done, 0);
    assert_eq!(item.kind_hint, KIND_BIGINT);
    assert_eq!(read_bigint(&item), "-9223372036854775809");

    cleanup(parser, doc);
}
