use std::sync::Mutex;

use pure_simdjson::{
    pure_simdjson_error_code_t::{PURE_SIMDJSON_ERR_CPU_UNSUPPORTED, PURE_SIMDJSON_OK},
    pure_simdjson_parser_free, pure_simdjson_parser_new,
    pure_simdjson_test_set_allow_fallback_for_tests,
    pure_simdjson_test_set_forced_implementation_for_tests,
};

static TEST_OVERRIDE_LOCK: Mutex<()> = Mutex::new(());

struct EnvGuard;

impl EnvGuard {
    fn new() -> Self {
        pure_simdjson_test_set_forced_implementation_for_tests(None);
        pure_simdjson_test_set_allow_fallback_for_tests(None);
        EnvGuard
    }
}

impl Drop for EnvGuard {
    fn drop(&mut self) {
        pure_simdjson_test_set_forced_implementation_for_tests(None);
        pure_simdjson_test_set_allow_fallback_for_tests(None);
    }
}

#[test]
fn parser_new_rejects_fallback_without_bypass() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("fallback override lock poisoned");
    let _env_guard = EnvGuard::new();
    pure_simdjson_test_set_forced_implementation_for_tests(Some(&b"fallback"[..]));

    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
    assert_eq!(parser, 0);
}

#[test]
fn parser_new_allows_fallback_with_hidden_bypass() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("fallback override lock poisoned");
    let _env_guard = EnvGuard::new();
    pure_simdjson_test_set_forced_implementation_for_tests(Some(&b"fallback"[..]));
    pure_simdjson_test_set_allow_fallback_for_tests(Some(true));

    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    assert_eq!(
        unsafe { pure_simdjson_parser_free(parser) },
        PURE_SIMDJSON_OK
    );
}
