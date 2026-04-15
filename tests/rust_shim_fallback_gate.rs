use std::sync::Mutex;

use pure_simdjson::{
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CPU_UNSUPPORTED, PURE_SIMDJSON_OK,
    },
    pure_simdjson_parser_free, pure_simdjson_parser_new,
};

static ENV_LOCK: Mutex<()> = Mutex::new(());

fn clear_test_env() {
    std::env::remove_var("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION");
    std::env::remove_var("PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS");
}

struct EnvGuard;

impl EnvGuard {
    fn new() -> Self {
        clear_test_env();
        EnvGuard
    }
}

impl Drop for EnvGuard {
    fn drop(&mut self) {
        clear_test_env();
    }
}

#[test]
fn parser_new_rejects_fallback_without_bypass() {
    let _env_lock = ENV_LOCK.lock().expect("fallback env lock poisoned");
    let _env_guard = EnvGuard::new();
    std::env::set_var("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION", "fallback");

    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
    assert_eq!(parser, 0);
}

#[test]
fn parser_new_allows_fallback_with_hidden_bypass() {
    let _env_lock = ENV_LOCK.lock().expect("fallback env lock poisoned");
    let _env_guard = EnvGuard::new();
    std::env::set_var("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION", "fallback");
    std::env::set_var("PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS", "1");

    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_ne!(parser, 0);
    assert_eq!(unsafe { pure_simdjson_parser_free(parser) }, PURE_SIMDJSON_OK);
}
