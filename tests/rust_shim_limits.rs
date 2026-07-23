use std::ptr;

use pure_simdjson::{
    pure_simdjson_doc_free, pure_simdjson_doc_t,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CAPACITY_LIMIT, PURE_SIMDJSON_ERR_DEPTH_LIMIT,
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT, PURE_SIMDJSON_ERR_INVALID_JSON, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_copy_last_error, pure_simdjson_parser_free,
    pure_simdjson_parser_get_last_error_len, pure_simdjson_parser_get_last_error_offset,
    pure_simdjson_parser_new, pure_simdjson_parser_new_configured, pure_simdjson_parser_parse,
    pure_simdjson_parser_t,
};

fn parser_new() -> pure_simdjson_parser_t {
    let mut parser = 0;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    parser
}

fn parser_new_configured(max_capacity: u64, max_depth: u32) -> pure_simdjson_parser_t {
    let mut parser = 0;
    let rc = unsafe { pure_simdjson_parser_new_configured(max_capacity, max_depth, &mut parser) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    parser
}

fn parser_parse(
    parser: pure_simdjson_parser_t,
    json: &[u8],
) -> (
    pure_simdjson::pure_simdjson_error_code_t,
    pure_simdjson_doc_t,
) {
    let mut doc = 0;
    let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };
    (rc, doc)
}

fn parser_last_error(parser: pure_simdjson_parser_t) -> Vec<u8> {
    let mut len = 0;
    assert_eq!(
        unsafe { pure_simdjson_parser_get_last_error_len(parser, &mut len) },
        PURE_SIMDJSON_OK
    );

    let mut message = vec![0; len];
    let mut written = usize::MAX;
    assert_eq!(
        unsafe {
            pure_simdjson_parser_copy_last_error(
                parser,
                message.as_mut_ptr(),
                message.len(),
                &mut written,
            )
        },
        PURE_SIMDJSON_OK
    );
    message.truncate(written);
    message
}

fn parser_last_error_offset(parser: pure_simdjson_parser_t) -> u64 {
    let mut offset = 0;
    assert_eq!(
        unsafe { pure_simdjson_parser_get_last_error_offset(parser, &mut offset) },
        PURE_SIMDJSON_OK
    );
    offset
}

fn json_string_with_total_len(total_len: usize) -> Vec<u8> {
    assert!(total_len >= 2);
    let mut json = vec![b'x'; total_len];
    json[0] = b'"';
    json[total_len - 1] = b'"';
    json
}

fn nested_array(depth: usize) -> Vec<u8> {
    let mut json = Vec::with_capacity(depth * 2 + 1);
    json.extend(std::iter::repeat_n(b'[', depth));
    json.push(b'0');
    json.extend(std::iter::repeat_n(b']', depth));
    json
}

fn free_parser(parser: pure_simdjson_parser_t) {
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}

#[test]
fn configured_constructor_normalizes_defaults_and_rejects_invalid_capacity() {
    assert_eq!(
        unsafe { pure_simdjson_parser_new_configured(0, 0, ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    for capacity in [1, 31, u64::from(u32::MAX) + 1] {
        let mut parser = u64::MAX;
        assert_eq!(
            unsafe { pure_simdjson_parser_new_configured(capacity, 0, &mut parser) },
            PURE_SIMDJSON_ERR_INVALID_ARGUMENT,
            "capacity {capacity}"
        );
        assert_eq!(parser, u64::MAX, "capacity {capacity} changed output");
    }

    free_parser(parser_new_configured(0, 0));
    free_parser(parser_new_configured(32, 0));
}

#[test]
fn configured_capacity_accepts_exact_size_and_rejects_limit_plus_one() {
    let parser = parser_new_configured(32, 0);

    let exact = json_string_with_total_len(32);
    let (exact_rc, exact_doc) = parser_parse(parser, &exact);
    assert_eq!(exact_rc, PURE_SIMDJSON_OK);
    assert_ne!(exact_doc, 0);
    assert_eq!(
        unsafe { pure_simdjson_doc_free(exact_doc) },
        PURE_SIMDJSON_OK
    );

    let oversized = json_string_with_total_len(33);
    let (oversized_rc, oversized_doc) = parser_parse(parser, &oversized);
    assert_eq!(oversized_rc, PURE_SIMDJSON_ERR_CAPACITY_LIMIT);
    assert_eq!(oversized_doc, 0);

    free_parser(parser);
}

#[test]
fn configured_depth_accepts_n_minus_one_and_rejects_n() {
    const MAX_DEPTH: u32 = 4;
    let parser = parser_new_configured(0, MAX_DEPTH);

    let (accepted_rc, accepted_doc) = parser_parse(parser, &nested_array(MAX_DEPTH as usize - 1));
    assert_eq!(accepted_rc, PURE_SIMDJSON_OK);
    assert_ne!(accepted_doc, 0);
    assert_eq!(
        unsafe { pure_simdjson_doc_free(accepted_doc) },
        PURE_SIMDJSON_OK
    );

    let (rejected_rc, rejected_doc) = parser_parse(parser, &nested_array(MAX_DEPTH as usize));
    assert_eq!(rejected_rc, PURE_SIMDJSON_ERR_DEPTH_LIMIT);
    assert_eq!(rejected_doc, 0);

    free_parser(parser);
}

#[test]
fn legacy_constructor_keeps_default_depth_boundary() {
    let parser = parser_new();

    let (accepted_rc, accepted_doc) = parser_parse(parser, &nested_array(1023));
    assert_eq!(accepted_rc, PURE_SIMDJSON_OK);
    assert_ne!(accepted_doc, 0);
    assert_eq!(
        unsafe { pure_simdjson_doc_free(accepted_doc) },
        PURE_SIMDJSON_OK
    );

    let (rejected_rc, rejected_doc) = parser_parse(parser, &nested_array(1024));
    assert_eq!(rejected_rc, PURE_SIMDJSON_ERR_DEPTH_LIMIT);
    assert_eq!(rejected_doc, 0);

    free_parser(parser);
}

#[test]
fn capacity_rejection_clears_prior_native_diagnostics() {
    let parser = parser_new_configured(32, 0);

    let (invalid_rc, invalid_doc) = parser_parse(parser, b"{");
    assert_eq!(invalid_rc, PURE_SIMDJSON_ERR_INVALID_JSON);
    assert_eq!(invalid_doc, 0);
    assert!(!parser_last_error(parser).is_empty());

    let oversized = json_string_with_total_len(33);
    let (capacity_rc, capacity_doc) = parser_parse(parser, &oversized);
    assert_eq!(capacity_rc, PURE_SIMDJSON_ERR_CAPACITY_LIMIT);
    assert_eq!(capacity_doc, 0);
    assert_eq!(parser_last_error(parser), b"");
    assert_eq!(parser_last_error_offset(parser), u64::MAX);

    free_parser(parser);
}
