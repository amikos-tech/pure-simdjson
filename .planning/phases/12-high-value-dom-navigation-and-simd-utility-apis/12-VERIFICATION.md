---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
verified: "2026-07-31T12:50:20Z"
status: gaps_found
score: "53/55 must-haves verified"
overrides_applied: 0
gaps:
  - truth: "Focused Rust and header checks cover both utility exports and their exact signatures"
    status: failed
    reason: "The public minify export violates its documented BUFFER_TOO_SMALL out-parameter contract, and the focused Rust test does not assert that out_written receives src_len. A direct call returned status 6 while leaving the caller's sentinel untouched."
    artifacts:
      - path: "src/runtime/mod.rs"
        issue: "native_minify returns Err(rc) on every non-OK status and discards the native written value."
      - path: "src/lib.rs"
        issue: "pure_simdjson_minify writes out_written only in the Ok branch, so status 6 loses the required capacity."
      - path: "tests/rust_shim_minify.rs"
        issue: "The undersized-destination test initializes written to SIZE_MAX but asserts only status and destination immutability."
    missing:
      - "Preserve the native status and written value together across src/runtime/mod.rs."
      - "Write src_len to out_written for PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL in the public export."
      - "Assert the status/required-capacity pair in Rust and dynamic C smoke coverage."
  - truth: "Loader tests reject ABI 1.2, bind and cache only a complete ABI 1.3 surface, fail closed on an incomplete ABI 1.3 artifact, and preserve later additive ABI 1.4 compatibility"
    status: failed
    reason: "The loader treats two earlier public ABI symbols as optional, so an ABI 1.3 library missing either allocator-telemetry export can still bind and be cached despite the complete-earlier-surface rule. The ABI 1.3 fixtures omit both symbols and cannot catch this."
    artifacts:
      - path: "internal/ffi/bindings.go"
        issue: "pure_simdjson_native_alloc_stats_reset and pure_simdjson_native_alloc_stats_snapshot use optional registration after the mandatory symbol loop."
      - path: "internal/ffi/bindings_test.go"
        issue: "abi13RequiredSymbols omits both public allocator-telemetry symbols."
      - path: "library_loading_test.go"
        issue: "abi13MandatoryFixtureSymbols omits both symbols, so the complete/incomplete loader tests model an incomplete surface as complete."
    missing:
      - "Move both public allocator-telemetry exports into the mandatory binding table."
      - "Add both names to ABI 1.3 binding and loader fixtures."
      - "Add one-symbol-missing regressions proving either omission fails before cache installation."
---

# Phase 12: High-value DOM navigation and SIMD utility APIs Verification Report

**Phase Goal:** Expose the mature, high-value parts of simdjson's DOM and implementation APIs as thin Go wrappers: standards-based navigation, indexed/container helpers, wildcard path selection, fast minification, and standalone UTF-8 validation.
**Verified:** 2026-07-31T12:50:20Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

The public navigation, wildcard, container, minify, and UTF-8 happy paths are implemented and pass focused and full race-enabled tests. The phase does not pass goal-backward verification because two fail-closed ABI contracts are observably false: minify drops its required-capacity output on status 6, and the Go loader accepts an ABI 1.3 library missing two earlier public symbols.

### Roadmap Success Criteria

| # | Roadmap truth | Status | Evidence |
|---|---|---|---|
| 1 | `Element.AtPointer` follows RFC 6901 and `Element.AtPath` follows the simdjson dot/index subset with typed traversal errors. | ✓ VERIFIED | Public methods in `element.go`; native calls delegate to `element::at_pointer`/`element::at_path`; focused Rust and Go tests pass. |
| 2 | Wildcard queries return ordered, document-tied views without claiming full RFC 9535 support. | ✓ VERIFIED | Scratch-vector-to-owned-view flow is wired; lifetime/free/concurrency tests pass; Go docs explicitly disclaim RFC 9535. |
| 3 | Arrays expose indexed access and length; arrays and objects expose size without Go iteration. | ✓ VERIFIED | `Array.At`, `Len`/`LenErr`, and `Object.Size`/`SizeErr` call native helpers; populated, empty, wrong-kind, closed, and bounds tests pass. |
| 4 | `Minify` and `ValidateUTF8` are allocation-conscious SIMD wrappers with overlap, empty, malformed, and cross-platform tests. | ✗ FAILED | Main behavior and tests exist, but the public ABI loses `out_written` on `BUFFER_TOO_SMALL`; the passing test misses this assertion. |

