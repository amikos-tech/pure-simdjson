use std::{env, path::Path};

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

    for path in [
        "build.rs",
        bridge_header,
        bridge_source,
        simdjson_header,
        simdjson_source,
    ] {
        println!("cargo:rerun-if-changed={path}");
        require_file(path);
    }

    let target = env::var("TARGET").expect("TARGET must be set by Cargo");

    if target.contains("linux-gnu") {
        println!("cargo:rustc-link-arg-cdylib=-static-libstdc++");
        println!("cargo:rustc-link-arg-cdylib=-static-libgcc");
    }

    cc::Build::new()
        .cpp(true)
        .std("c++17")
        .include("third_party/simdjson/singleheader")
        .include("src/native")
        .file(simdjson_source)
        .file(bridge_source)
        .compile("pure_simdjson_native");
}
