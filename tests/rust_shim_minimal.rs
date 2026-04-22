use std::{
    collections::HashSet,
    ptr,
    sync::{Arc, Barrier, Mutex},
    thread,
};

use pure_simdjson::{
    pure_simdjson_copy_implementation_name, pure_simdjson_doc_free, pure_simdjson_doc_root,
    pure_simdjson_doc_t, pure_simdjson_element_get_int64, pure_simdjson_element_type,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CPP_EXCEPTION, PURE_SIMDJSON_ERR_INVALID_ARGUMENT,
        PURE_SIMDJSON_ERR_INVALID_HANDLE, PURE_SIMDJSON_ERR_INVALID_JSON,
        PURE_SIMDJSON_ERR_PARSER_BUSY, PURE_SIMDJSON_ERR_WRONG_TYPE, PURE_SIMDJSON_OK,
    },
    pure_simdjson_get_abi_version, pure_simdjson_get_implementation_name_len,
    pure_simdjson_handle_t, pure_simdjson_native_alloc_stats_reset,
    pure_simdjson_native_alloc_stats_snapshot, pure_simdjson_native_alloc_stats_t,
    pure_simdjson_parser_copy_last_error, pure_simdjson_parser_free,
    pure_simdjson_parser_get_last_error_len, pure_simdjson_parser_get_last_error_offset,
    pure_simdjson_parser_new, pure_simdjson_parser_parse, pure_simdjson_parser_t,
    pure_simdjson_test_force_cpp_exception_for_tests,
    pure_simdjson_value_kind_t::{
        PURE_SIMDJSON_VALUE_KIND_ARRAY, PURE_SIMDJSON_VALUE_KIND_BOOL,
        PURE_SIMDJSON_VALUE_KIND_FLOAT64, PURE_SIMDJSON_VALUE_KIND_INT64,
        PURE_SIMDJSON_VALUE_KIND_NULL, PURE_SIMDJSON_VALUE_KIND_OBJECT,
        PURE_SIMDJSON_VALUE_KIND_STRING, PURE_SIMDJSON_VALUE_KIND_UINT64,
    },
    pure_simdjson_value_view_t, PURE_SIMDJSON_ABI_VERSION,
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