### Plan Must-Have Truths

Every PLAN frontmatter truth was checked against current code and fresh execution. The score below is based on these 55 plan truths.

| Plan | Truth | Status | Evidence |
|---|---|---|---|
| 12-01 T1 | ABI 1.3 keeps INVALID_PATH=11 and INDEX_OUT_OF_RANGE=12 distinct. | ✓ VERIFIED | `src/lib.rs` pins ABI `0x0001_0003` and codes 11/12; generated header and Go constants agree. |
| 12-01 T2 | AtPointer delegates RFC 6901 resolution and returns a registered descendant. | ✓ VERIFIED | `Element.AtPointer` → binding → Rust registry → `psimdjson_element_at_pointer_index` → upstream `at_pointer`; tests pass. |
| 12-01 T3 | AtPath delegates the documented simdjson dot/index subset. | ✓ VERIFIED | Same end-to-end chain reaches upstream `at_path`; dot-path and invalid bare-name cases pass. |
| 12-01 T4 | Header contract requires both navigation symbols. | ✓ VERIFIED | Both are in `REQUIRED_SYMBOLS`; `make verify-contract` passed. |
| 12-01 T5 | ABI checker pins exact navigation signatures. | ✓ VERIFIED | `diag-surface` signature fixtures passed in the fresh contract run. |
| 12-02 T1 | Native Array.At is upstream's O(n) tape scan, not random access. | ✓ VERIFIED | C++ bridge calls `array.at(size_t(index))`; Go docs disclose O(n)/O(n²) behavior. |
| 12-02 T2 | Native Array.Len/Object.Size are O(1) tape counts with 0xFFFFFF saturation. | ✓ VERIFIED | C++ bridge calls `array.size()`/`object.size()`; Go docs disclose the exact cap. |
| 12-02 T3 | Out-of-range Array.At maps to status 12. | ✓ VERIFIED | `map_error` maps `INDEX_OUT_OF_BOUNDS`; Rust and Go bounds tests pass. |
| 12-02 T4 | Contract requires all three DOM-04 symbols. | ✓ VERIFIED | Required-symbol and generated-header contract run passed. |
| 12-02 T5 | Empty/wrong-kind native cases and exact signatures are tested. | ✓ VERIFIED | Navigation suite passed 19/19 including empty and wrong-kind size cases. |
| 12-03 T1 | Wildcard scratch indices are synchronously copied to a Rust-owned ValueView array. | ✓ VERIFIED | C++ doc owns `wildcard_indices`; registry copies under `with_resolved_view` before returning. |
| 12-03 T2 | Wildcard results preserve order and valid empty results are OK/null/zero. | ✓ VERIFIED | Spike-005 table test and Go non-nil-empty tests pass. |
| 12-03 T3 | View arrays use a separate allocation ledger. | ✓ VERIFIED | `view_array_allocations` is distinct from `byte_allocations`. |
| 12-03 T4 | ValueView free rejects mismatch and double free. | ✓ VERIFIED | Exact pointer/count ledger and focused free-discipline tests pass. |
| 12-03 T5 | Copied views outlive the carrier, not the document; same-doc calls serialize. | ✓ VERIFIED | Lifetime and concurrency tests pass. |
| 12-03 T6 | Null/count boundary pairs are explicit. | ✓ VERIFIED | Null/zero succeeds; null/nonzero and nonnull/zero reject in implementation and tests. |
| 12-03 T7 | Wildcard/free exact signatures are enforced. | ✓ VERIFIED | `make verify-contract` passed both signatures. |
| 12-04 T1 | Native minify checks capacity before upstream entry. | ✓ VERIFIED | `psimdjson_minify` stores `src_len`, rejects short capacity at lines 1130-1135, before kernel/upstream work. |
| 12-04 T2 | Only same-start aliasing or disjoint buffers are accepted. | ✓ VERIFIED | Both partial-overlap directions reject before writes; Rust/Go tests and sanitizer gate pass. |
| 12-04 T3 | Utilities apply fallback/selection-lock ordering and scan after mutex release. | ✓ VERIFIED | Both C++ utility functions use scoped selection locking; fresh utility-lock test passes. |
| 12-04 T4 | Minify is documented as non-validating except unclosed strings. | ✓ VERIFIED | C++/Rust/Go comments and normative FFI text state this limitation; malformed test passes. |
| 12-04 T5 | Focused Rust/header checks cover both utility exports and exact signatures. | ✗ FAILED | Signatures pass, but the undersized test omits the decisive `written == src_len` assertion; direct ABI execution proves the missed failure. |
| 12-05 T1 | Go ABI and status constants match native ABI 1.3/codes 11/12. | ✓ VERIFIED | Go numeric tests and compile-time constants pass. |
| 12-05 T2 | Bootstrap source identity is unpublished `0.2.0-dev`, not a false release. | ✓ VERIFIED | Source constant/canary use `0.2.0-dev`; no release mutation occurred. |
| 12-05 T3 | Bootstrap docs distinguish current source from historical ABI 1.2 and require local override. | ✓ VERIFIED | Documentation text and `make verify-docs` pass. |
| 12-06 T1 | Public AtPointer exposes typed RFC 6901 errors. | ✓ VERIFIED | RFC escapes, malformed syntax, missing, wrong-kind, out-of-range, root, and trailing-empty-key cases pass. |
| 12-06 T2 | Public AtPath documents leading separator and bracket-quote asymmetry. | ✓ VERIFIED | Doc comments and dedicated behavior cases are present and pass. |
| 12-06 T3 | AtPathAll rejects wildcard-free paths and normalizes branch misses to non-nil empty. | ✓ VERIFIED | Go precheck precedes FFI; empty/missing/out-of-range/non-container cases pass. |
| 12-06 T4 | Navigation sentinels are distinct and missing/wrong-type reuse existing sentinels. | ✓ VERIFIED | Shared status switch and `errors.Is` tests pass. |
| 12-06 T5 | ErrBufferTooSmall is public; only invalid argument maps intentionally to internal error. | ✓ VERIFIED | Sentinel and switch mapping are present. |
| 12-07 T1 | Minify allocates; MinifyInto allows exact alias/disjoint and rejects partial overlap pre-FFI. | ✓ VERIFIED | Public code and both-direction unchanged-storage tests pass. |
| 12-07 T2 | Short Go destinations reject pre-FFI with ErrBufferTooSmall. | ✓ VERIFIED | `MinifyInto` checks lengths before load/call; unchanged-buffer test passes. |
| 12-07 T3 | Utilities operate on caller-owned slices without Doc/Parser lifetime coupling. | ✓ VERIFIED | APIs are package functions over byte slices and active library bindings. |
| 12-07 T4 | ValidateUTF8 returns `(bool,error)` and invalid UTF-8 is `(false,nil)`. | ✓ VERIFIED | Valid, invalid, empty, parser-regression, and operational-gate tests pass. |
| 12-07 T5 | Go kernel state mirrors preflight/CPU/success/post-gate outcomes. | ✓ VERIFIED | Subprocess-isolated utility lock tests pass. |
| 12-08 T1 | Array.At returns `(Element,error)` with no error-hiding twin. | ✓ VERIFIED | Exact method shape is present. |
| 12-08 T2 | Array.At uses Go `int` and rejects negatives before FFI. | ✓ VERIFIED | Code precheck and poisoned-view negative-index test pass. |
| 12-08 T3 | Len/LenErr and Size/SizeErr follow panic-safe dual methods. | ✓ VERIFIED | Empty, closed, zero-value, and wrong-kind cases pass. |
| 12-08 T4 | Out-of-range At returns ErrIndexOutOfRange. | ✓ VERIFIED | Public and native tests pass. |
| 12-08 T5 | O(n) and 16,777,215 saturation are documented. | ✓ VERIFIED | Public comments contain both exact disclosures. |
| 12-09 T1 | Packed native/header ABI is `0x00010003`. | ✓ VERIFIED | Rust constant, cbindgen source, generated header, and Go mirror agree. |
| 12-09 T2 | Status values 11/12 are append-only/distinct. | ✓ VERIFIED | Rust/C/Go numeric assertions pass. |
| 12-09 T3 | Rust, generated C, C assertions, and Python fixtures agree. | ✓ VERIFIED | Fresh full contract gate passed. |
| 12-09 T4 | Normative table describes statuses without renumbering older values. | ✓ VERIFIED | Documentation and contract check pass. |
| 12-10 T1 | Durable D-14 probe is outside planning and dynamically counts supported kernels. | ✓ VERIFIED | Script/probe inspection plus fresh run: `kernels=arm64,fallback total=24 runs=3`. |
| 12-10 T2 | Phase-2 workflow runs D-14 when script/tests-native change. | ✓ VERIFIED | Both paths and invocation are in the workflow. |
| 12-10 T3 | Native smoke resolves/invokes all nine ABI 1.3 exports. | ✓ VERIFIED | Closed-world source checks pass; fresh native smoke passed. |
| 12-10 T4 | Five-platform Go workflow includes the Phase 12 branch. | ✓ VERIFIED | Exact branch is configured and workflow contract tests pass. |
| 12-10 T5 | Current loader-contract text requires ABI 1.3 and labels ABI 1.2 historical/rejected. | ✓ VERIFIED | Normative docs contain the required wording; docs gate passes. |
| 12-10 T6 | Workflow tests pin triggers, smoke calls, and absence of planning-local production dependencies. | ✓ VERIFIED | 17/17 workflow-contract tests pass. |
| 12-11 T1 | All nine Phase 12 symbols are mandatory, never optional. | ✓ VERIFIED | All nine appear in the mandatory `symbols` slice. |
| 12-11 T2 | Wrappers preserve ordering/KeepAlive/copies and wildcard arrays free once. | ✓ VERIFIED | Manual data-flow trace and binding tests verify marshaling and copy-before-free. |
| 12-11 T3 | Binding tests exercise all nine Phase 12 symbols and boundary states. | ✓ VERIFIED | `go test ./internal/ffi` passes; test matrix includes wildcard/minify/UTF-8 cases. |
| 12-11 T4 | Release policy accepts 0.2.0-dev/ABI 1.3 and rejects 0.1.7. | ✓ VERIFIED | 14/14 policy tests pass. |
| 12-11 T5 | Loader binds/caches only a complete ABI 1.3 surface and fails closed when incomplete. | ✗ FAILED | Two public earlier-surface symbols are optional and absent from both “complete” fixture lists. |

