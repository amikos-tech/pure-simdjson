# Phase 2 Native Smoke Harness

The Phase 2 smoke harness compiles a native C executable against the committed public header and the built library, then exercises the minimal ABI proof path:

- `pure_simdjson_get_abi_version`
- `pure_simdjson_parser_new`
- `pure_simdjson_parser_parse`
- `pure_simdjson_doc_root`
- `pure_simdjson_element_type`
- `pure_simdjson_element_get_int64`
- `pure_simdjson_doc_free`
- `pure_simdjson_parser_get_last_error_len`
- `pure_simdjson_parser_copy_last_error`
- `pure_simdjson_parser_free`

It checks the happy path on the literal JSON `42`, verifies document release allows a second parse, and asserts the invalid payload `{"x":}` reports `PURE_SIMDJSON_ERR_INVALID_JSON` with a non-empty diagnostic buffer.

## Linux

```sh
mkdir -p target/phase2-smoke
cargo build --release
cc -Iinclude tests/smoke/minimal_parse.c -Ltarget/release -lpure_simdjson -Wl,-rpath,$PWD/target/release -o target/phase2-smoke/minimal_parse
target/phase2-smoke/minimal_parse
```

Optional export check:

```sh
nm -D --defined-only target/release/libpure_simdjson.so
```

## Windows (MSVC)

Run these from a Visual Studio Developer Command Prompt after `cargo build --release`:

```bat
if not exist target\phase2-smoke mkdir target\phase2-smoke
cl /nologo /Iinclude tests\smoke\minimal_parse.c /link /LIBPATH:target\release pure_simdjson.dll.lib /OUT:target\phase2-smoke\minimal_parse.exe
set PATH=%CD%\target\release;%PATH%
target\phase2-smoke\minimal_parse.exe
```

Optional export check:

```bat
dumpbin /EXPORTS target\release\pure_simdjson.dll
```
