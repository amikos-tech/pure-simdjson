// Dedicated test binary for tests that assert exact values of the process-global
// native_alloc_telemetry counters. Cargo runs tests within a single binary in parallel by
// default, and the telemetry state is a shared singleton — every parser created by a
// concurrent test pollutes the live_bytes / alloc_count totals these tests read.
//
// Keeping them in their own binary isolates the process so only the tests below allocate
// against the tracker, and the module-level mutex serialises the two tests against each
// other so their reset/snapshot windows cannot interleave.

use std::sync::Mutex;

use pure_simdjson::{
    pure_simdjson_doc_free, pure_simdjson_doc_t,
    pure_simdjson_error_code_t::PURE_SIMDJSON_OK, pure_simdjson_native_alloc_stats_reset,
    pure_simdjson_native_alloc_stats_snapshot, pure_simdjson_native_alloc_stats_t,
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_parse,
    pure_simdjson_parser_t,
};

// Serialises the two tests in this binary; without it they race on the shared
// telemetry_state() singleton and one test's allocations pollute the other's assertions.
static TELEMETRY_TEST_LOCK: Mutex<()> = Mutex::new(());

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

#[test]
fn native_alloc_stats_round_trip_before_and_after_parse_activity() {
    let _guard = TELEMETRY_TEST_LOCK.lock().unwrap_or_else(|p| p.into_inner());

    let parser = parser_new();
    native_alloc_stats_reset();

    let before = native_alloc_stats_snapshot();
    assert_ne!(before.epoch, 0);
    assert_eq!(before.live_bytes, 0);
    assert_eq!(before.total_alloc_bytes, 0);
    assert_eq!(before.alloc_count, 0);
    assert_eq!(before.free_count, 0);
    assert_eq!(before.untracked_free_count, 0);

    let doc = parser_parse_literal(parser, br#"{"value":[1,2,3],"label":"hello"}"#);
    let during = native_alloc_stats_snapshot();
    assert_eq!(during.epoch, before.epoch);
    assert!(during.live_bytes > 0);
    assert!(during.total_alloc_bytes >= during.live_bytes);
    assert!(during.alloc_count > 0);
    assert_eq!(during.untracked_free_count, 0);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let after = native_alloc_stats_snapshot();
    assert_eq!(after.epoch, before.epoch);
    assert_eq!(after.live_bytes, 0);
    assert!(after.free_count > 0);
    assert_eq!(after.untracked_free_count, 0);
}

#[test]
fn native_alloc_stats_reset_excludes_preexisting_live_allocations() {
    let _guard = TELEMETRY_TEST_LOCK.lock().unwrap_or_else(|p| p.into_inner());

    let parser = parser_new();
    let doc = parser_parse_literal(parser, b"[1,2,3]");
    let warm = native_alloc_stats_snapshot();
    assert!(warm.live_bytes > 0);

    native_alloc_stats_reset();
    let reset = native_alloc_stats_snapshot();
    assert_ne!(reset.epoch, warm.epoch);
    assert_eq!(reset.live_bytes, 0);
    assert_eq!(reset.total_alloc_bytes, 0);
    assert_eq!(reset.alloc_count, 0);
    assert_eq!(reset.free_count, 0);
    assert_eq!(reset.untracked_free_count, 0);

    assert_eq!(unsafe { pure_simdjson_doc_free(doc) }, PURE_SIMDJSON_OK);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );

    let after = native_alloc_stats_snapshot();
    assert_eq!(after.epoch, reset.epoch);
    assert_eq!(after.live_bytes, 0);
    assert_eq!(after.total_alloc_bytes, 0);
    assert_eq!(after.alloc_count, 0);
    assert_eq!(after.free_count, 0);
    assert_eq!(after.untracked_free_count, 0);
}
