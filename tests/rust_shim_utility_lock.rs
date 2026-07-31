use std::ptr;

use pure_simdjson::{
    pure_simdjson_copy_implementation_name,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CPU_UNSUPPORTED, PURE_SIMDJSON_ERR_KERNEL_LOCKED, PURE_SIMDJSON_OK,
    },
    pure_simdjson_get_implementation_name_len, pure_simdjson_set_implementation,
    pure_simdjson_test_set_forced_implementation_for_tests, pure_simdjson_validate_utf8,
};

fn implementation_name() -> Vec<u8> {
    let mut len = 0_usize;
    assert_eq!(
        unsafe { pure_simdjson_get_implementation_name_len(&mut len) },
        PURE_SIMDJSON_OK
    );
    assert_ne!(len, 0);

    let mut name = vec![0_u8; len];
    let mut written = 0_usize;
    assert_eq!(
        unsafe {
            pure_simdjson_copy_implementation_name(name.as_mut_ptr(), name.len(), &mut written)
        },
        PURE_SIMDJSON_OK
    );
    assert_eq!(written, len);
    name
}

fn set_implementation(name: &[u8]) -> pure_simdjson::pure_simdjson_error_code_t {
    let name_ptr = if name.is_empty() {
        ptr::null()
    } else {
        name.as_ptr()
    };
    unsafe { pure_simdjson_set_implementation(name_ptr, name.len()) }
}

#[test]
fn forced_fallback_rejection_preserves_native_state_then_success_locks_selection() {
    pure_simdjson_test_set_forced_implementation_for_tests(Some(&b"fallback"[..]));
    let mut valid = 0_u8;
    let rejected = unsafe { pure_simdjson_validate_utf8(ptr::null(), 0, &mut valid) };
    pure_simdjson_test_set_forced_implementation_for_tests(None);

    // The forced override is rejected by Rust before the C++ bridge. A setter
    // succeeding here proves that pre-FFI rejection left native selection
    // unlocked; it deliberately does not claim C++ implicit-fallback coverage.
    assert_eq!(rejected, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
    let current = implementation_name();
    assert_eq!(set_implementation(&current), PURE_SIMDJSON_OK);

    let input = b"valid";
    assert_eq!(
        unsafe { pure_simdjson_validate_utf8(input.as_ptr(), input.len(), &mut valid) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(valid, 1);
    assert_eq!(
        set_implementation(&current),
        PURE_SIMDJSON_ERR_KERNEL_LOCKED
    );
}
