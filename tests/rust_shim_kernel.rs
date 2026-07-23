use std::{
    env,
    process::{Command, Output},
    ptr,
    sync::{Arc, Barrier},
    thread,
};

use pure_simdjson::{
    pure_simdjson_copy_implementation_name,
    pure_simdjson_error_code_t::{
        PURE_SIMDJSON_ERR_CPU_UNSUPPORTED, PURE_SIMDJSON_ERR_INVALID_ARGUMENT,
        PURE_SIMDJSON_ERR_KERNEL_LOCKED, PURE_SIMDJSON_OK,
    },
    pure_simdjson_get_implementation_name_len, pure_simdjson_lock_implementation_selection,
    pure_simdjson_parser_free, pure_simdjson_parser_new, pure_simdjson_parser_new_configured,
    pure_simdjson_set_implementation, pure_simdjson_test_set_forced_implementation_for_tests,
};

const SCENARIO_ENV: &str = "PURE_SIMDJSON_KERNEL_SCENARIO";
const SCENARIOS: &[&str] = &[
    "auto",
    "valid",
    "invalid",
    "unsupported_when_observable",
    "explicit_fallback",
    "automatic_fallback",
    "post_parser_lock",
    "post_configured_parser_lock",
    "explicit_lock",
    "setter_constructor_race",
];

