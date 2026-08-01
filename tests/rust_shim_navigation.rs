use std::{ptr, slice, thread};

use pure_simdjson::{
    pure_simdjson_array_at, pure_simdjson_array_len, pure_simdjson_doc_free,
    pure_simdjson_doc_root, pure_simdjson_doc_t, pure_simdjson_element_at_path,
    pure_simdjson_element_at_path_wildcard, pure_simdjson_element_at_pointer,
    pure_simdjson_element_get_int64,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE, PURE_SIMDJSON_ERR_INVALID_ARGUMENT,
        PURE_SIMDJSON_ERR_INVALID_HANDLE, PURE_SIMDJSON_ERR_INVALID_PATH,
        PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_object_size, pure_simdjson_parser_free, pure_simdjson_parser_new,
    pure_simdjson_parser_parse, pure_simdjson_parser_t, pure_simdjson_value_view_t,
    pure_simdjson_value_views_free,
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

fn wildcard_call(
    view: &pure_simdjson_value_view_t,
    path: &[u8],
) -> (
    pure_simdjson::pure_simdjson_error_code_t,
    *mut pure_simdjson_value_view_t,
    usize,
) {
    let mut views = ptr::null_mut();
    let mut count = 0_usize;
    let rc = unsafe {
        pure_simdjson_element_at_path_wildcard(
            view,
            path.as_ptr(),
            path.len(),
            &mut views,
            &mut count,
        )
    };
    (rc, views, count)
}

fn wildcard_int64_values(
    view: &pure_simdjson_value_view_t,
    path: &[u8],
) -> Result<Vec<i64>, pure_simdjson::pure_simdjson_error_code_t> {
    let (rc, views_ptr, count) = wildcard_call(view, path);
    if rc != PURE_SIMDJSON_OK {
        return Err(rc);
    }

    let views: &[pure_simdjson_value_view_t] = if count == 0 {
        assert!(
            views_ptr.is_null(),
            "empty wildcard result must use null/zero"
        );
        &[]
    } else {
        assert!(
            !views_ptr.is_null(),
            "non-empty wildcard result needs storage"
        );
        // SAFETY: a successful wildcard call returns exactly `count` initialized views.
        unsafe { slice::from_raw_parts(views_ptr, count) }
    };
    let values = views.iter().map(read_int64).collect();
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views_ptr, count) },
        PURE_SIMDJSON_OK
    );
    Ok(values)
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

#[test]
fn wildcard_paths_follow_spike_005_truth_table() {
    let cases: &[(&str, &[u8], &[u8], &[i64])] = &[
        (
            "ordered array-of-object matches",
            br#"{"items":[{"id":3},{"id":1},{"id":2}]}"#,
            b".items[*].id",
            &[3, 1, 2],
        ),
        ("empty container", br#"{"items":[]}"#, b".items[*].id", &[]),
        (
            "partial branches",
            br#"{"items":[{"id":1},{"other":2},{"id":3}]}"#,
            b".items[*].id",
            &[1, 3],
        ),
        (
            "heterogeneous branches",
            br#"{"items":[{"id":1},5,null,{"id":3}]}"#,
            b".items[*].id",
            &[1, 3],
        ),
        (
            "missing prefix",
            br#"{"items":[{"id":1}]}"#,
            b".missing[*].id",
            &[],
        ),
        (
            "out-of-range branches",
            br#"{"items":[[1],[],[3]]}"#,
            b".items[*][4]",
            &[],
        ),
        (
            "non-container branches",
            br#"{"items":[1,{"id":2},null,{"id":3}]}"#,
            b".items[*].id",
            &[2, 3],
        ),
        (
            "multiple wildcard levels",
            br#"{"groups":[{"items":[{"id":1},{"id":2}]},{"items":[{"id":3}]}]}"#,
            b".groups[*].items[*].id",
            &[1, 2, 3],
        ),
        (
            "quoted bracket prefix remains literal",
            br#"{"obj":{"'foo'":[1,2],"foo":[3]}}"#,
            b".obj['foo'][*]",
            &[1, 2],
        ),
        (
            "leading quoted bracket prefix remains literal",
            br#"{"'obj'":{"first":1,"second":2}}"#,
            b"['obj'].*",
            &[1, 2],
        ),
        (
            "quoted bracket suffix remains literal",
            br#"{"items":[{"'foo'":4,"foo":5}]}"#,
            b".items[*]['foo']",
            &[4],
        ),
        (
            "root-dollar dot wildcard",
            br#"{"items":[1,2]}"#,
            b"$.items.*",
            &[1, 2],
        ),
        (
            "wildcard followed by index",
            br#"{"items":[[1],[2]]}"#,
            b".items[*][0]",
            &[1, 2],
        ),
        (
            "indexed prefix before wildcard",
            br#"{"arr":[[1,2]]}"#,
            b".arr[0][*]",
            &[1, 2],
        ),
        (
            "quoted dotted prefix remains AtPath literal",
            br#"{"obj":{"'foo.bar'":[1,2],"foo.bar":[3]}}"#,
            b".obj['foo.bar'][*]",
            &[1, 2],
        ),
        (
            "literal index between wildcards",
            br#"{"groups":[[[1,2]],[[3,4]]]}"#,
            b".groups[*][0][*]",
            &[1, 2, 3, 4],
        ),
    ];

    for &(name, json, path, expected) in cases {
        let parser = parser_new();
        let doc = parser_parse_literal(parser, json);
        let root = doc_root(doc);

        let values = wildcard_int64_values(&root, path)
            .unwrap_or_else(|rc| panic!("{name}: wildcard call failed with {rc:?}"));

        assert_eq!(values, expected, "{name}");
        cleanup(parser, doc);
    }
}

#[test]
fn malformed_wildcard_paths_return_invalid_path() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"items":[{"id":1}],"rows":[[1]]}"#);
    let root = doc_root(doc);

    for path in [
        b"a.b".as_slice(), b"*", b".a[0", b".items[*].", b".items[*][", b"", b"['*'][*]", b".items.thing*", b".items[*]junk[0]", b".a[*]b[0]", b".rows[01][*]", b".a.01[*]", b"\xff",
    ] {
        let (rc, views, count) = wildcard_call(&root, path);
        assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_PATH, "path {path:?}");
        assert!(views.is_null(), "failed call must not issue an allocation");
        assert_eq!(count, 0, "failed call must not issue elements");
    }

    cleanup(parser, doc);
}

