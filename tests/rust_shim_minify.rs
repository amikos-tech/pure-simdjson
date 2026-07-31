use std::{ptr, sync::Mutex};

use pure_simdjson::{
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED,
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT, PURE_SIMDJSON_ERR_INVALID_JSON, PURE_SIMDJSON_OK,
    },
    pure_simdjson_minify, pure_simdjson_test_set_allow_fallback_for_tests,
    pure_simdjson_test_set_forced_implementation_for_tests, pure_simdjson_validate_utf8,
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
fn minify_in_place_dst_equals_src_matches_non_aliased_output() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = br#"{"a": 1,  "b":   2}"#;

    let mut disjoint = vec![0_u8; input.len()];
    let mut disjoint_written = 0_usize;
    let disjoint_rc = unsafe {
        pure_simdjson_minify(
            input.as_ptr(),
            input.len(),
            disjoint.as_mut_ptr(),
            disjoint.len(),
            &mut disjoint_written,
        )
    };
    assert_eq!(disjoint_rc, PURE_SIMDJSON_OK);

    let mut in_place = input.to_vec();
    let mut in_place_written = 0_usize;
    let in_place_ptr = in_place.as_mut_ptr();
    let in_place_rc = unsafe {
        pure_simdjson_minify(
            in_place_ptr,
            in_place.len(),
            in_place_ptr,
            in_place.len(),
            &mut in_place_written,
        )
    };
    assert_eq!(in_place_rc, PURE_SIMDJSON_OK);
    assert_eq!(in_place_written, disjoint_written);
    assert_eq!(&in_place[..in_place_written], &disjoint[..disjoint_written]);
}

#[test]
fn minify_disjoint_buffer_succeeds() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = b"[ 1, 2, 3 ]";
    let mut output = vec![0xA5_u8; input.len()];
    let mut written = 0_usize;

    let rc = unsafe {
        pure_simdjson_minify(
            input.as_ptr(),
            input.len(),
            output.as_mut_ptr(),
            output.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(&output[..written], b"[1,2,3]");
}

#[test]
fn minify_rejects_destination_starting_inside_source_without_writing() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = br#"{"a": 1}"#;
    let mut storage = vec![0xA5_u8; input.len() + 1];
    storage[..input.len()].copy_from_slice(input);
    let before = storage.clone();
    let mut written = usize::MAX;

    let rc = unsafe {
        pure_simdjson_minify(
            storage.as_ptr(),
            input.len(),
            storage.as_mut_ptr().add(1),
            input.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_ARGUMENT);
    assert_eq!(storage, before);
}

#[test]
fn minify_rejects_source_starting_inside_destination_without_writing() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = br#"{"a": 1}"#;
    let mut storage = vec![0xA5_u8; input.len() + 1];
    storage[1..].copy_from_slice(input);
    let before = storage.clone();
    let mut written = usize::MAX;

    let rc = unsafe {
        pure_simdjson_minify(
            storage.as_ptr().add(1),
            input.len(),
            storage.as_mut_ptr(),
            input.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_ARGUMENT);
    assert_eq!(storage, before);
}

#[test]
fn minify_empty_input_returns_zero_written() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let mut written = usize::MAX;

    let rc = unsafe { pure_simdjson_minify(ptr::null(), 0, ptr::null_mut(), 0, &mut written) };

    assert_eq!(rc, PURE_SIMDJSON_OK);
    assert_eq!(written, 0);
}

#[test]
fn minify_unclosed_string_returns_invalid_json() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = b"\"unterminated";
    let mut output = vec![0_u8; input.len()];
    let mut written = 0_usize;

    let rc = unsafe {
        pure_simdjson_minify(
            input.as_ptr(),
            input.len(),
            output.as_mut_ptr(),
            output.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_INVALID_JSON);
}

#[test]
fn minify_undersized_dst_returns_buffer_too_small_before_writing() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = br#"{"a": 1}"#;
    let mut output = vec![0xA5_u8; input.len() - 1];
    let before = output.clone();
    let mut written = usize::MAX;

    let rc = unsafe {
        pure_simdjson_minify(
            input.as_ptr(),
            input.len(),
            output.as_mut_ptr(),
            output.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL);
    assert_eq!(output, before);
}

#[test]
fn utility_raw_pointer_boundaries_are_checked() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let input = b"x";
    let mut output = [0xA5_u8; 1];
    let mut written = usize::MAX;

    assert_eq!(
        unsafe {
            pure_simdjson_minify(
                ptr::null(),
                input.len(),
                output.as_mut_ptr(),
                output.len(),
                &mut written,
            )
        },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe {
            pure_simdjson_minify(
                input.as_ptr(),
                input.len(),
                ptr::null_mut(),
                input.len(),
                &mut written,
            )
        },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe {
            pure_simdjson_minify(
                input.as_ptr(),
                input.len(),
                output.as_mut_ptr(),
                output.len(),
                ptr::null_mut(),
            )
        },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );

    written = usize::MAX;
    assert_eq!(
        unsafe { pure_simdjson_minify(ptr::null(), 0, ptr::null_mut(), 0, &mut written) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(written, 0);

    let mut valid = 0xFF_u8;
    assert_eq!(
        unsafe { pure_simdjson_validate_utf8(ptr::null(), 1, &mut valid) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe { pure_simdjson_validate_utf8(input.as_ptr(), input.len(), ptr::null_mut()) },
        PURE_SIMDJSON_ERR_INVALID_ARGUMENT
    );
    assert_eq!(
        unsafe { pure_simdjson_validate_utf8(ptr::null(), 0, &mut valid) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(valid, 1);
}

#[test]
fn validate_utf8_accepts_valid_and_rejects_invalid() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    let valid_input = "valid λ".as_bytes();
    let invalid_input = [0x80_u8];
    let mut valid = 0_u8;

    assert_eq!(
        unsafe { pure_simdjson_validate_utf8(valid_input.as_ptr(), valid_input.len(), &mut valid) },
        PURE_SIMDJSON_OK
    );
    assert_eq!(valid, 1);

    assert_eq!(
        unsafe {
            pure_simdjson_validate_utf8(invalid_input.as_ptr(), invalid_input.len(), &mut valid)
        },
        PURE_SIMDJSON_OK
    );
    assert_eq!(valid, 0);
}

#[test]
fn minify_rejects_fallback_without_bypass() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    pure_simdjson_test_set_forced_implementation_for_tests(Some(&b"fallback"[..]));
    let input = b"{}";
    let mut output = [0_u8; 2];
    let mut written = 0_usize;

    let rc = unsafe {
        pure_simdjson_minify(
            input.as_ptr(),
            input.len(),
            output.as_mut_ptr(),
            output.len(),
            &mut written,
        )
    };

    assert_eq!(rc, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
}

#[test]
fn validate_utf8_rejects_fallback_without_bypass() {
    let _override_lock = TEST_OVERRIDE_LOCK
        .lock()
        .expect("utility override lock poisoned");
    let _env_guard = EnvGuard::new();
    pure_simdjson_test_set_forced_implementation_for_tests(Some(&b"fallback"[..]));
    let input = b"valid";
    let mut valid = 0_u8;

    let rc = unsafe { pure_simdjson_validate_utf8(input.as_ptr(), input.len(), &mut valid) };

    assert_eq!(rc, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
}
