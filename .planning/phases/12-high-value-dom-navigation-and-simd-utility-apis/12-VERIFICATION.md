---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
verified: "2026-07-31T18:05:00Z"
status: passed
score: "55/55 must-haves verified"
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "53/55 must-haves verified"
  gaps_closed:
    - "Focused Rust and header checks cover both utility exports and their exact signatures (12-04 T5 / minify BUFFER_TOO_SMALL out_written contract)"
    - "Loader tests reject ABI 1.2, bind and cache only a complete ABI 1.3 surface, fail closed on an incomplete ABI 1.3 artifact, and preserve later additive ABI 1.4 compatibility (12-11 T5 / allocator-telemetry mandatory binding)"
  gaps_remaining: []
  regressions: []
---

# Phase 12: High-value DOM navigation and SIMD utility APIs Verification Report

**Phase Goal:** Expose the mature, high-value parts of simdjson's DOM and implementation APIs as thin Go wrappers: standards-based navigation, indexed/container helpers, wildcard path selection, fast minification, and standalone UTF-8 validation.
**Verified:** 2026-07-31T18:05:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure plans 12-12 and 12-13

## Goal Achievement

Both blockers recorded in the prior verification (2026-07-31T12:50:20Z) are closed and independently re-proven with adversarial, fresh execution — not by trusting the gap-closure SUMMARYs. The minify `BUFFER_TOO_SMALL` out-parameter contract now round-trips the required capacity, and the ABI 1.3 loader now fails closed on either missing allocator-telemetry symbol before cache installation. All 55 plan must-have truths and all 4 roadmap success criteria are verified. The public navigation, wildcard, container, minify, and UTF-8 surfaces are implemented, wired to real upstream simdjson calls, and pass focused, full-suite, race-enabled, contract, documentation, sanitizer, and dynamic-library-smoke gates.

### Roadmap Success Criteria

| # | Roadmap truth | Status | Evidence |
|---|---|---|---|
| 1 | `Element.AtPointer` follows RFC 6901 and `Element.AtPath` follows the simdjson dot/index subset with typed traversal errors. | ✓ VERIFIED | Unchanged since prior pass; re-ran `cargo test --test rust_shim_navigation` (19/19 passed) and `go test . -race -run TestMinify` unrelated packages unaffected; navigation code paths untouched by gap-closure commits. |
| 2 | Wildcard queries return ordered, document-tied views without claiming full RFC 9535 support. | ✓ VERIFIED | Same as above — `rust_shim_navigation.rs` wildcard tests pass; code untouched by 12-12/12-13. |
| 3 | Arrays expose indexed access and length; arrays and objects expose size without Go iteration. | ✓ VERIFIED | `Array.At`/`Len`/`Object.Size` code untouched by gap closure; full `go test ./... -race -count=1` green. |
| 4 | `Minify` and `ValidateUTF8` are allocation-conscious SIMD wrappers with overlap, empty, malformed, and cross-platform tests. | ✓ VERIFIED | Gap 1 closed: direct `ctypes` call against the freshly rebuilt `target/release/libpure_simdjson.dylib` confirms `rc=6 written=8 dst_unchanged=True` (source `{"a": 1}`, 8 bytes, 7-byte dest) instead of the prior `written=SIZE_MAX`. Success path (`rc=0 written=7 out=b'{"a":1}'`), same-start aliasing, and partial-overlap rejection (`rc=1`) all verified in the same adversarial run — no regression to 12-04 T1/T2. |

### Plan Must-Have Truths

Every PLAN frontmatter truth was re-checked. The two previously-failed truths were re-verified with fresh, adversarial execution (not the passing test suite alone); the 53 previously-passing truths were regression-checked via full-suite re-runs of every gate the gap-closure commits could plausibly have affected.

