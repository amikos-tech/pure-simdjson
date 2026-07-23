use pure_simdjson::{
    pure_simdjson_doc_free,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CAPACITY_LIMIT, PURE_SIMDJSON_ERR_DEPTH_LIMIT,
        PURE_SIMDJSON_ERR_INVALID_JSON, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_free, pure_simdjson_parser_get_last_error_offset,
    pure_simdjson_parser_new_configured, pure_simdjson_parser_parse, pure_simdjson_parser_t,
};

const DEFAULT_MAX_CAPACITY: u64 = u32::MAX as u64;
const DEFAULT_MAX_DEPTH: u32 = 1024;

const POINTER_NOT_QUERIED: u32 = 0;
const POINTER_IN_BOUNDS: u32 = 1;

const REPLAY_NONE: u32 = 0;
const REPLAY_RAW_JSON: u32 = 1;
const REPLAY_RECURSIVE: u32 = 2;

const TERMINAL_CAPACITY: u32 = 1;
const TERMINAL_DEPTH: u32 = 2;
const TERMINAL_MEMALLOC: u32 = 3;
const TERMINAL_INTERNAL: u32 = 4;

#[repr(C)]
#[derive(Clone, Copy, Debug, Default)]
struct ReplayObservation {
    primary_error: i32,
    replay_error: i32,
    location_error: i32,
    replay_pass: u32,
    pointer_relation: u32,
    pass_count: u32,
    allocation_count: u32,
    iterate_count: u32,
    derived_offset: u64,
    first_max_capacity: u64,
    second_max_capacity: u64,
    first_max_depth: u32,
    second_max_depth: u32,
}

unsafe extern "C" {
    fn psimdjson_test_characterize_diagnostic(
        input_ptr: *const u8,
        input_len: usize,
        max_capacity: u64,
        max_depth: u32,
        out_parse_status: *mut i32,
        out_offset: *mut u64,
        out_has_offset: *mut u8,
        out_observation: *mut ReplayObservation,
    ) -> i32;
    fn psimdjson_test_recursive_replay_observation(
        input_ptr: *const u8,
        input_len: usize,
        max_capacity: u64,
        max_depth: u32,
        out_offset: *mut u64,
        out_has_offset: *mut u8,
        out_observation: *mut ReplayObservation,
    ) -> i32;
    fn psimdjson_test_terminal_diagnostic_observation(
        terminal_case: u32,
        out_offset: *mut u64,
        out_has_offset: *mut u8,
        out_observation: *mut ReplayObservation,
    ) -> i32;
    fn psimdjson_test_checked_diagnostic_offset(
        input_addr: usize,
        input_len: usize,
        location_addr: usize,
        out_offset: *mut u64,
        out_has_offset: *mut u8,
    ) -> i32;
}

#[derive(Debug)]
struct Characterization {
    parse_status: i32,
    offset: u64,
    has_offset: bool,
    observation: ReplayObservation,
}

fn characterize(json: &[u8], max_capacity: u64, max_depth: u32) -> Characterization {
    let mut parse_status = i32::MIN;
    let mut offset = 0;
    let mut has_offset = u8::MAX;
    let mut observation = ReplayObservation::default();
    let rc = unsafe {
        psimdjson_test_characterize_diagnostic(
            json.as_ptr(),
            json.len(),
            max_capacity,
            max_depth,
            &mut parse_status,
            &mut offset,
            &mut has_offset,
            &mut observation,
        )
    };
    assert_eq!(rc, PURE_SIMDJSON_OK as i32);
    assert!(has_offset <= 1);
    Characterization {
        parse_status,
        offset,
        has_offset: has_offset != 0,
        observation,
    }
}

fn parser_new_configured(max_capacity: u64, max_depth: u32) -> pure_simdjson_parser_t {
    let mut parser = 0;
    assert_eq!(
        unsafe { pure_simdjson_parser_new_configured(max_capacity, max_depth, &mut parser) },
        PURE_SIMDJSON_OK
    );
    assert_ne!(parser, 0);
    parser
}

fn parse_status(parser: pure_simdjson_parser_t, json: &[u8]) -> i32 {
    let mut doc = 0;
    let rc = unsafe { pure_simdjson_parser_parse(parser, json.as_ptr(), json.len(), &mut doc) };
    if rc == PURE_SIMDJSON_OK {
        assert_ne!(doc, 0);
        assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    } else {
        assert_eq!(doc, 0);
    }
    rc as i32
}

fn last_error_offset(parser: pure_simdjson_parser_t) -> u64 {
    let mut offset = 0;
    assert_eq!(
        unsafe { pure_simdjson_parser_get_last_error_offset(parser, &mut offset) },
        PURE_SIMDJSON_OK
    );
    offset
}

