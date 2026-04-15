---
phase: 02-rust-shim-minimal-parse-path
verified: 2026-04-15T13:54:19Z
status: passed
score: 15/15 must-haves verified
overrides_applied: 0
---

# Phase 2: Rust Shim + Minimal Parse Path Verification Report

**Phase Goal:** A buildable Rust cdylib + staticlib that compiles vendored simdjson via the `cc` crate and exposes the smallest end-to-end parse path. Proves the FFI contract holds in code on at least one platform.
**Verified:** 2026-04-15T13:54:19Z
**Status:** passed
**Re-verification:** No - initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Release builds produce the native library artifacts required by Phase 2 on the verified platforms. | ✓ VERIFIED | `cargo build --release` succeeded locally and produced `target/release/libpure_simdjson.a` plus `target/release/libpure_simdjson.dylib`; GitHub Actions run `24457845948` shows `linux-smoke`, `windows-smoke`, and `darwin-build-only` all completed with `success`. |
| 2 | A C-language smoke test loads the library, runs `parser_new -> parser_parse -> doc_root -> element_type -> element_get_int64` on literal `42`, and gets back `42`. | ✓ VERIFIED | `make phase2-smoke-linux` built `tests/smoke/minimal_parse.c` against `include/pure_simdjson.h` and printed `phase2 smoke passed`. |
| 3 | `get_abi_version()` returns the ABI value committed in Phase 1. | ✓ VERIFIED | `tests/rust_shim_minimal.rs` covers `get_abi_version_returns_phase1_constant`; the public smoke harness also compares the returned value against `PURE_SIMDJSON_ABI_VERSION`. |
| 4 | The committed header still matches the cbindgen-generated header byte-for-byte. | ✓ VERIFIED | `make verify-contract` regenerated the header, diffed it against `include/pure_simdjson.h`, and passed the ABI contract checks. |
| 5 | Forcing the `fallback` kernel makes `parser_new` return `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED`. | ✓ VERIFIED | `tests/rust_shim_fallback_gate.rs` exercises both `parser_new_rejects_fallback_without_bypass` and `parser_new_allows_fallback_with_hidden_bypass`; both passed under `cargo test`. |
| 6 | The crate builds vendored simdjson `v4.6.1` from `third_party/simdjson` through `build.rs` and the `cc` crate. | ✓ VERIFIED | `build.rs` compiles `third_party/simdjson/singleheader/simdjson.cpp` with `cc::Build::cpp(true).std("c++17")`; `git -C third_party/simdjson describe --tags --exact-match` returned `v4.6.1`. |
| 7 | No C++ exception can cross the native bridge into Rust. | ✓ VERIFIED | `src/native/simdjson_bridge.h` marks bridge functions `noexcept`; `src/native/simdjson_bridge.cpp` wraps every entry point in `try/catch (...)`; `psimdjson_test_force_cpp_exception_returns_err_cpp_exception` passed. |
| 8 | The build keeps simdjson runtime dispatch intact and does not inject manual kernel-selection flags. | ✓ VERIFIED | `build.rs` only adds GNU Linux static runtime link args and never sets `-march=native`, `/MT`, `/MD`, or cmake-based overrides. |
| 9 | The public ABI uses a real generation-checked parser/doc lifecycle rather than a temporary pointer shim. | ✓ VERIFIED | `src/runtime/registry.rs` implements packed handles, `ParserState::{Idle, Busy}`, parser-busy enforcement, stale-handle rejection, and doc-owned parser release semantics; lifecycle tests passed. |
| 10 | Every public export is mechanically wrapped so Rust panics cannot cross the C ABI boundary. | ✓ VERIFIED | `src/lib.rs` contains 24 public `pure_simdjson_*` exports and 24 matching `ffi_wrap("pure_simdjson_...")` calls. |
| 11 | The minimal real path and diagnostics spine work end-to-end with parser busy/free/stale-handle enforcement. | ✓ VERIFIED | `tests/rust_shim_minimal.rs` passed for literal `42`, invalid JSON diagnostics, parser busy on second parse, `parser_free` while doc live, stale doc/view rejection, and double-free rejection. |
| 12 | `element_type` maps all Phase 2 root discriminants to the public `PURE_SIMDJSON_VALUE_KIND_*` constants. | ✓ VERIFIED | `element_type_maps_phase2_root_literals` passed for `null`, `true`, `-42`, `42`, `18446744073709551615`, `1.5`, `"x"`, `[1]`, and `{"k":1}`. |
| 13 | The deterministic C++ exception test hook converts a thrown C++ exception into `PURE_SIMDJSON_ERR_CPP_EXCEPTION`. | ✓ VERIFIED | `src/native/simdjson_bridge.cpp` exposes `psimdjson_test_force_cpp_exception`; the Rust integration test confirmed it returns `PURE_SIMDJSON_ERR_CPP_EXCEPTION`. |
| 14 | Linux and Windows have concrete smoke paths, Windows was observed green, and Darwin has a compile-only verification path. | ✓ VERIFIED | `tests/smoke/README.md` documents Linux and MSVC commands; `.github/workflows/phase2-rust-shim-smoke.yml` defines the three jobs; `gh run view 24457845948 --json jobs` shows `linux-smoke`, `windows-smoke`, and `darwin-build-only` all concluded `success`. |
| 15 | All SHIM-06 exports are present in the public surface, with non-minimal accessors retained as explicit stubs. | ✓ VERIFIED | `include/pure_simdjson.h` contains the full Phase 2 export list; `src/lib.rs` keeps non-minimal accessors and iterators as deliberate `phase1_contract_stub(...)` implementations while the minimal path is real. |

