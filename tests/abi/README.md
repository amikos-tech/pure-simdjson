# ABI Verification Rules

This directory provides the static gates for the Phase 1 FFI contract. `make verify-contract && make verify-docs` is the required automated check for Plan `01-03`.

| Check | Files / Command | Requirements Covered | Purpose |
| --- | --- | --- | --- |
| Header regeneration diff | `make verify-contract` temp-header diff against `include/pure_simdjson.h` | `FFI-01` | Proves the committed public header still round-trips from the Rust ABI source. |
| Metadata ABI unit tests | `cargo test` (run by `make verify-contract`) | `FFI-01`, `FFI-07` | Pins the Rust ABI version constant and live metadata helper semantics that cannot be expressed as C compile-time assertions. |
| `error-code-outparams` | `python3 tests/abi/check_header.py --rule error-code-outparams include/pure_simdjson.h` | `FFI-02` | Fails if any exported symbol stops returning `pure_simdjson_error_code_t` or starts transporting ABI structs by value. |
| `no-mixed-float-int` | `python3 tests/abi/check_header.py --rule no-mixed-float-int include/pure_simdjson.h` | `FFI-03` | Enforces the no scalar float/int mixing rule that keeps the ABI portable for purego. |
| Layout assertions | `cc -Iinclude tests/abi/handle_layout.c -c` | `FFI-04` | Locks the packed handle split and the fixed 32-byte value/iterator layouts. |
| Contract-doc panic policy grep | `make verify-docs` (`ffi_wrap`, `catch_unwind`, `panic = "abort"`) | `FFI-05` | Ensures the normative contract states the unwind boundary precisely. |
| Contract-doc exception policy grep | `make verify-docs` (`.get(err)`) | `FFI-06` | Ensures the contract requires non-throwing simdjson usage at the Rust/C++ seam. |
| `diag-surface` plus ABI/doc version grep | `python3 tests/abi/check_header.py --rule diag-surface ...` and `make verify-docs` (`^0.1.x`) | `FFI-07` | Locks the ABI handshake, parser/doc handle role names, and advisory diagnostics surface used for compatibility and troubleshooting. |
| `string-copy-ownership` plus doc padding grep | `python3 tests/abi/check_header.py --rule string-copy-ownership ...` and `make verify-docs` (`SIMDJSON_PADDING`) | `FFI-08` | Enforces copy-out string ownership and keeps the copied, padded-input rule visible in the contract. |
| Required section and symbol grep | `make verify-docs` plus `--rule required-symbols` | `DOC-02` | Verifies that the normative Markdown contract exists and stays aligned with the committed ABI surface. |

## Rule Summary

- `required-symbols`: ensures the committed Phase 1 symbol set is exact, with no missing or unexpected `pure_simdjson_*` exports.
- `error-code-outparams`: ensures exported functions keep the typed `pure_simdjson_error_code_t` status return and continue to transport ABI structs by pointer.
- `string-copy-ownership`: ensures string access stays `uint8_t **out_ptr` + `size_t *out_len` with `pure_simdjson_bytes_free`.
- `diag-surface`: ensures ABI version, implementation name, parser/doc handle role names, and bounded parser diagnostics remain part of the public surface.

If a later phase changes the ABI intentionally, this table must be updated in the same change so the requirement trace remains auditable.