fn implementation_name() -> Vec<u8> {
    let mut len = 0_usize;
    assert_eq!(
        unsafe { pure_simdjson_get_implementation_name_len(&mut len) },
        PURE_SIMDJSON_OK
    );
    assert_ne!(len, 0, "active implementation name must not be empty");

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

fn parser_new() -> (pure_simdjson::pure_simdjson_error_code_t, u64) {
    let mut parser = 0_u64;
    let rc = unsafe { pure_simdjson_parser_new(&mut parser) };
    (rc, parser)
}

fn assert_child_success(output: &Output, scenario: &str) {
    assert!(
        output.status.success(),
        "kernel scenario {scenario:?} failed: status={:?}\nstdout={}\nstderr={}",
        output.status.code(),
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(
        stdout.contains("1 passed"),
        "kernel scenario {scenario:?} did not run exactly one test:\n{stdout}",
    );
}

fn run_child_scenario(scenario: &str) {
    match scenario {
        "auto" => {
            assert_eq!(set_implementation(b""), PURE_SIMDJSON_OK);
            assert!(!implementation_name().is_empty());
        }
        "valid" => {
            let supported_name = implementation_name();
            assert_eq!(set_implementation(&supported_name), PURE_SIMDJSON_OK);
            assert_eq!(implementation_name(), supported_name);
        }
        "invalid" => {
            let before = implementation_name();
            assert_eq!(
                set_implementation(b"not-a-compiled-implementation"),
                PURE_SIMDJSON_ERR_INVALID_ARGUMENT
            );
            assert_eq!(implementation_name(), before);

            assert_eq!(
                unsafe { pure_simdjson_set_implementation(ptr::null(), 1) },
                PURE_SIMDJSON_ERR_INVALID_ARGUMENT
            );
            assert_eq!(implementation_name(), before);
        }
        "unsupported_when_observable" => {
            let candidates: &[&[u8]] = &[
                b"icelake",
                b"haswell",
                b"westmere",
                b"arm64",
                b"ppc64",
                b"lasx",
                b"lsx",
            ];
            let mut observed = false;

            for candidate in candidates {
                let before = implementation_name();
                match set_implementation(candidate) {
                    PURE_SIMDJSON_ERR_CPU_UNSUPPORTED => {
                        assert_eq!(implementation_name(), before);
                        observed = true;
                        break;
                    }
                    PURE_SIMDJSON_ERR_INVALID_ARGUMENT => {
                        assert_eq!(implementation_name(), before);
                    }
                    PURE_SIMDJSON_OK => {
                        assert_eq!(implementation_name(), *candidate);
                    }
                    rc => panic!(
                        "unexpected status {rc:?} while probing implementation {:?}",
                        String::from_utf8_lossy(candidate)
                    ),
                }
            }

            if !observed {
                eprintln!("no compiled-but-unsupported implementation is observable on this host");
            }
        }
        "explicit_fallback" => {
            assert_eq!(set_implementation(b"fallback"), PURE_SIMDJSON_OK);
            assert_eq!(implementation_name(), b"fallback");

            let (rc, parser) = parser_new();
            assert_eq!(rc, PURE_SIMDJSON_OK);
            assert_ne!(parser, 0);
            assert_eq!(
                unsafe { pure_simdjson_parser_free(parser) },
                PURE_SIMDJSON_OK
            );
        }
        "automatic_fallback" => {
            pure_simdjson_test_set_forced_implementation_for_tests(Some(b"fallback"));
            let (rc, parser) = parser_new();
            pure_simdjson_test_set_forced_implementation_for_tests(None);

            assert_eq!(rc, PURE_SIMDJSON_ERR_CPU_UNSUPPORTED);
            assert_eq!(parser, 0);
        }
        "post_parser_lock" => {
            let current = implementation_name();
            let (rc, parser) = parser_new();
            assert_eq!(rc, PURE_SIMDJSON_OK);
            assert_ne!(parser, 0);
            assert_eq!(
                set_implementation(&current),
                PURE_SIMDJSON_ERR_KERNEL_LOCKED
            );
            assert_eq!(
                unsafe { pure_simdjson_parser_free(parser) },
                PURE_SIMDJSON_OK
            );
        }
        "post_configured_parser_lock" => {
            let current = implementation_name();
            let mut parser = 0_u64;
            assert_eq!(
                unsafe { pure_simdjson_parser_new_configured(0, 0, &mut parser) },
                PURE_SIMDJSON_OK
            );
            assert_ne!(parser, 0);
            assert_eq!(
                set_implementation(&current),
                PURE_SIMDJSON_ERR_KERNEL_LOCKED
            );
            assert_eq!(
                unsafe { pure_simdjson_parser_free(parser) },
                PURE_SIMDJSON_OK
            );
        }
        "explicit_lock" => {
            let current = implementation_name();
            assert_eq!(
                pure_simdjson_lock_implementation_selection(),
                PURE_SIMDJSON_OK
            );
            assert_eq!(
                set_implementation(&current),
                PURE_SIMDJSON_ERR_KERNEL_LOCKED
            );
        }
        "setter_constructor_race" => {
            let implementation = Arc::new(implementation_name());
            let barrier = Arc::new(Barrier::new(3));

            let setter_barrier = Arc::clone(&barrier);
            let setter_implementation = Arc::clone(&implementation);
            let setter = thread::spawn(move || {
                setter_barrier.wait();
                set_implementation(&setter_implementation)
            });

            let constructor_barrier = Arc::clone(&barrier);
            let constructor = thread::spawn(move || {
                constructor_barrier.wait();
                parser_new()
            });

            barrier.wait();
            let setter_rc = setter.join().expect("setter thread panicked");
            let (constructor_rc, parser) = constructor.join().expect("constructor thread panicked");

            assert!(
                matches!(
                    setter_rc,
                    PURE_SIMDJSON_OK | PURE_SIMDJSON_ERR_KERNEL_LOCKED
                ),
                "setter returned unexpected status {setter_rc:?}"
            );
            assert_eq!(constructor_rc, PURE_SIMDJSON_OK);
            assert_ne!(parser, 0);
            assert_eq!(
                unsafe { pure_simdjson_parser_free(parser) },
                PURE_SIMDJSON_OK
            );
        }
        other => panic!("unknown kernel scenario {other:?}"),
    }
}

#[test]
fn kernel_selection_scenarios_are_process_isolated() {
    if let Ok(scenario) = env::var(SCENARIO_ENV) {
        run_child_scenario(&scenario);
        return;
    }

    for scenario in SCENARIOS {
        let output = Command::new(env::current_exe().expect("kernel test binary path"))
            .env(SCENARIO_ENV, scenario)
            .arg("--exact")
            .arg("kernel_selection_scenarios_are_process_isolated")
            .arg("--nocapture")
            .output()
            .unwrap_or_else(|error| panic!("spawn kernel scenario {scenario:?}: {error}"));

        assert_child_success(&output, scenario);
    }
}