fn nested_array(depth: usize, malformed: bool) -> Vec<u8> {
    let mut json = Vec::with_capacity(depth * 2 + 2);
    json.extend(std::iter::repeat_n(b'[', depth));
    json.push(b'1');
    if malformed {
        json.push(b',');
    }
    json.extend(std::iter::repeat_n(b']', depth));
    json
}

#[test]
fn characterize_v464_error_locations() {
    let mut invalid_utf8 = br#"{"x":""#.to_vec();
    invalid_utf8.push(0xff);
    invalid_utf8.extend_from_slice(br#""}"#);

    let cases: [(&str, Vec<u8>); 9] = [
        ("empty", Vec::new()),
        ("invalid_utf8", invalid_utf8),
        ("unclosed_string", br#"{"x":"abc}"#.to_vec()),
        ("array_trailing_comma", b"[1,]".to_vec()),
        ("trailing_content", br#"{"a":1} trailing"#.to_vec()),
        (
            "missing_object_key",
            br#"{"double":13.06,false,"integer":-343}"#.to_vec(),
        ),
        ("unexpected_root_token", b"x".to_vec()),
        ("extra_closing_bracket", br#"["extra close"]]"#.to_vec()),
        ("mismatched_container", br#"{"a":[1,2}"#.to_vec()),
    ];

    eprintln!(
        "case\tprimary_status\treplay_pass\treplay_status\tlocation_status\tpointer_relation\toffset\tknown"
    );
    for (name, input) in cases {
        let result = characterize(&input, DEFAULT_MAX_CAPACITY, DEFAULT_MAX_DEPTH);
        let observation = result.observation;
        eprintln!(
            "{name}\t{}\t{}\t{}\t{}\t{}\t{}\t{}",
            observation.primary_error,
            observation.replay_pass,
            observation.replay_error,
            observation.location_error,
            observation.pointer_relation,
            result.offset,
            result.has_offset,
        );

        assert_eq!(result.parse_status, PURE_SIMDJSON_ERR_INVALID_JSON as i32);
        assert!(matches!(
            observation.replay_pass,
            REPLAY_NONE | REPLAY_RAW_JSON | REPLAY_RECURSIVE
        ));
        assert!(observation.pass_count <= 2);
        assert_eq!(observation.pass_count, observation.allocation_count);
        assert_eq!(observation.pass_count, observation.iterate_count);
        if result.has_offset {
            assert_eq!(observation.pointer_relation, POINTER_IN_BOUNDS);
            assert!(result.offset < input.len() as u64);
        } else {
            assert_eq!(result.offset, u64::MAX);
        }
    }

    let success = characterize(b"{}", DEFAULT_MAX_CAPACITY, DEFAULT_MAX_DEPTH);
    assert_eq!(success.parse_status, PURE_SIMDJSON_OK as i32);
    assert_eq!(success.observation.replay_pass, REPLAY_NONE);
    assert_eq!(success.observation.pass_count, 0);
    assert_eq!(success.offset, u64::MAX);
    assert!(!success.has_offset);
}

#[test]
fn diagnostic_replay_respects_capacity_n_minus_1_and_n() {
    const INPUT_LEN: usize = 33;
    let mut malformed = vec![b' '; INPUT_LEN];
    malformed[..4].copy_from_slice(b"[1,]");

    let parser = parser_new_configured((INPUT_LEN - 1) as u64, 5);
    assert_eq!(
        parse_status(parser, &malformed),
        PURE_SIMDJSON_ERR_CAPACITY_LIMIT as i32
    );
    assert_eq!(last_error_offset(parser), u64::MAX);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let below = characterize(&malformed, (INPUT_LEN - 1) as u64, 5);
    assert_eq!(below.observation.pass_count, 0);
    assert_eq!(below.observation.replay_pass, REPLAY_NONE);
    assert_eq!(below.offset, u64::MAX);
    assert!(!below.has_offset);

    let exact = characterize(&malformed, INPUT_LEN as u64, 5);
    assert_eq!(exact.parse_status, PURE_SIMDJSON_ERR_INVALID_JSON as i32);
    assert_eq!(exact.observation.replay_pass, REPLAY_RECURSIVE);
    assert_eq!(exact.observation.pass_count, 2);
    assert_eq!(exact.observation.first_max_capacity, INPUT_LEN as u64);
    assert_eq!(exact.observation.second_max_capacity, INPUT_LEN as u64);
    assert_eq!(exact.observation.first_max_depth, 5);
    assert_eq!(exact.observation.second_max_depth, 5);
}

#[test]
fn diagnostic_replay_respects_depth_n_minus_1_and_n() {
    const MAX_DEPTH: u32 = 4;
    let n_minus_1 = nested_array(MAX_DEPTH as usize - 1, true);
    let n = nested_array(MAX_DEPTH as usize, true);

    let parser = parser_new_configured(0, MAX_DEPTH);
    assert_eq!(
        parse_status(parser, &n_minus_1),
        PURE_SIMDJSON_ERR_INVALID_JSON as i32
    );
    assert_eq!(
        parse_status(parser, &n),
        PURE_SIMDJSON_ERR_DEPTH_LIMIT as i32
    );
    assert_eq!(last_error_offset(parser), u64::MAX);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let accepted_depth = characterize(&n_minus_1, DEFAULT_MAX_CAPACITY, MAX_DEPTH);
    assert_eq!(accepted_depth.observation.replay_pass, REPLAY_RECURSIVE);
    assert_eq!(accepted_depth.observation.pass_count, 2);
    assert_eq!(accepted_depth.observation.first_max_depth, MAX_DEPTH);
    assert_eq!(accepted_depth.observation.second_max_depth, MAX_DEPTH);

    let rejected_depth = characterize(&n, DEFAULT_MAX_CAPACITY, MAX_DEPTH);
    assert_eq!(rejected_depth.observation.pass_count, 0);
    assert_eq!(
        rejected_depth.observation.pointer_relation,
        POINTER_NOT_QUERIED
    );
    assert_eq!(rejected_depth.offset, u64::MAX);
    assert!(!rejected_depth.has_offset);

    let recursive_boundary = nested_array(MAX_DEPTH as usize, false);
    let mut offset = 0;
    let mut has_offset = u8::MAX;
    let mut observation = ReplayObservation::default();
    assert_eq!(
        unsafe {
            psimdjson_test_recursive_replay_observation(
                recursive_boundary.as_ptr(),
                recursive_boundary.len(),
                DEFAULT_MAX_CAPACITY,
                MAX_DEPTH,
                &mut offset,
                &mut has_offset,
                &mut observation,
            )
        },
        PURE_SIMDJSON_OK as i32
    );
    assert_eq!(observation.replay_pass, REPLAY_RECURSIVE);
    assert_eq!(observation.pass_count, 1);
    assert_eq!(observation.allocation_count, 1);
    assert_eq!(observation.iterate_count, 1);
    assert_eq!(offset, u64::MAX);
    assert_eq!(has_offset, 0);
}

#[test]
fn diagnostic_replay_resource_failures_stay_unknown() {
    for terminal_case in [
        TERMINAL_CAPACITY,
        TERMINAL_DEPTH,
        TERMINAL_MEMALLOC,
        TERMINAL_INTERNAL,
    ] {
        let mut offset = 0;
        let mut has_offset = u8::MAX;
        let mut observation = ReplayObservation::default();
        assert_eq!(
            unsafe {
                psimdjson_test_terminal_diagnostic_observation(
                    terminal_case,
                    &mut offset,
                    &mut has_offset,
                    &mut observation,
                )
            },
            PURE_SIMDJSON_OK as i32,
            "terminal case {terminal_case}"
        );
        assert_eq!(observation.replay_pass, REPLAY_NONE);
        assert_eq!(observation.pass_count, 0);
        assert_eq!(observation.allocation_count, 0);
        assert_eq!(observation.iterate_count, 0);
        assert_eq!(observation.pointer_relation, POINTER_NOT_QUERIED);
        assert_eq!(offset, u64::MAX);
        assert_eq!(has_offset, 0);
    }
}

#[test]
fn diagnostic_pointer_range_requires_an_in_bounds_non_end_address() {
    fn checked(input_addr: usize, input_len: usize, location_addr: usize) -> (u64, u8) {
        let mut offset = 0;
        let mut has_offset = u8::MAX;
        assert_eq!(
            unsafe {
                psimdjson_test_checked_diagnostic_offset(
                    input_addr,
                    input_len,
                    location_addr,
                    &mut offset,
                    &mut has_offset,
                )
            },
            PURE_SIMDJSON_OK as i32
        );
        (offset, has_offset)
    }

    assert_eq!(checked(0x1000, 8, 0x1000), (0, 1));
    assert_eq!(checked(0x1000, 8, 0x1007), (7, 1));
    assert_eq!(checked(0x1000, 8, 0x1008), (u64::MAX, 0));
    assert_eq!(checked(0x1000, 8, 0x0fff), (u64::MAX, 0));
    assert_eq!(checked(0x1000, 8, 0x1009), (u64::MAX, 0));
    assert_eq!(checked(usize::MAX - 3, 8, usize::MAX - 3), (u64::MAX, 0));
}