#[test]
fn wildcard_failures_clear_writable_output_sentinels() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"items":[1]}"#);
    let root = doc_root(doc);
    let mut views = 1_usize as *mut pure_simdjson_value_view_t;
    let mut count = 99_usize;
    let path = b".items[*].";
    let rc = unsafe {
        pure_simdjson_element_at_path_wildcard(
            &root, path.as_ptr(), path.len(), &mut views, &mut count,
        )
    };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_PATH);
    assert!(views.is_null());
    assert_eq!(count, 0);
    cleanup(parser, doc);
}

#[test]
fn value_views_free_enforces_exact_pointer_count_pairs() {
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(ptr::null_mut(), 0) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(ptr::null_mut(), 1) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"items":[{"id":1},{"id":2}]}"#);
    let root = doc_root(doc);
    let (rc, views, count) = wildcard_call(&root, b".items[*].id");
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert!(!views.is_null());
    assert_eq!(count, 2);

    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views, count + 1) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "mismatched count must preserve the registered allocation"
    );
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views, 0) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT,
        "nonnull/zero must not reconstruct the allocation"
    );
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views, count) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views, count) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE,
        "double-free must be rejected before pointer dereference"
    );

    cleanup(parser, doc);
}

#[test]
fn copied_wildcard_view_outlives_array_but_not_document() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, br#"{"items":[{"id":41},{"id":42}]}"#);
    let root = doc_root(doc);
    let (rc, views, count) = wildcard_call(&root, b".items[*].id");
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(count, 2);
    // SAFETY: the successful call returned two initialized views.
    let copied = unsafe { *views };

    assert_eq!(
        unsafe { pure_simdjson_value_views_free(views, count) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(read_int64(&copied), 41);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    let mut value = 0_i64;
    assert_eq!(
        unsafe { pure_simdjson_element_get_int64(&copied, &mut value) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn same_document_wildcard_calls_are_serialized_by_registry() {
    let parser = parser_new();
    let doc = parser_parse_literal(
        parser,
        br#"{"items":[{"id":1,"score":10},{"id":2,"score":20}]}"#,
    );
    let root = doc_root(doc);

    let (ids, scores) = thread::scope(|scope| {
        let ids_root = root;
        let scores_root = root;
        let ids = scope.spawn(move || wildcard_int64_values(&ids_root, b".items[*].id"));
        let scores = scope.spawn(move || wildcard_int64_values(&scores_root, b".items[*].score"));
        (
            ids.join().expect("id wildcard thread panicked"),
            scores.join().expect("score wildcard thread panicked"),
        )
    });

    assert_eq!(ids.expect("id wildcard call failed"), vec![1, 2]);
    assert_eq!(scores.expect("score wildcard call failed"), vec![10, 20]);
    cleanup(parser, doc);
}