**Score:** 53/55 plan truths verified

### Required Artifacts

The artifact query reported 43/43 declared artifact occurrences present and substantive. Manual Level-3/Level-4 checks found the following functional exceptions.

| Artifact/group | Expected | Status | Details |
|---|---|---|---|
| `element.go`, `errors.go`, navigation/indexed Go tests | Public DOM APIs, taxonomy, lifecycle, and docs | ✓ VERIFIED | Substantive, called through live bindings, and covered by passing public tests. |
| `src/native/simdjson_bridge.cpp`, `src/runtime/registry.rs`, `tests/rust_shim_navigation.rs` | Upstream navigation/container bridge, tracked wildcard transport | ✓ VERIFIED | Exists, substantive, wired, and real document data flows through it. |
| `minify.go`, `utf8.go`, `kernel.go`, Go utility tests | Public Go SIMD utilities and kernel-state handling | ✓ VERIFIED | Public behavior passes focused and race-enabled tests. |
| `src/lib.rs`, `src/runtime/mod.rs` | Public native utility export and thin runtime handoff | ✗ PARTIAL | Success paths flow; `BUFFER_TOO_SMALL` loses the required `written` value between bridge and export. |
| `tests/rust_shim_minify.rs` | Raw utility boundary coverage | ✗ PARTIAL | Exercises status 6 but never asserts its required output parameter. |
| `internal/ffi/bindings.go` | Complete mandatory ABI binding before cache | ✗ PARTIAL | Nine Phase 12 symbols are mandatory, but two earlier public symbols are downgraded to optional. |
| `internal/ffi/bindings_test.go`, `library_loading_test.go` | Complete/incomplete ABI 1.3 fail-closed tests | ✗ PARTIAL | Both fixture lists omit allocator reset/snapshot and model an incomplete surface as complete. |
| ABI/header sources and tests (`src/lib.rs`, `cbindgen.toml`, `include/`, `tests/abi/`) | ABI 1.3 numeric/signature contract | ✓ VERIFIED | Deterministic regeneration, Python rules, and C layout compile pass. |
| Bootstrap/release policy sources and docs | Unreleased 0.2.0-dev/ABI 1.3 identity | ✓ VERIFIED | Constants, canary, policy tests, and docs agree. |
| Durable probe, smoke, workflows, workflow tests | Cross-platform/sanitizer gates | ✓ VERIFIED | Sources are substantive and wired; local D-14 and native smoke execute successfully. |

