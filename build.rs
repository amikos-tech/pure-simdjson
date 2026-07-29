use std::{
    env, fs,
    path::{Path, PathBuf},
    process::Command,
};

const SIMDJSON_BASE_COMMIT: &str = "1bcf71bd85059ab6574ea1159de9298dcc1212c5";
const SIMDJSON_DIR: &str = "third_party/simdjson";
const SIMDJSON_PATCH: &str = "patches/simdjson-v4.6.4-positive-bigint.patch";
const PATCHED_BIGINT_IMPLEMENTATION_COUNT: usize = 9;
const PATCHED_BIGINT_TOO_MANY_DIGITS: &str = concat!(
    "if (digit_count > longest_digit_count) {\n",
    "    if (jsoncharutils::is_not_structural_or_whitespace(*p)) { return INVALID_NUMBER(src); }\n",
    "    return BIGINT_NUMBER(src);\n",
    "  }",
);
const PATCHED_BIGINT_NEGATIVE_OVERFLOW: &str = concat!(
    "if (i > uint64_t(INT64_MAX)+1) {\n",
    "        if (jsoncharutils::is_not_structural_or_whitespace(*p)) { return INVALID_NUMBER(src); }\n",
    "        return BIGINT_NUMBER(src);\n",
    "      }",
);
const PATCHED_BIGINT_POSITIVE_OVERFLOW: &str = concat!(
    "}  else if (src[0] != uint8_t('1') || i <= uint64_t(INT64_MAX)) {\n",
    "      if (jsoncharutils::is_not_structural_or_whitespace(*p)) { return INVALID_NUMBER(src); }\n",
    "      return BIGINT_NUMBER(src);\n",
    "    }",
);
const UNGUARDED_BIGINT_TOO_MANY_DIGITS: &str =
    "if (digit_count > longest_digit_count) { return BIGINT_NUMBER(src); }";
const UNGUARDED_BIGINT_NEGATIVE_OVERFLOW: &str =
    "if (i > uint64_t(INT64_MAX)+1) { return BIGINT_NUMBER(src);  }";
const UNGUARDED_BIGINT_POSITIVE_OVERFLOW: &str =
    "}  else if (src[0] != uint8_t('1') || i <= uint64_t(INT64_MAX)) { return BIGINT_NUMBER(src); }";

fn require_success(command: &mut Command, context: &str) {
    let status = command
        .status()
        .unwrap_or_else(|error| panic!("{context}: {error}"));
    assert!(status.success(), "{context}");
}

fn require_bigint_branch_parity(source: &str, branch: &str, guarded: &str, unguarded: &str) {
    assert_eq!(
        source.matches(guarded).count(),
        PATCHED_BIGINT_IMPLEMENTATION_COUNT,
        "guarded {branch} BigInt branch must occur exactly once in every singleheader implementation"
    );
    assert_eq!(
        source.matches(unguarded).count(),
        0,
        "unguarded {branch} BigInt branch remains in the patched singleheader"
    );
}

fn patched_simdjson_source(source: &str) -> PathBuf {
    let head = Command::new("git")
        .args(["-C", SIMDJSON_DIR, "rev-parse", "HEAD"])
        .output()
        .expect("failed to verify the simdjson base commit");
    assert!(
        head.status.success(),
        "failed to verify the simdjson base commit"
    );
    assert_eq!(
        String::from_utf8_lossy(&head.stdout).trim(),
        SIMDJSON_BASE_COMMIT,
        "simdjson base drifted; expected {SIMDJSON_BASE_COMMIT}, got {}",
        String::from_utf8_lossy(&head.stdout).trim()
    );

    require_success(
        Command::new("git").args(["-C", SIMDJSON_DIR, "diff", "--quiet", "HEAD", "--"]),
        "simdjson has tracked working-tree changes",
    );
    require_success(
        Command::new("git").args([
            "-C",
            SIMDJSON_DIR,
            "diff",
            "--cached",
            "--quiet",
            "HEAD",
            "--",
        ]),
        "simdjson has staged changes",
    );

    let out_dir = PathBuf::from(env::var_os("OUT_DIR").expect("OUT_DIR must be set"));
    let patched_source = out_dir.join("simdjson.cpp");
    fs::copy(source, &patched_source)
        .unwrap_or_else(|error| panic!("failed to copy audited simdjson source: {error}"));

    let package_root =
        PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR must be set"));
    let patch = package_root.join(SIMDJSON_PATCH);
    require_success(
        Command::new("git")
            .current_dir(&out_dir)
            .env("GIT_CEILING_DIRECTORIES", &package_root)
            .args(["apply", "--check", "--ignore-space-change"])
            .arg(&patch),
        "simdjson patch does not match the audited base",
    );
    require_success(
        Command::new("git")
            .current_dir(&out_dir)
            .env("GIT_CEILING_DIRECTORIES", &package_root)
            .args(["apply", "--ignore-space-change"])
            .arg(&patch),
        "failed to apply the approved simdjson patch",
    );
    let patched_text = fs::read_to_string(&patched_source)
        .unwrap_or_else(|error| panic!("failed to verify patched simdjson source: {error}"));
    require_bigint_branch_parity(
        &patched_text,
        "too-many-digits",
        PATCHED_BIGINT_TOO_MANY_DIGITS,
        UNGUARDED_BIGINT_TOO_MANY_DIGITS,
    );
    require_bigint_branch_parity(
        &patched_text,
        "negative-overflow",
        PATCHED_BIGINT_NEGATIVE_OVERFLOW,
        UNGUARDED_BIGINT_NEGATIVE_OVERFLOW,
    );
    require_bigint_branch_parity(
        &patched_text,
        "positive-overflow",
        PATCHED_BIGINT_POSITIVE_OVERFLOW,
        UNGUARDED_BIGINT_POSITIVE_OVERFLOW,
    );

    patched_source
}

fn require_file(path: &str) {
    assert!(
        Path::new(path).is_file(),
        "required native input is missing: {path}"
    );
}

fn main() {
    let simdjson_header = "third_party/simdjson/singleheader/simdjson.h";
    let simdjson_source = "third_party/simdjson/singleheader/simdjson.cpp";
    let bridge_header = "src/native/simdjson_bridge.h";
    let bridge_source = "src/native/simdjson_bridge.cpp";
    let telemetry_header = "src/native/native_alloc_telemetry.h";
    let telemetry_source = "src/native/native_alloc_telemetry.cpp";

    for path in [
        "build.rs",
        SIMDJSON_PATCH,
        bridge_header,
        bridge_source,
        telemetry_header,
        telemetry_source,
        simdjson_header,
        simdjson_source,
    ] {
        println!("cargo:rerun-if-changed={path}");
        require_file(path);
    }

    let patched_simdjson_source = patched_simdjson_source(simdjson_source);
    let target = env::var("TARGET").expect("TARGET must be set by Cargo");

    // glibc only; musl targets need a different libstdc++/libc++ story and
    // are out of scope for the current ABI v0.1 build contract.
    if target.contains("linux-gnu") {
        println!("cargo:rustc-link-arg-cdylib=-static-libstdc++");
        println!("cargo:rustc-link-arg-cdylib=-static-libgcc");
        println!("cargo:rustc-link-arg-cdylib=-Wl,--exclude-libs,ALL");
    }

    cc::Build::new()
        .cpp(true)
        .std("c++17")
        .define("SIMDJSON_IMPLEMENTATION_FALLBACK", "1")
        .include("third_party/simdjson/singleheader")
        .include("src/native")
        .file(patched_simdjson_source)
        .file(bridge_source)
        .file(telemetry_source)
        .compile("pure_simdjson_native");
}