| Plan | Truth | Status | Evidence |
|---|---|---|---|
| 12-01 T1–T5 | ABI 1.3 status codes, AtPointer/AtPath delegation, header contract, exact signatures. | ✓ VERIFIED | Unchanged; `cargo test --test rust_shim_navigation` and `make verify-contract` re-run clean. |
| 12-02 T1–T5 | Native Array.At/Len/Object.Size behavior, bounds mapping, symbol contract. | ✓ VERIFIED | Unchanged; navigation suite re-run 19/19 passed. |
| 12-03 T1–T7 | Wildcard scratch-to-owned-view transport, ordering, allocation ledger, free discipline, lifetime/serialization, null/count boundaries, signatures. | ✓ VERIFIED | Unchanged; same suite re-run clean. |
| 12-04 T1 | Native minify checks capacity before upstream entry. | ✓ VERIFIED | Re-run `cargo test --test rust_shim_minify`; adversarial `ctypes` Case 1 confirms the check still triggers (`rc=6`) before any output byte is written (`dst_unchanged=True`). |
| 12-04 T2 | Only same-start aliasing or disjoint buffers are accepted. | ✓ VERIFIED | Adversarial Case 3 (same-start alias, `rc=0`) and Case 4 (partial overlap starting mid-buffer, `rc=1`, rejected) both confirmed directly against the built dylib — no regression from the Gap 1 fix. |
| 12-04 T3 | Utilities apply fallback/selection-lock ordering and scan after mutex release. | ✓ VERIFIED | `cargo test --test rust_shim_utility_lock` re-run: 1/1 passed. |
| 12-04 T4 | Minify is documented as non-validating except unclosed strings. | ✓ VERIFIED | `minify_unclosed_string_returns_invalid_json` re-run passed; doc comment unchanged in intent (only extended, see header regen check below). |
| 12-04 T5 | Focused Rust/header checks cover both utility exports and exact signatures. | ✓ VERIFIED (was FAILED) | `tests/rust_shim_minify.rs::minify_undersized_dst_returns_buffer_too_small_before_writing` now contains `assert_eq!(written, input.len());` (confirmed by direct file read) and passes. `tests/smoke/ffi_export_surface.c` gained a new block asserting `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL` + `undersized_written == sizeof(minify_source) - 1` against the real dylib; `run_native_smoke.sh` printed `ffi export surface smoke passed`. Independent `ctypes` adversarial call also confirms `written == src_len` (8) on status 6. |
| 12-05 T1–T3 | Go ABI/status constants, bootstrap source identity, bootstrap docs. | ✓ VERIFIED | Unaffected by gap closure; `make verify-docs` and Go ABI tests re-run clean. |
| 12-06 T1–T5 | Public AtPointer/AtPath typed errors, AtPathAll semantics, sentinel distinctness, ErrBufferTooSmall mapping. | ✓ VERIFIED | Unaffected; `go test ./... -race -count=1` green. |
| 12-07 T1 | Minify allocates; MinifyInto allows exact alias/disjoint and rejects partial overlap pre-FFI. | ✓ VERIFIED | Re-ran `go test . -race -run TestMinify` (`TestMinifyInto_Overlap`, `_Disjoint`, `_PartialOverlap` all pass) — Go-level pre-FFI rejection is independent of and unaffected by the Rust-side capacity fix. |
| 12-07 T2 | Short Go destinations reject pre-FFI with ErrBufferTooSmall. | ✓ VERIFIED | `TestMinifyInto_UndersizedDst` re-run passed; Go's `MinifyInto` still checks length before calling FFI, so it never even exercises the fixed BUFFER_TOO_SMALL path — confirmed no regression. |
| 12-07 T3–T5 | Utilities operate on caller-owned slices; ValidateUTF8 tri-state; kernel-state mirroring. | ✓ VERIFIED | `TestMinifyAutomaticFallbackRejected`, `TestMinifyLocksKernelSelection`, etc. re-run passed (subprocess-isolated, ~1s each as expected). |
| 12-08 T1–T5 | Array.At shape, int/negative rejection, Len/Size dual methods, out-of-range mapping, doc disclosures. | ✓ VERIFIED | Unaffected by gap closure; full Go suite green. |
| 12-09 T1 | Packed native/header ABI is `0x00010003`. | ✓ VERIFIED | `make verify-contract` re-run: header diff against freshly-`cbindgen`-regenerated output is empty; ABI constant unchanged. |
| 12-09 T2 | Status values 11/12 are append-only/distinct. | ✓ VERIFIED | Unaffected; contract gate re-run clean. |
| 12-09 T3 | Rust, generated C, C assertions, and Python fixtures agree. | ✓ VERIFIED | `make verify-contract` re-run: `tests/abi/test_check_header.py` (26 tests), `test_check_bootstrap_abi_state.py` (14 tests), `test_release_workflow_contracts.py` (17 tests) all pass; `cc -Iinclude tests/abi/handle_layout.c -c` compiles clean. Confirms the 12-12 header regeneration (extending `pure_simdjson_minify`'s doc comment) introduced **only** two added comment lines — verified via `git show 0b192c2 -- include/pure_simdjson.h`, no signature/struct/enum change. |
| 12-09 T4 | Normative table describes statuses without renumbering older values. | ✓ VERIFIED | `make verify-docs` re-run: exit 0. |
| 12-10 T1–T6 | D-14 probe, phase-2 workflow triggers, native smoke, five-platform branch, loader-contract docs, workflow-contract tests. | ✓ VERIFIED | Re-ran `bash scripts/ci/verify_minify_buffer_safety.sh` → `kernels=arm64,fallback total=24 runs=3`; re-ran `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` → `ffi export surface smoke passed`. Neither script nor workflow file was touched by 12-12/12-13. |
| 12-11 T1 | All nine Phase 12 symbols are mandatory, never optional. | ✓ VERIFIED | Unaffected — these nine were already mandatory prior to gap closure; still present in `bindings.go`'s `symbols` slice. |
| 12-11 T2 | Wrappers preserve ordering/KeepAlive/copies and wildcard arrays free once. | ✓ VERIFIED | Unaffected by gap closure; binding tests re-run clean. |
| 12-11 T3 | Binding tests exercise all nine Phase 12 symbols and boundary states. | ✓ VERIFIED | `go test ./internal/ffi/... -race -count=1` re-run: all pass, no regression from the two newly-mandatory symbols added alongside them. |
| 12-11 T4 | Release policy accepts 0.2.0-dev/ABI 1.3 and rejects 0.1.7. | ✓ VERIFIED | Unaffected; `test_check_bootstrap_abi_state.py` (14/14) re-run clean. |
| 12-11 T5 | Loader binds/caches only a complete ABI 1.3 surface and fails closed when incomplete. | ✓ VERIFIED (was FAILED) | `internal/ffi/bindings.go`: `pure_simdjson_native_alloc_stats_reset`/`_snapshot` are now in the mandatory `symbols` slice (confirmed by direct file read); `grep -c registerOptionalFuncWithRegistrar internal/ffi/bindings.go` shows only the `psdj_internal_materialize_build` call remains optional. `abi13RequiredSymbols` (bindings_test.go) and `abi13MandatoryFixtureSymbols` (library_loading_test.go) both include the two symbols at the correct position. `TestBindRequiresNativeAllocStatsSymbols`, `TestABI13IncompleteMissingAllocStatsResetFailsClosed`, `TestABI13IncompleteMissingAllocStatsSnapshotFailsClosed` all pass and assert `cachedLibrary == nil` + no `"implementation-name"` event — i.e., failure occurs strictly before cache installation, not merely "binding errors." **Adversarial regression proof:** reverted `internal/ffi/bindings.go` to its pre-fix (optional-registration) content and re-ran the internal/ffi gap tests — both `TestBindRequiresNativeAllocStatsSymbols` subtests and `TestBindLooksUpCompleteABI13Surface` failed as expected (`Bind() error = nil with ... missing`). Separately reverted only the `abi13MandatoryFixtureSymbols` fixture-list hunk in `library_loading_test.go` (keeping the two new test functions) and re-ran — both `TestABI13IncompleteMissingAllocStats{Reset,Snapshot}FailsClosed` failed as expected (`activeLibrary() error = nil, want incomplete ABI 1.3 failure`). Both files were restored to their fixed state afterward (`git status --short` clean). `TestABILaterAdditiveMinorBindsAndCaches` (ABI 1.4 forward compatibility) re-run and still passes — promoting the two symbols to mandatory does not reject a future additive ABI. |

**Score:** 55/55 plan truths verified

### Required Artifacts

| Artifact/group | Expected | Status | Details |
|---|---|---|---|
| `element.go`, `errors.go`, navigation/indexed Go tests | Public DOM APIs, taxonomy, lifecycle, docs | ✓ VERIFIED | Unchanged since prior pass; full Go suite green. |
| `src/native/simdjson_bridge.cpp`, `src/runtime/registry.rs`, `tests/rust_shim_navigation.rs` | Upstream navigation/container bridge, tracked wildcard transport | ✓ VERIFIED | Untouched by gap closure (confirmed via `git diff --stat` over both gap-closure commit ranges — zero C++ files modified); tests re-run clean. |
| `minify.go`, `utf8.go`, `kernel.go`, Go utility tests | Public Go SIMD utilities and kernel-state handling | ✓ VERIFIED | `TestMinify*` suite re-run passed; Go-level pre-FFI checks are independent of the Rust fix and remain correct. |
| `src/lib.rs`, `src/runtime/mod.rs` | Public native utility export and thin runtime handoff | ✓ VERIFIED (was PARTIAL) | `native_minify` now returns `(pure_simdjson_error_code_t, usize)`; `pure_simdjson_minify` writes `out_written` via `ptr::write` guarded by `rc == err_ok() || rc == err_buffer_too_small()`, never through `write_out`. Confirmed by direct source read and adversarial `ctypes` call. |
| `tests/rust_shim_minify.rs` | Raw utility boundary coverage | ✓ VERIFIED (was PARTIAL) | `assert_eq!(written, input.len());` present and passing. |
| `tests/smoke/ffi_export_surface.c` | Dynamic C smoke coverage of status+capacity pair | ✓ VERIFIED (new) | New undersized-destination block asserts both status 6 and `out_written == src_len`; `run_native_smoke.sh` passed against the real dylib. |
| `internal/ffi/bindings.go` | Complete mandatory ABI binding before cache | ✓ VERIFIED (was PARTIAL) | Both allocator-telemetry symbols moved into the mandatory `symbols` slice; only `psdj_internal_materialize_build` remains optional. |
| `internal/ffi/bindings_test.go`, `library_loading_test.go` | Complete/incomplete ABI 1.3 fail-closed tests | ✓ VERIFIED (was PARTIAL) | Both fixture lists include the two symbols at the correct position; new per-symbol missing-binding and fail-closed-before-cache regressions pass, and were adversarially proven to fail against pre-fix code. |
| `include/pure_simdjson.h` | ABI-stable, regenerated only for the new doc comment | ✓ VERIFIED | `make verify-contract`'s header diff is empty against a fresh `cbindgen` run; `git show 0b192c2` confirms only 2 added comment lines, no signature/struct/enum change — highest-risk deviation in the gap closure confirmed benign. |
| ABI/header sources and tests, bootstrap/release policy, durable probe/smoke/workflows | ABI 1.3 numeric/signature contract, unreleased identity, cross-platform/sanitizer gates | ✓ VERIFIED | All re-run clean (`make verify-contract`, `make verify-docs`, `verify_minify_buffer_safety.sh`, `run_native_smoke.sh`); none of these files were touched by 12-12/12-13. |

All other declared artifacts (`internal/ffi/types*.go`, `internal/bootstrap/*`, `docs/bootstrap.md`, `docs/ffi-contract.md`, `element_*_test.go`, `minify_test.go`, `utf8_test.go`, policy scripts, workflow-contract tests) continue to pass existence, substance, and wiring checks; none were modified by the gap-closure plans.

### Key Link Verification

`gsd-sdk verify.key-links` again could not parse the PLANs' annotated `from` strings and would emit false "Source file not found" results; each link was traced manually, as in the prior verification.

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Go navigation/container methods | `internal/ffi` bindings | Typed wrapper calls + `runtime.KeepAlive` | ✓ WIRED | Unaffected by gap closure; re-run clean. |
| Go utility functions | active library + kernel state + bindings | Preflight, `kernelMu`, binding call, status mapping | ✓ WIRED | `TestMinify*` and `TestValidateUTF8*` re-run clean. |
| Rust public navigation exports | registry | `runtime::registry::*` calls inside `ffi_wrap` | ✓ WIRED | Unaffected; navigation suite re-run clean. |
| `src/lib.rs pure_simdjson_minify` | `src/runtime/mod.rs native_minify` | `let (rc, written) = runtime::native_minify(...)` tuple destructure | ✓ WIRED (was PARTIAL) | Confirmed by direct source read: no `Result::Ok/Err` match remains; tuple destructure feeds a guarded `ptr::write`. |
| Minify C++ bridge | Rust runtime/public export | status + required-capacity handoff | ✓ WIRED (was PARTIAL) | `written` now travels for both `OK` and `BUFFER_TOO_SMALL`; adversarial `ctypes` call confirms `written=8` on status 6, `written=7` on status 0. |
| ABI source | cbindgen config/header/checkers | generated header + contract gate | ✓ WIRED | Header diff empty; contract gate re-run clean. |
| Loader version probe | mandatory binding | probe, bind, then cache | ✓ WIRED (was PARTIAL) | Both allocator-telemetry symbols now sit in the mandatory loop; missing either fails `Bind()`/`activeLibraryWithOps` before any `Bindings` value or cache write, confirmed by direct test execution and by adversarial revert-and-rerun. |
| Release workflow | native export smoke | existing `run_native_smoke.sh` path | ✓ WIRED | Fresh dynamic smoke passed twice (once after a forced rebuild) on darwin/arm64. |

### Data-Flow Trace (Level 4)

| Artifact/flow | Data | Source | Produces real data | Status |
|---|---|---|---|---|
| `Element.AtPointer` / `AtPath` | Result `ValueView` | Parsed native DOM → upstream navigation → registered descendant | Yes | ✓ FLOWING |
| `Element.AtPathAll` | Ordered `[]Element` | Upstream wildcard matches → doc scratch indices → Rust-owned views → Go copy | Yes | ✓ FLOWING |
| `Array.At`, `Len`, `Object.Size` | Element/count | Native DOM tape | Yes | ✓ FLOWING |
| `Minify` success | Compacted bytes + written count | Upstream SIMD minifier | Yes | ✓ FLOWING |
| `Minify` short-destination error | Required capacity | C++ sets `src_len` → Rust tuple → Go export via guarded `ptr::write` | Yes (was No) | ✓ FLOWING (was DISCONNECTED) — adversarial `ctypes` proof: `written=8` on `rc=6` |
| `ValidateUTF8` | Validity bit | Upstream SIMD validator | Yes | ✓ FLOWING |
| ABI 1.3 loader gate | Required symbol set | Header/normative ABI → Go binding table (now complete) → cache | Yes (was Incomplete) | ✓ FLOWING (was HOLLOW) — both allocator-telemetry symbols mandatory; missing either blocks cache write, confirmed by adversarial revert |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Release library builds (forced rebuild via `touch src/lib.rs`) | `cargo build --release` | Exit 0, no warnings | ✓ PASS |
| Native navigation/wildcard/minify/UTF-8 focused suites | `cargo test --test rust_shim_navigation --test rust_shim_minify --test rust_shim_utility_lock -- --test-threads=1` | 31 passed, 0 failed | ✓ PASS |
| Complete Rust workspace test suite | `cargo test --release` | All 15 test binaries, 0 failed (regression sweep across the whole blast radius, not just the touched files) | ✓ PASS |
| Complete Go tree under race detector | `go test ./... -race -count=1` | All 4 packages passed | ✓ PASS |
| Go build across whole module | `go build ./...` | Exit 0 | ✓ PASS |
| ABI/header contract | `make verify-contract` | All Rust tests, header diff (empty), C layout, and Python ABI fixtures (26+14+17 tests) passed | ✓ PASS |
| Documentation contract | `make verify-docs` | Exit 0 | ✓ PASS |
| D-14 alias safety (ASan/UBSan) | `bash scripts/ci/verify_minify_buffer_safety.sh` | `kernels=arm64,fallback total=24 runs=3` | ✓ PASS |
| Dynamic ABI surface smoke (run twice, once post-rebuild) | `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` | `ffi export surface smoke passed` (both runs) | ✓ PASS |
| **Adversarial: minify status-6 required capacity** | Direct `ctypes` call to built `pure_simdjson_minify`, `SIZE_MAX` sentinel, `dst_len=7 < src_len=8` | `rc=6 written=8 dst_unchanged=True expected_written=8` | ✓ PASS (was FAIL) |
| **Adversarial: minify success path unaffected** | Same `ctypes` harness, `dst_len=8` | `rc=0 written=7 out=b'{"a":1}'` | ✓ PASS |
| **Adversarial: same-start aliasing still accepted** | Same harness, `dst_ptr == src_ptr` | `rc=0 written=7` | ✓ PASS |
| **Adversarial: partial overlap still rejected** | Same harness, `dst_ptr = src_ptr + 2` (overlapping, different start) | `rc=1` (rejected, not silently truncated) | ✓ PASS |
| **Adversarial: loader gap-2 regression proof (missing symbols)** | Revert `internal/ffi/bindings.go` to pre-fix optional registration; run `TestBindRequiresNativeAllocStatsSymbols`, `TestBindLooksUpCompleteABI13Surface` | Both fail as expected (`Bind() error = nil with ... missing`) | ✓ PASS (regression genuinely catches pre-fix bug) |
| **Adversarial: loader gap-2 regression proof (fixture)** | Revert only `abi13MandatoryFixtureSymbols` fixture-list hunk in `library_loading_test.go` (keep new tests); run `TestABI13IncompleteMissingAllocStats{Reset,Snapshot}FailsClosed` | Both fail as expected (`activeLibrary() error = nil, want incomplete ABI 1.3 failure`) | ✓ PASS (fixture, not just new test code, was necessary) |
| Header-required vs loader-mandatory set after fix | Direct file reads of `bindings.go`, `bindings_test.go`, `library_loading_test.go` | Both symbols present in all three mandatory lists at matching positions | ✓ PASS (was FAIL) |
| Scope discipline: 12-12 touched zero C++ files | `git diff --stat` over `ab31d76..c870f3f` | `src/lib.rs`, `src/runtime/mod.rs`, `include/pure_simdjson.h`, `tests/rust_shim_minify.rs`, `tests/smoke/ffi_export_surface.c`, planning docs — no `.cpp`/`.h` under `src/native/` | ✓ PASS |
| Scope discipline: 12-13 touched zero Rust/C/C++ files | `git diff --stat` over `910d09a..050f960` | `internal/ffi/bindings.go`, `internal/ffi/bindings_test.go`, `library_loading_test.go`, planning docs only | ✓ PASS |

All adversarial checks were re-run against a library forced to rebuild from current source (`touch src/lib.rs && cargo build --release`), and the working tree was confirmed clean (`git status --short`) after every revert-and-restore cycle used for regression proof.

### Probe Execution

No PLAN/SUMMARY declares a conventional `probe-*.sh`, and none exists under `scripts/**/tests`. The phase's probe-equivalent D-14 verifier was re-run and again passed three sanitizer executions, as shown above.

### Requirements Coverage

All six IDs declared across PLAN frontmatter (including 12-12/12-13's `requirements: [UTIL-01]` and `[DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02]`) match the six Phase 12 mappings in `REQUIREMENTS.md`; no orphaned Phase 12 requirement exists.

| Requirement | Source plans | Description | Status | Evidence |
|---|---|---|---|---|
| DOM-01 | 12-01, 12-05, 12-06, 12-09, 12-10, 12-11, 12-13 | RFC 6901 `AtPointer` via upstream navigation | ✓ SATISFIED | Full native/Go chain and RFC/error tests pass; unaffected by gap closure. |
| DOM-02 | 12-01, 12-05, 12-06, 12-09, 12-10, 12-11, 12-13 | Documented simdjson dot/index `AtPath` subset | ✓ SATISFIED | Native delegation, honest docs, and behavior tests pass. |
| DOM-03 | 12-03, 12-05, 12-06, 12-09, 12-10, 12-11, 12-13 | Ordered wildcard `AtPathAll` with document lifetime | ✓ SATISFIED | Ordered/empty/partial/lifetime/free/concurrency paths pass. |
| DOM-04 | 12-02, 12-05, 12-08, 12-09, 12-10, 12-11, 12-13 | Indexed arrays and constant-time container counts | ✓ SATISFIED | Public/native behavior and edge cases pass. |
| UTIL-01 | 12-04, 12-05, 12-06, 12-07, 12-09, 12-10, 12-11, 12-12, 12-13 | Allocation-conscious SIMD minify API | ✓ SATISFIED (was BLOCKED) | `pure_simdjson_minify`'s `BUFFER_TOO_SMALL` out-parameter contract is now upheld end-to-end, proven by a direct adversarial `ctypes` call against the built library, plus strengthened Rust and dynamic C smoke assertions. |
| UTIL-02 | 12-04, 12-05, 12-07, 12-09, 12-10, 12-11, 12-13 | Standalone SIMD UTF-8 validation | ✓ SATISFIED | Valid/invalid/empty/CPU/parse-regression tests pass; unaffected by gap closure. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---:|---|---|---|
| `src/lib.rs` | 960-968, 1006-1014 | Iterator lease allocated before null output is rejected (WR-01) | ⚠ Warning | Predates Phase 12 (`git blame`); explicitly deferred during gap-closure planning per 12-12's scope note. Not touched by 12-12/12-13. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | 5-16 | Contract gate paths omit `cbindgen.toml` and `tests/abi/**` (WR-02) | ⚠ Warning | Explicitly deferred; workflow file untouched by gap closure. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | 41 | `cargo install cbindgen --locked` has no version (WR-03) | ⚠ Warning | Explicitly deferred; workflow file untouched by gap closure. |
| `materializer_fastpath.go` | 217 | `go vet` unsafe-pointer diagnostic | ℹ Info | Predates Phase 12; recorded in `deferred-items.md`; fresh `go vet ./...` still exits 1 with only this single finding, confirmed unchanged. |

No `TBD`, `FIXME`, or `XXX` debt markers were found in any file touched by 12-12 or 12-13 (`src/runtime/mod.rs`, `src/lib.rs`, `tests/rust_shim_minify.rs`, `tests/smoke/ffi_export_surface.c`, `include/pure_simdjson.h`, `internal/ffi/bindings.go`, `internal/ffi/bindings_test.go`, `library_loading_test.go`). Both prior blockers (minify capacity discard, optional allocator-telemetry symbols) are resolved and no longer present in the code. Both prior WR items remain explicit, rationale-documented deferrals rather than new gaps.

### Human Verification Required

None. No deferred `<human-check>` blocks exist in any Phase 12 plan, including 12-12/12-13. Hosted five-platform release evidence remains explicitly owned by Phase 16.

### Deferred Items

Carried forward from the prior verification and gap-closure planning, with explicit rationale — not treated as gaps:

| # | Item | Addressed In | Evidence |
|---|------|--------------|----------|
| 1 | WR-01 rejected-iterator-constructor lease ordering (`src/lib.rs:960-968,1006-1014`) | Deferred, no specific phase claims it yet | Predates Phase 12 per `git blame`; 12-12's plan scope note explicitly excludes it as unrelated-function scope creep. |
| 2 | WR-02 contract workflow trigger paths (`.github/workflows/phase2-rust-shim-smoke.yml:5-16`) | Deferred, no specific phase claims it yet | Explicitly deferred during gap-closure planning; workflow file untouched by 12-12/12-13. |
| 3 | WR-03 unpinned `cbindgen` version (`.github/workflows/phase2-rust-shim-smoke.yml:41`) | Deferred, no specific phase claims it yet | Explicitly deferred during gap-closure planning; workflow file untouched by 12-12/12-13. |
| 4 | Hosted five-platform release evidence | Phase 16 | Phase 16 goal: "Validate the expanded ABI and APIs on all five targets, publish benchmark evidence, and ship a fresh-machine-tested v0.2 release through CI." |
| 5 | `go vet ./...` `materializer_fastpath.go:217` unsafe-pointer diagnostic | Deferred, pre-existing | Recorded in `deferred-items.md`; predates Phase 12; not touched by any Phase 12 plan. |

## Gaps Summary

No gaps remain. Both blockers from the initial verification pass (2026-07-31T12:50:20Z) — the minify `BUFFER_TOO_SMALL` capacity discard and the ABI 1.3 loader's optional allocator-telemetry symbols — are closed and independently re-proven with fresh, adversarial evidence gathered directly against the built artifacts and reverted/restored source, not by trusting the gap-closure plans' SUMMARYs or their own test suites in isolation. All 55 plan must-have truths, all 4 roadmap success criteria, and all 6 Phase 12 requirements (DOM-01 through UTIL-02) are verified. Scope discipline was confirmed for both gap-closure plans (12-12 touched zero C++ files; 12-13 touched zero Rust/C/C++ files), and the header regeneration in 12-12 was confirmed to be doc-comment-only with no ABI-affecting change. Phase 12 achieves its stated goal and is ready to proceed.

---

_Verified: 2026-07-31T18:05:00Z_
_Verifier: the agent (gsd-verifier)_