All other declared artifacts (`internal/ffi/types*.go`, `internal/bootstrap/*`, `docs/bootstrap.md`, `docs/ffi-contract.md`, `element_*_test.go`, `minify_test.go`, `utf8_test.go`, policy scripts, and workflow-contract tests) passed existence, substance, and wiring checks.

### Key Link Verification

`gsd-sdk verify.key-links` could not parse the PLANs' annotated `from` strings (for example, `element.go Element.AtPointer`) and emitted false “Source file not found” results. Each link was therefore traced manually.

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Go navigation/container methods | `internal/ffi` bindings | Typed wrapper calls + `runtime.KeepAlive` | ✓ WIRED | All methods call their intended bindings and consume returned values/statuses. |
| Go utility functions | active library + kernel state + bindings | Preflight, `kernelMu`, binding call, status mapping | ✓ WIRED | Public utility and subprocess lock tests pass. |
| Rust public navigation exports | registry | `runtime::registry::*` calls inside `ffi_wrap` | ✓ WIRED | Output pointers validated and registry results written. |
| Registry navigation/container helpers | native C++ bridge | `native_*` wrappers | ✓ WIRED | Upstream results are returned/registered; no static fallback data. |
| Wildcard C++ scratch | Rust-owned carrier | synchronous index copy under registry mutex | ✓ WIRED | Lifetime/concurrency/free tests pass. |
| Minify C++ bridge | Rust runtime/public export | status + required-capacity handoff | ✗ PARTIAL | Status flows, but `written` is discarded on non-OK status. |
| ABI source | cbindgen config/header/checkers | generated header + contract gate | ✓ WIRED | ABI/signature checks pass. |
| Phase-2 workflow | D-14 verifier | push paths + run step | ✓ WIRED | Source link exists and verifier passes locally. |
| Release workflow | native export smoke | existing `run_native_smoke.sh` path | ✓ WIRED | Fresh dynamic smoke passed on darwin/arm64. |
| Loader version probe | mandatory binding | probe, bind, then cache | ✗ PARTIAL | Nine new symbols are mandatory, but earlier allocator telemetry bypasses fail-closed binding. |