**Score:** 15/15 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `Cargo.toml` | Crate outputs include Phase 2 ship targets and native build dependency | ✓ VERIFIED | Declares `cdylib` and `staticlib` outputs, plus `rlib` for test linkage; `[build-dependencies]` includes `cc`. |
| `build.rs` | Deterministic simdjson + bridge build with exact native inputs | ✓ VERIFIED | Compiles only `third_party/simdjson/singleheader/simdjson.cpp` and `src/native/simdjson_bridge.cpp`, adds rerun guards, sets `c++17`, and uses GNU Linux static runtime link args only on `linux-gnu`. |
| `.gitmodules` | Vendored simdjson recorded as a submodule | ✓ VERIFIED | Contains the `third_party/simdjson` submodule entry. |
| `third_party/simdjson` | Pinned vendored simdjson checkout | ✓ VERIFIED | Exact tag probe returned `v4.6.1`. |
| `src/native/simdjson_bridge.h` | Narrow non-throwing bridge declarations | ✓ VERIFIED | Declares the minimal `psimdjson_*` bridge with `noexcept` entry points only. |
| `src/native/simdjson_bridge.cpp` | Exception-containing bridge implementation for the minimal parse path | ✓ VERIFIED | Implements implementation-name helpers, padding, parser/doc helpers, diagnostics, root/type/int64, and the forced-exception test hook. |
| `src/runtime/mod.rs` | Rust-side bridge bindings and Phase 2 runtime helpers | ✓ VERIFIED | Binds to `psimdjson_*`, exposes implementation-name and padding helpers, and includes the hidden fallback selectors and exception test hook. |
| `src/runtime/registry.rs` | Generation-checked parser/doc registry with explicit lifecycle states | ✓ VERIFIED | Stores packed handles, tracks `Idle` vs `Busy`, retains padded input buffers, and validates doc-root views before native access. |
| `src/lib.rs` | Public exports and panic containment | ✓ VERIFIED | Exposes the full Phase 2 ABI surface, routes all exports through `ffi_wrap`, and wires the minimal real path while leaving later accessors as explicit stubs. |
| `include/pure_simdjson.h` | Committed public ABI header | ✓ VERIFIED | Header matches the generated output and contains the full Phase 2 export surface. |
| `tests/rust_shim_minimal.rs` | Lifecycle, diagnostics, padding, and minimal-path tests | ✓ VERIFIED | Covers ABI version, implementation name, literal `42`, root kind mapping, diagnostics, lifecycle edges, and forced C++ exception handling. |
| `tests/rust_shim_fallback_gate.rs` | Deterministic fallback CPU gate coverage | ✓ VERIFIED | Serializes env mutation and proves both rejection and bypass behavior. |
| `tests/smoke/minimal_parse.c` | Public-header C smoke harness | ✓ VERIFIED | Exercises the committed header against the built library for happy-path parse, cleanup, re-parse, and invalid-JSON diagnostics. |
| `tests/smoke/README.md` | Exact Linux and MSVC smoke commands | ✓ VERIFIED | Documents local Linux and Windows/MSVC compile and run commands plus export checks. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | Dedicated Linux/Windows smoke workflow with Darwin build-only job | ✓ VERIFIED | Defines only `linux-smoke`, `windows-smoke`, and `darwin-build-only`, and the observed run completed successfully. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `build.rs` | `third_party/simdjson/singleheader/simdjson.cpp` | exact `cc` compile input | ✓ VERIFIED | `build.rs` includes the exact amalgamation path in `.file(simdjson_source)`. |
| `src/native/simdjson_bridge.h` | `src/native/simdjson_bridge.cpp` | matching `psimdjson_` bridge symbols | ✓ VERIFIED | Header declarations and C++ definitions match for the minimal bridge surface. |
| `src/lib.rs` | `src/runtime/registry.rs` | public exports delegate to the generation-checked runtime | ✓ VERIFIED | `pure_simdjson_parser_new`, `parser_parse`, `doc_free`, `doc_root`, `element_type`, and `element_get_int64` all call into `runtime::registry`. |
| `src/runtime/registry.rs` | `src/runtime/mod.rs` and `src/native/simdjson_bridge.cpp` | native parse/type/int64 helpers | ✓ VERIFIED | `registry.rs` calls `super::native_parser_parse`, `native_element_type`, and `native_element_get_int64`, which bind to the `psimdjson_*` bridge. |
| `tests/rust_shim_fallback_gate.rs` | `src/lib.rs` | hidden forced-fallback selector evaluated in `parser_new` | ✓ VERIFIED | Tests set `PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION` and exercise `pure_simdjson_parser_new`, which gates through `reject_fallback_implementation()`. |
| `tests/smoke/minimal_parse.c` | `include/pure_simdjson.h` | compiled harness against the committed header | ✓ VERIFIED | The harness includes `pure_simdjson.h` and compiled cleanly during `make phase2-smoke-linux`. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | `tests/smoke/minimal_parse.c` | linux and MSVC smoke execution | ✓ VERIFIED | The workflow runs `make phase2-smoke-linux` on Linux and compiles `tests\\smoke\\minimal_parse.c` with `cl` on Windows. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | GitHub Actions run `24457845948` | branch push triggered remote smoke/build proof | ✓ VERIFIED | `gh run view 24457845948 --json jobs` confirmed all three jobs completed successfully against branch `gsd/phase-02-rust-shim-minimal-parse-path`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `src/runtime/registry.rs` | `owned_input` | `parser_parse` allocates `vec![0u8; input_len + padding]`, copies the caller bytes, and keeps the padded arena in the live doc entry | Yes | ✓ FLOWING |
| `src/runtime/registry.rs` | `doc_handle`, `root_ptr`, `kind_hint` | `super::native_parser_parse` -> `psimdjson_parser_parse` -> `simdjson::dom::parser::parse_into_document(...).get(root)` -> `doc_root`/`native_element_type` | Yes | ✓ FLOWING |
| `tests/smoke/minimal_parse.c` | `int_value`, `last_error_len`, `copied_len` | Public ABI calls into the built library and consume real parser/doc state and diagnostics, not fixtures or hardcoded values | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Release build succeeds | `cargo build --release` | Exit 0; local build produced `libpure_simdjson.a` and `libpure_simdjson.dylib` | ✓ PASS |
| Contract/header verification succeeds | `make verify-contract` | cbindgen diff, header ABI checks, and handle-layout compile all passed | ✓ PASS |
| Runtime and fallback tests pass | `cargo test` | 23 tests passed: 10 unit, 11 `rust_shim_minimal`, 2 `rust_shim_fallback_gate` | ✓ PASS |
| Public-header smoke harness works on the current host | `make phase2-smoke-linux` | Built `tests/smoke/minimal_parse.c` and printed `phase2 smoke passed` | ✓ PASS |
| Remote Linux/Windows/macOS smoke/build verification exists | `gh run view 24457845948 --json jobs,headSha,headBranch` | `linux-smoke=success`, `windows-smoke=success`, `darwin-build-only=success` on branch `gsd/phase-02-rust-shim-minimal-parse-path` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `SHIM-01` | `02-01`, `02-03` | Crate `pure_simdjson` with `crate-type = ["cdylib", "staticlib"]` | ✓ SATISFIED | `Cargo.toml` includes both ship targets, local release build emitted `.a` and `.dylib`, and GitHub run `24457845948` built the library on Linux/Windows/macOS. |
| `SHIM-02` | `02-01`, `02-03` | `build.rs` compiles vendored simdjson v4.6.1 amalgamation via `cc` (C++17, GNU Linux static runtime args) | ✓ SATISFIED | `build.rs` uses `cc::Build::cpp(true).std("c++17")`, compiles the singleheader amalgamation, and emits `-static-libstdc++` / `-static-libgcc` for `linux-gnu`. |
| `SHIM-03` | `02-01` | simdjson vendored as git submodule at `third_party/simdjson` with pinned commit | ✓ SATISFIED | `.gitmodules` records the submodule and `git -C third_party/simdjson describe --tags --exact-match` returned `v4.6.1`. |
| `SHIM-04` | `02-01` | Runtime kernel dispatch left to simdjson auto-detection; no `-march=native` | ✓ SATISFIED | `build.rs` contains no `-march=native` or manual kernel override flags. |
| `SHIM-05` | `02-02` | `get_implementation_name()` exposes the selected kernel for diagnostics | ✓ SATISFIED | Implementation-name length/copy helpers are wired through the bridge and verified by `implementation_name_round_trip_uses_real_bridge_name`. |
| `SHIM-06` | `02-02`, `02-03` | Export surface exists for the minimal parser/doc path plus deferred accessor stubs | ✓ SATISFIED | `include/pure_simdjson.h` contains the full export list, `src/lib.rs` exports all functions, and both local and remote smoke paths validate the minimal ABI behavior. |
| `SHIM-07` | `02-02` | `parser_new` rejects the `fallback` kernel unless the hidden test bypass is enabled | ✓ SATISFIED | `tests/rust_shim_fallback_gate.rs` passed for both rejection and bypass cases. |