fn element_type_of(root: &pure_simdjson_value_view_t) -> u32 {
    let mut kind = 0_u32;
    let rc = unsafe { pure_simdjson_element_type(root, &mut kind) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    kind
}

fn element_get_int64_of(root: &pure_simdjson_value_view_t) -> i64 {
    let mut value = 0_i64;
    let rc = unsafe { pure_simdjson_element_get_int64(root, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    value
}

fn implementation_name() -> Vec<u8> {
    let mut len = 0_usize;
    let len_rc = unsafe { pure_simdjson_get_implementation_name_len(&mut len) };
    assert_eq!(len_rc, PURE_SIMDJSON_OK);

    let mut name = vec![0_u8; len];
    let mut written = 0_usize;
    let copy_rc = unsafe {
        pure_simdjson_copy_implementation_name(name.as_mut_ptr(), name.len(), &mut written)
    };
    assert_eq!(copy_rc, PURE_SIMDJSON_OK);
    name.truncate(written);
    name
}

fn parser_last_error(parser: pure_simdjson_handle_t) -> String {
    let mut len = 0_usize;
    let len_rc = unsafe { pure_simdjson_parser_get_last_error_len(parser, &mut len) };
    assert_eq!(len_rc, PURE_SIMDJSON_OK);

    let mut bytes = vec![0_u8; len];
    let mut written = 0_usize;
    let copy_rc = unsafe {
        pure_simdjson_parser_copy_last_error(parser, bytes.as_mut_ptr(), bytes.len(), &mut written)
    };
    assert_eq!(copy_rc, PURE_SIMDJSON_OK);
    bytes.truncate(written);
    String::from_utf8(bytes).expect("last error should be valid UTF-8")
}

fn parser_last_error_offset(parser: pure_simdjson_handle_t) -> u64 {
    let mut offset = 0_u64;
    let offset_rc = unsafe { pure_simdjson_parser_get_last_error_offset(parser, &mut offset) };
    assert_eq!(offset_rc, PURE_SIMDJSON_OK);
    offset
}

fn native_alloc_stats_reset() {
    let rc = unsafe { pure_simdjson_native_alloc_stats_reset() };
    assert_eq!(rc, PURE_SIMDJSON_OK);
}

fn native_alloc_stats_snapshot() -> pure_simdjson_native_alloc_stats_t {
    let mut stats = pure_simdjson_native_alloc_stats_t::default();
    let rc = unsafe { pure_simdjson_native_alloc_stats_snapshot(&mut stats) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    stats
}

fn pack_handle(slot: u32, generation: u32) -> pure_simdjson_handle_t {
    u64::from(slot) | (u64::from(generation) << 32)
}

#[test]
fn get_abi_version_returns_phase1_constant() {
    let mut abi_version = 0_u32;

    let rc = unsafe { pure_simdjson_get_abi_version(&mut abi_version) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(abi_version, PURE_SIMDJSON_ABI_VERSION);
}

#[test]
fn implementation_name_round_trip_uses_real_bridge_name() {
    let name = implementation_name();

    assert!(!name.is_empty());
    assert_ne!(name, b"contract-only");
}

#[test]
fn native_alloc_stats_snapshot_null_out_is_invalid_argument() {
    let rc = unsafe { pure_simdjson_native_alloc_stats_snapshot(ptr::null_mut()) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_ARGUMENT);
}

#[test]
fn native_alloc_stats_round_trip_before_and_after_parse_activity() {
    let parser = parser_new();
    native_alloc_stats_reset();

    let before = native_alloc_stats_snapshot();
    assert_eq!(before.live_bytes, 0);
    assert_eq!(before.total_alloc_bytes, 0);
    assert_eq!(before.alloc_count, 0);
    assert_eq!(before.free_count, 0);

    let doc = parser_parse_literal(parser, br#"{"value":[1,2,3],"label":"hello"}"#);
    let during = native_alloc_stats_snapshot();
    assert!(during.live_bytes > 0);
    assert!(during.total_alloc_bytes >= during.live_bytes);
    assert!(during.alloc_count > 0);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let after = native_alloc_stats_snapshot();
    assert_eq!(after.live_bytes, 0);
    assert!(after.free_count > 0);
}

#[test]
fn native_alloc_stats_reset_excludes_preexisting_live_allocations() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[1,2,3]");
    let warm = native_alloc_stats_snapshot();
    assert!(warm.live_bytes > 0);

    native_alloc_stats_reset();
    let reset = native_alloc_stats_snapshot();
    assert_eq!(reset.live_bytes, 0);
    assert_eq!(reset.total_alloc_bytes, 0);
    assert_eq!(reset.alloc_count, 0);
    assert_eq!(reset.free_count, 0);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let after = native_alloc_stats_snapshot();
    assert_eq!(after.live_bytes, 0);
    assert_eq!(after.total_alloc_bytes, 0);
    assert_eq!(after.alloc_count, 0);
    assert_eq!(after.free_count, 0);
}

#[test]
fn element_get_int64_reads_literal_42_root() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);

    assert_eq!(
        element_type_of(&root),
        PURE_SIMDJSON_VALUE_KIND_INT64 as u32
    );
    assert_eq!(element_get_int64_of(&root), 42);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn element_type_maps_phase2_root_literals() {
    let cases = [
        (b"null".as_slice(), PURE_SIMDJSON_VALUE_KIND_NULL as u32),
        (b"true".as_slice(), PURE_SIMDJSON_VALUE_KIND_BOOL as u32),
        (b"-42".as_slice(), PURE_SIMDJSON_VALUE_KIND_INT64 as u32),
        (b"42".as_slice(), PURE_SIMDJSON_VALUE_KIND_INT64 as u32),
        (
            b"18446744073709551615".as_slice(),
            PURE_SIMDJSON_VALUE_KIND_UINT64 as u32,
        ),
        (b"1.5".as_slice(), PURE_SIMDJSON_VALUE_KIND_FLOAT64 as u32),
        (br#""x""#.as_slice(), PURE_SIMDJSON_VALUE_KIND_STRING as u32),
        (b"[1]".as_slice(), PURE_SIMDJSON_VALUE_KIND_ARRAY as u32),
        (
            br#"{"k":1}"#.as_slice(),
            PURE_SIMDJSON_VALUE_KIND_OBJECT as u32,
        ),
    ];

    for (json, expected_kind) in cases {
        let parser = parser_new();
        let doc = parser_parse_literal(parser, json);
        let root = doc_root(doc);

        assert_eq!(
            element_type_of(&root),
            expected_kind,
            "root literal {:?}",
            json
        );

        assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
        assert_eq!(
            unsafe { pure_simdjson_parser_free(parser) },
            PURE_SIMDJSON_OK
        );
    }
}

#[test]
fn invalid_json_reports_last_error_and_unknown_offset() {
    let parser = parser_new();
    let mut doc = 0_u64;

    let rc = unsafe { pure_simdjson_parser_parse(parser, b"{".as_ptr(), 1, &mut doc) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(doc, 0);
    assert!(!parser_last_error(parser).is_empty());

    let mut offset = 0_u64;
    let offset_rc = unsafe { pure_simdjson_parser_get_last_error_offset(parser, &mut offset) };
    assert_eq!(offset_rc, PURE_SIMDJSON_OK);
    assert_eq!(offset, u64::MAX);

    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn zero_length_input_returns_invalid_json() {
    let parser = parser_new();
    let mut doc = 0_u64;

    let rc = unsafe { pure_simdjson_parser_parse(parser, ptr::null(), 0, &mut doc) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(doc, 0);
    assert!(!parser_last_error(parser).is_empty());
    assert_eq!(parser_last_error_offset(parser), u64::MAX);

    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn successful_parse_after_invalid_json_clears_prior_error() {
    let parser = parser_new();
    let mut bad_doc = 0_u64;

    let bad_rc = unsafe { pure_simdjson_parser_parse(parser, b"{".as_ptr(), 1, &mut bad_doc) };
    assert_eq!(bad_rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(bad_doc, 0);
    assert!(!parser_last_error(parser).is_empty());

    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);

    assert_eq!(element_get_int64_of(&root), 42);
    assert_eq!(parser_last_error(parser), "");
    assert_eq!(parser_last_error_offset(parser), u64::MAX);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn parser_busy_rejects_second_parse_until_doc_is_freed() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let mut next_doc = 0_u64;

    let busy_rc = unsafe { pure_simdjson_parser_parse(parser, b"43".as_ptr(), 2, &mut next_doc) };
    assert_eq!(busy_rc, PURE_SIMDJSON_ERR_PARSER_BUSY);
    assert_eq!(next_doc, 0);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);

    let reparsed_doc = parser_parse_literal(parser, b"43");
    assert_eq!(
        unsafe { pure_simdjson_doc_free(reparsed_doc) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn parser_free_while_doc_live_returns_parser_busy() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");

    let free_rc = unsafe { pure_simdjson_parser_free(parser) };
    assert_eq!(free_rc, PURE_SIMDJSON_ERR_PARSER_BUSY);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn stale_handle_after_doc_free_returns_invalid_handle() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);

    let mut kind = 0_u32;
    let kind_rc = unsafe { pure_simdjson_element_type(&root, &mut kind) };
    assert_eq!(kind_rc, PURE_SIMDJSON_ERR_INVALID_HANDLE);

    let mut next_root = pure_simdjson_value_view_t::default();
    let root_rc = unsafe { pure_simdjson_doc_root(doc, &mut next_root) };
    assert_eq!(root_rc, PURE_SIMDJSON_ERR_INVALID_HANDLE);

    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn double_free_returns_invalid_handle() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_doc_free(doc) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
}

#[test]
fn psimdjson_test_force_cpp_exception_returns_err_cpp_exception() {
    assert_eq!(
        pure_simdjson_test_force_cpp_exception_for_tests(),
        PURE_SIMDJSON_ERR_CPP_EXCEPTION
    );
}

#[test]
fn parser_get_last_error_helpers_validate_handles() {
    let mut len = 0_usize;
    let len_rc = unsafe { pure_simdjson_parser_get_last_error_len(0, &mut len) };
    assert_eq!(len_rc, PURE_SIMDJSON_ERR_INVALID_HANDLE);

    let mut written = 0_usize;
    let copy_rc =
        unsafe { pure_simdjson_parser_copy_last_error(0, ptr::null_mut(), 0, &mut written) };
    assert_eq!(copy_rc, PURE_SIMDJSON_ERR_INVALID_HANDLE);
}

#[test]
fn null_pointer_matrix_returns_invalid_argument() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);

    assert_eq!(
        unsafe { pure_simdjson_parser_new(ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    let mut parsed_doc = 0_u64;
    assert_eq!(
        unsafe { pure_simdjson_parser_parse(parser, ptr::null(), 1, &mut parsed_doc) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe { pure_simdjson_parser_parse(parser, b"42".as_ptr(), 2, ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    assert_eq!(
        unsafe { pure_simdjson_doc_root(doc, ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    let mut kind = 0_u32;
    assert_eq!(
        unsafe { pure_simdjson_element_type(ptr::null(), &mut kind) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe { pure_simdjson_element_type(&root, ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    let mut value = 0_i64;
    assert_eq!(
        unsafe { pure_simdjson_element_get_int64(ptr::null(), &mut value) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe { pure_simdjson_element_get_int64(&root, ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn distinct_parsers_work_under_contention() {
    let thread_count = 8;
    let iterations = 100;
    let start = Arc::new(Barrier::new(thread_count));
    let seen_handles = Arc::new(Mutex::new(HashSet::new()));
    let mut workers = Vec::with_capacity(thread_count);

    for thread_index in 0..thread_count {
        let start = Arc::clone(&start);
        let seen_handles = Arc::clone(&seen_handles);
        workers.push(thread::spawn(move || {
            start.wait();
            for iteration in 0..iterations {
                let expected = (thread_index * 1000 + iteration) as i64;
                let json = expected.to_string();
                let parser = parser_new();
                {
                    let mut seen_handles = seen_handles.lock().expect("mutex should not poison");
                    assert!(
                        seen_handles.insert(parser),
                        "duplicate parser handle observed: {parser:#x}"
                    );
                }

                let doc = parser_parse_literal(parser, json.as_bytes());
                {
                    let mut seen_handles = seen_handles.lock().expect("mutex should not poison");
                    assert!(
                        seen_handles.insert(doc),
                        "duplicate doc handle observed: {doc:#x}"
                    );
                }

                let root = doc_root(doc);
                assert_eq!(element_get_int64_of(&root), expected);
                assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
                assert_eq!(
                    unsafe { pure_simdjson_parser_free(parser) },
                    PURE_SIMDJSON_OK
                );
            }
        }));
    }

    for worker in workers {
        worker.join().expect("worker thread should complete");
    }
}

#[test]
fn parser_and_doc_handles_do_not_alias_across_types() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");

    assert_ne!(parser, doc);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(doc) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
    assert_eq!(
        unsafe { pure_simdjson_doc_free(parser) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn element_get_int64_reports_wrong_type_for_bool() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"true");
    let root = doc_root(doc);
    let mut value = 0_i64;

    let rc = unsafe { pure_simdjson_element_get_int64(&root, &mut value) };
    assert_eq!(rc, PURE_SIMDJSON_ERR_WRONG_TYPE);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn oversized_integer_is_rejected_at_parse_time() {
    let parser = parser_new();
    let mut doc = 0_u64;

    let rc = unsafe {
        pure_simdjson_parser_parse(parser, b"99999999999999999999".as_ptr(), 20, &mut doc)
    };
    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(doc, 0);
    assert!(!parser_last_error(parser).is_empty());
    assert_eq!(parser_last_error_offset(parser), u64::MAX);

    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn tampered_root_view_tag_and_reserved_bits_return_invalid_handle() {
    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"42");
    let root = doc_root(doc);

    let mut kind = 0_u32;
    let mut wrong_tag = root;
    wrong_tag.state1 = 0;
    assert_eq!(
        unsafe { pure_simdjson_element_type(&wrong_tag, &mut kind) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );

    let mut reserved_bits = root;
    reserved_bits.reserved = 1;
    assert_eq!(
        unsafe { pure_simdjson_element_type(&reserved_bits, &mut kind) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn forged_out_of_range_parser_slot_returns_invalid_handle() {
    let forged = pack_handle(u32::MAX, 1);

    assert_eq!(
        unsafe { pure_simdjson_parser_free(forged) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );

    let mut len = 0_usize;
    assert_eq!(
        unsafe { pure_simdjson_parser_get_last_error_len(forged, &mut len) },
        PURE_SIMDJSON_ERR_INVALID_HANDLE
    );
}