### Data-Flow Trace (Level 4)

| Artifact/flow | Data | Source | Produces real data | Status |
|---|---|---|---|---|
| `Element.AtPointer` / `AtPath` | Result `ValueView` | Parsed native DOM → upstream navigation → registered descendant | Yes | ✓ FLOWING |
| `Element.AtPathAll` | Ordered `[]Element` | Upstream wildcard matches → doc scratch indices → Rust-owned views → Go copy | Yes | ✓ FLOWING |
| `Array.At`, `Len`, `Object.Size` | Element/count | Native DOM tape | Yes | ✓ FLOWING |
| `Minify` success | Compacted bytes + written count | Upstream SIMD minifier | Yes | ✓ FLOWING |
| `Minify` short-destination error | Required capacity | C++ sets `src_len`, Rust discards it | No at public ABI | ✗ DISCONNECTED |
| `ValidateUTF8` | Validity bit | Upstream SIMD validator | Yes | ✓ FLOWING |
| ABI 1.3 loader gate | Required symbol set | Header/normative ABI → Go binding table → cache | Incomplete | ✗ HOLLOW |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Release library builds | `cargo build --release` | Exit 0 | ✓ PASS |
| Native navigation/wildcard/minify/UTF-8 suites | `cargo test --test rust_shim_navigation --test rust_shim_minify --test rust_shim_utility_lock -- --test-threads=1` | 31 passed, 0 failed | ✓ PASS |
| Public Phase 12 Go behavior under race detector | Focused `go test . -race -run ...` | Exit 0 | ✓ PASS |
| Complete Go tree under race detector | `go test ./... -race -count=1` | All 4 packages passed | ✓ PASS |
| ABI/header contract | `make verify-contract` | All Rust tests, header diff/rules, and C layout passed | ✓ PASS |
| Documentation contract | `make verify-docs` | Exit 0 | ✓ PASS |
| D-14 alias safety | `bash scripts/ci/verify_minify_buffer_safety.sh` | `kernels=arm64,fallback total=24 runs=3` | ✓ PASS |
| Dynamic ABI surface smoke | `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` | `ffi export surface smoke passed` | ✓ PASS |
| Minify status-6 required capacity | Direct `ctypes` call to built `pure_simdjson_minify` | `rc=6 written=18446744073709551615 dst_unchanged=True expected_written=8` | ✗ FAIL |
| Header-required vs loader-mandatory set | `comm` over `tests/abi/check_header.py` and the mandatory binding table | Missing: allocator reset and snapshot | ✗ FAIL |

### Probe Execution

No PLAN/SUMMARY declares a conventional `probe-*.sh`, and none exists under `scripts/**/tests`. The phase's probe-equivalent D-14 verifier was explicitly run and passed three sanitizer executions as shown above.