No orphaned Phase 2 requirements were found: the union of `requirements:` across `02-01-PLAN.md`, `02-02-PLAN.md`, and `02-03-PLAN.md` is exactly `SHIM-01..07`, and all seven IDs are mapped to Phase 2 in `.planning/REQUIREMENTS.md`.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `include/pure_simdjson.h` | 176 | Public header comments still say the live parser/doc exports are Phase 1 stubs returning `PURE_SIMDJSON_ERR_INTERNAL` | ⚠️ Warning | The ABI behavior is verified, but the generated public documentation is misleading for consumers reading the header comments. |
| `src/lib.rs` | 310 | Rust doc comments feeding cbindgen were not updated after the Phase 2 runtime landed | ⚠️ Warning | The source-of-truth comments no longer describe the actual runtime behavior for the minimal path. |
| `src/lib.rs` | 551 | Deferred accessors and iterators remain explicit contract stubs | ℹ️ Info | This is intentional Phase 2 scope control and not a blocker for the minimal parse path. |
| `Cargo.toml` | 7 | `rlib` was added alongside the planned ship formats | ℹ️ Info | This deviates from the plan's exact crate-type pin, but it does not block the Phase 2 goal because `cdylib` and `staticlib` still build and ship correctly. |

### Gaps Summary

No blocking gaps found. The phase goal is achieved: the crate builds vendored simdjson through `build.rs`, the real minimal parser/doc path works through the committed public header, the ABI contract diff check passes, the fallback CPU gate is tested, and the smoke proof exists both locally and in a verified GitHub Actions run.

Disconfirmation pass notes:
- The plan's exact `crate-type = ["cdylib", "staticlib"]` pin drifted to include `rlib`, but the underlying shim requirement is still satisfied because the shipped formats remain present and verified.
- The public header/source comments for several now-live exports are stale and should be corrected in a follow-up pass.
- The public C smoke harness still does not cover stale-handle or CPU-unsupported error paths through the C ABI, although those behaviors are covered by Rust integration tests.

---

_Verified: 2026-04-15T13:54:19Z_
_Verifier: Claude (gsd-verifier)_
