use pure_simdjson::{
    pure_simdjson_doc_t,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_INVALID_JSON, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_parse,
    pure_simdjson_parser_t,
};

fn parser_new() -> pure_simdjson_parser_t {
    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };
    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    parser
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