### Requirements Coverage

All six IDs declared across PLAN frontmatter exactly match the six Phase 12 mappings in `REQUIREMENTS.md`; no orphaned Phase 12 requirement exists.

| Requirement | Source plans | Description | Status | Evidence |
|---|---|---|---|---|
| DOM-01 | 12-01, 12-05, 12-06, 12-09, 12-10, 12-11 | RFC 6901 `AtPointer` via upstream navigation | ✓ SATISFIED | Full native/Go chain and RFC/error tests pass. |
| DOM-02 | 12-01, 12-05, 12-06, 12-09, 12-10, 12-11 | Documented simdjson dot/index `AtPath` subset | ✓ SATISFIED | Native delegation, honest docs, and behavior tests pass. |
| DOM-03 | 12-03, 12-05, 12-06, 12-09, 12-10, 12-11 | Ordered wildcard `AtPathAll` with document lifetime | ✓ SATISFIED | Ordered/empty/partial/lifetime/free/concurrency paths pass. |
| DOM-04 | 12-02, 12-05, 12-08, 12-09, 12-10, 12-11 | Indexed arrays and constant-time container counts | ✓ SATISFIED | Public/native behavior and edge cases pass. |
| UTIL-01 | 12-04, 12-05, 12-06, 12-07, 12-09, 12-10, 12-11 | Allocation-conscious SIMD minify API | ✗ BLOCKED | Main API works, but its normative status-6 required-capacity behavior is broken at the public ABI. |
| UTIL-02 | 12-04, 12-05, 12-07, 12-09, 12-10, 12-11 | Standalone SIMD UTF-8 validation | ✓ SATISFIED | Valid/invalid/empty/CPU/parse-regression tests pass. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---:|---|---|---|
| `src/runtime/mod.rs` | 328-340 | Error return discards a required out-parameter | 🛑 Blocker | Breaks public minify capacity negotiation. |
| `tests/rust_shim_minify.rs` | 191-205 | Sentinel initialized but never asserted | 🛑 Blocker | Lets the broken out-parameter contract pass the full suite. |
| `internal/ffi/bindings.go` | 148-159 | Public ABI symbols registered as optional | 🛑 Blocker | Allows incomplete ABI 1.3 artifacts to be cached. |
| `src/lib.rs` | 960-968, 1006-1014 | Iterator lease allocated before null output is rejected | ⚠ Warning | A rejected iterator constructor consumes unreachable document bookkeeping. `git blame` shows this predates Phase 12. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | 5-16 | Contract gate paths omit `cbindgen.toml` and `tests/abi/**` | ⚠ Warning | Those contract inputs can change without scheduling this workflow. |
| `.github/workflows/phase2-rust-shim-smoke.yml` | 41 | `cargo install cbindgen --locked` has no version | ⚠ Warning | Generator behavior/toolchain requirements can drift. |
| `materializer_fastpath.go` | 217 | `go vet` unsafe-pointer diagnostic | ℹ Info | Fresh `go vet ./...` exits 1; blame and `deferred-items.md` confirm it predates Phase 12. |

No `TBD`, `FIXME`, or `XXX` debt markers were found in Phase 12 files. The `return NULL` matches in the C smoke are legitimate checked allocation-helper returns, not stubs.

### Human Verification Required

None for this verdict. No deferred `<human-check>` blocks exist in the plans. Actual five-platform hosted release evidence remains explicitly owned by Phase 16; Phase 12's workflow wiring and local native execution were verified here.

### Deferred Items

Neither blocker is specifically covered by a later phase goal or success criterion. Phase 16's general release stabilization is not specific enough to defer an observable Phase 12 contract violation, so both remain actionable gaps.

### Gaps Summary

The feature breadth is real, not placeholder work: navigation, wildcard ownership, container helpers, public Go utilities, ABI generation, sanitizer checks, and dynamic smoke all execute successfully. The phase still cannot pass because its fail-closed edges are incomplete:

1. Minify's native bridge computes the required destination capacity on status 6, but the Rust handoff discards it before the public caller can observe it; the test suite fails to assert the output.
2. The ABI 1.3 loader's mandatory surface excludes two earlier public allocator-telemetry exports, contradicting the generated/header contract and allowing an incomplete library to bind and cache.

These are not deferred roadmap work and no verification override exists.

---

_Verified: 2026-07-31T12:50:20Z_
_Verifier: the agent (gsd-verifier)_
