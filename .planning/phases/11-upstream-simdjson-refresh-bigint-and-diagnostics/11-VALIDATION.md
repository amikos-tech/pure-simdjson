---
phase: 11
slug: upstream-simdjson-refresh-bigint-and-diagnostics
status: source-gate-approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-22
last_audited: 2026-07-30
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for the upstream refresh, exact BigInt behavior, production diagnostics, parser limits, ABI 1.2, and coordinated bootstrap artifact state.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing`/fuzz with race detection; Rust unit and integration tests; Python `unittest`; C ABI compile/smoke checks; GitHub Actions five-platform matrix. |
| **Config file** | `Makefile`, `Cargo.toml`, `go.mod`, and `.github/workflows/*.yml`; no separate test configuration. |
| **Quick run command** | `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1 && cargo build --release --locked && go test . -run 'Test(BigInt|ParserOption|ParserPoolOption|Kernel|ErrorOffset|Capacity|Depth)' -count=1` |
| **Full suite command** | `make verify-contract && make verify-docs && cargo build --release --locked && go test ./... -race -count=1 -timeout=180s && python3 scripts/release/test_check_bootstrap_abi_state.py && python3 scripts/release/test_release_workflow_contracts.py && python3 scripts/release/test_public_bootstrap_validation_contracts.py && python3 scripts/release/test_render_release_notes.py` |
| **Estimated runtime** | Quick path ≈2 minutes warm; full local suite ≈10–20 minutes depending on native build cache. |

---

## Sampling Rate

- **After every task commit:** Run the narrow Go/Rust test command for the touched layer; also run `make verify-contract` whenever ABI/header code changes.
- **After every plan wave:** Run `cargo build --release --locked && go test ./... -race`; run the correctness oracle after upstream/native behavior changes.
- **Before `$gsd-verify-work`:** Run the full suite, ensure generated-header diff is clean, and run the existing PR benchmark smoke.
- **Artifact-producing merge gate:** Require the hosted five-platform release build/smoke and Phase 06.1 public bootstrap validation; local tests cannot substitute for these.
- **Max feedback latency:** Keep per-task focused feedback below 120 seconds on a warm development build; split larger gates to wave boundaries.

---

## Threat References

| Ref | Threat | Required control |
|-----|--------|------------------|
| T-11-01 | Oversized input allocates before the configured capacity check | Reject length in Rust before arena resize/copy and return an honest typed error. |
| T-11-02 | Unsupported forced kernel executes illegal instructions | Validate compiled implementation name and runtime CPU support before selection. |
| T-11-03 | ABI 1.2 artifact omits a mandatory symbol | Probe ABI first, then require the complete ABI 1.2 symbol table and fail closed. |
| T-11-04 | BigInt text escapes document/native lifetime | Copy exact text through the existing Rust-owned copy-out/free path before returning Go text. |
| T-11-05 | Fabricated error offset misdirects diagnostics | Mark known only for an upstream-proven in-range location; otherwise report unknown. |
| T-11-06 | Kernel override races parser or pool creation | Serialize selection/creation and lock selection after the first parser or pool. |
| T-11-07 | Caller-controlled depth exhausts the native stack | Reject depth above 1024 before selection, allocation, replay, or materialization; prove the maximum accepted malformed case in a child process. |
| T-11-16-01..03 | A malformed oversized integer is truncated into a valid BigInt or an architecture copy misses the guard | Validate every BigInt token boundary, fail closed on generated-source parity drift, and persist malformed cases in the correctness oracle. |
| T-11-17-01..03 | A caught C++ exception is misclassified or the hidden test seam widens the ABI | Route all caught exceptions through status 97, keep returned engine `MEMALLOC` distinct, and retain one excluded selector seam. |
| T-11-18-01..07 | Parser-side `std::bad_alloc` terminates before a typed status or mutates success-only output | Perform no allocation before the diagnostic guard and require subprocess proof of status 97, sentinel integrity, and process survival. |

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Evidence Files | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|----------------|--------|
| `11-02-T1/T3`, `11-09-T1`, `11-13-T2` | `11-02`, `11-09`, `11-13` | `1/7/10` | UP-01 | — | Exact v4.6.4 pin, reproducible patch application, generated ABI, and unchanged build contract | contract + integration | `make verify-contract`; `cargo build --release --locked`; exact upstream commit check | `patches/simdjson-v4.6.4-positive-bigint.patch`, `tests/rust_shim_minimal.rs`, `tests/rust_shim_fast_materializer.rs`, `tests/abi/check_header.py`, `11-ARTIFACT-READINESS.md` | ✅ green |
| `11-02-T2/T3`, `11-03-T2`, `11-10-T1/T2` | `11-02`, `11-03`, `11-10` | `1/2/8` | NUM-01, NUM-02 | T-11-04 | Boundary classification, strict getters, copied lifetime, and materializer propagation | native + Go unit | `make verify-contract`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `tests/rust_shim_bigint.rs`, `bigint_test.go`, `materializer_fastpath_test.go`, `element_fuzz_test.go` | ✅ green |
| `11-06-T1/T2`, `11-12-T2` | `11-06`, `11-12` | `5/9` | DIAG-01 | T-11-02, T-11-06 | Kernel validation, automatic reset, unsupported selection, and creation lock | subprocess integration | `make verify-contract`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `tests/rust_shim_kernel.rs`, `kernel_test.go` | ✅ green |
| `11-05-T1/T2`, `11-12-T1` | `11-05`, `11-12` | `4/9` | DIAG-02 | T-11-05 | Known nonzero, known zero, unknown, and stale-detail reset remain truthful end to end | native + Go unit | `make verify-contract`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `tests/rust_shim_diagnostics.rs`, `parser_test.go`, `errors_test.go` | ✅ green |
| `11-04-T1/T2`, `11-11-T1` | `11-04`, `11-11` | `3/8` | LIMIT-01 | T-11-01 | Immutable options validate once; capacity rejects before copy; depth boundaries are exact | Rust white-box + Go integration | `make verify-contract`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `tests/rust_shim_limits.rs`, `parser_options_test.go`, `parser_test.go` | ✅ green |
| `11-11-T2`, `11-12-T2` | `11-11`, `11-12` | `8/9` | LIMIT-01 | T-11-01, T-11-06 | Pool misses preserve normalized config and mismatched `Put` fails | Go race unit | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `pool_test.go`, `kernel_test.go`, `docs/concurrency.md`, `example_test.go` | ✅ green |
| `11-07-T2`, `11-08-T1/T2`, `11-09-T1/T2` | `11-07`, `11-08`, `11-09` | `6/7` | D-15..D-18 | T-11-03 | ABI 1.1 mismatches; complete 1.2 binds; incomplete 1.2 fails closed | loader + ABI contract | `make verify-contract`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `library_loading_test.go`, `internal/ffi/bindings_test.go`, `tests/abi/check_header.py`, `tests/abi/test_check_header.py`, `tests/abi/handle_layout.c`, `tests/smoke/ffi_export_surface.c` | ✅ green |
| `11-07-T1`, `11-13-T1/T2` | `11-07`, `11-13` | `6/10` | D-17, D-18 | T-11-03 | Bootstrap policy/canary and release workflow require the same ABI 1.2 artifact state | Python + packaged smoke contract | four release Python tests plus explicit-library `go run ./tests/smoke/go_bootstrap_smoke.go` | `scripts/release/test_check_bootstrap_abi_state.py`, `scripts/release/test_release_workflow_contracts.py`, `scripts/release/test_public_bootstrap_validation_contracts.py`, `tests/smoke/go_bootstrap_smoke.go`, `11-ARTIFACT-READINESS.md` | ✅ green |
| `11-02-T3`, `11-13-T2` | `11-02`, `11-13` | `1/10` | UP-01 | — | Existing JSON accept/reject behavior remains aligned with the committed oracle | correctness | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test . -run '^TestJSONTestSuiteOracle$' -count=1` | `benchmark_oracle_test.go`, `testdata/jsontestsuite/expectations.tsv`, `11-ARTIFACT-READINESS.md` | ✅ green |
| `11-13-T2` | `11-13` | `10` | UP-01 | — | Representative Tier 1/2/3 paths receive a fresh source signal | benchmark smoke | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir "$PHASE11_BENCH_DIR"` | `scripts/bench/run_pr_benchmark.sh`, `benchmark_fullparse_test.go`, `benchmark_typed_test.go`, `benchmark_selective_test.go`, `11-ARTIFACT-READINESS.md` | ✅ green |
| `11-15-T1/T2` | `11-15` | `12` | DIAG-02, LIMIT-01 | T-11-05, T-11-07 | One 1024 ceiling protects construction, replay, and materialization; the largest accepted malformed case returns a truthful typed diagnostic without terminating | Rust subprocess + Go/native contract | `cargo test --locked --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1`; fresh-library parser/depth Go tests | `tests/rust_shim_limits.rs`, `tests/rust_shim_diagnostics.rs`, `parser_options_test.go`, `materializer_fastpath_test.go` | ✅ green |
| `11-16-T1/T2` | `11-16` | `13` | UP-01, NUM-01 | T-11-16-01..03 | Every generated BigInt branch validates its delimiter; malformed roots and descendants reject while valid exact text remains kind 9 | native ABI + correctness oracle | `cargo test --locked --test rust_shim_bigint -- --test-threads=1`; `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test . -run '^TestJSONTestSuiteOracle$' -count=1` | `tests/rust_shim_bigint.rs`, `build.rs`, `testdata/jsontestsuite/cases/n_*bigint*`, `testdata/jsontestsuite/expectations.tsv` | ✅ green |
| `11-17-T1`, `11-18-T1` | `11-17`, `11-18` | `14/15` | UP-01 | T-11-17-01..03, T-11-18-01..07 | Runtime errors and allocation exceptions remain contained at the C++/Rust boundary; parser-side `bad_alloc` returns 97 without termination or output mutation | direct native + subprocess + ABI contract | `cargo test --locked --test rust_shim_minimal psimdjson_test_force_ -- --nocapture --test-threads=1`; `make verify-contract` | `tests/rust_shim_minimal.rs`, `src/native/simdjson_bridge.cpp`, `include/pure_simdjson.h`, `docs/ffi-contract.md` | ✅ green |

*Status: ✅ green — source-gate evidence is recorded in `11-ARTIFACT-READINESS.md`; current Plans 15–18 evidence is recorded in the audit below.*

---

## Wave 0 Requirements

- [x] `tests/rust_shim_bigint.rs` — native BigInt boundaries, kind `9`, copied text, and strict numeric-accessor status.
- [x] `tests/rust_shim_diagnostics.rs` — characterization corpus for upstream-proven known/nonzero, known/zero, and unknown error locations.
- [x] `tests/rust_shim_kernel.rs` — isolated process-global kernel selection, support validation, auto reset, and post-creation lock behavior.
- [x] Go BigInt tests — root, descendant, iterator, lookup, copied lifetime, numeric wrong-type, and fast-materializer paths.
- [x] Go parser option/pool tests — omitted/explicit defaults, duplicate behavior, invalid values, pre-copy capacity rejection, depth boundaries, homogeneous misses, mismatched `Put`, and race coverage.
- [x] Go error-offset tests — `HasOffset`, known zero, unknown, formatting, and no stale details after a Rust-side capacity rejection.
- [x] ABI binding fixtures — ABI 1.1, complete ABI 1.2, and ABI 1.2 missing one mandatory symbol.
- [x] Extended `tests/abi/check_header.py`, `tests/abi/test_check_header.py`, `tests/abi/handle_layout.c`, and `tests/smoke/ffi_export_surface.c` for ABI `0x00010002`, appended kind, retained layouts, and mandatory exports.
- [x] Extended release/bootstrap policy tests for recovery version `0.1.7` and ABI 1.2.
- [x] Updated every deliberate `NewParserPool` source-break call site, including tests, examples, and `docs/concurrency.md`.

---

## Manual-Only Verifications

**Outstanding:** none.

The former operator-only gates are resolved in `11-14-SUMMARY.md`: the final intermediate ABI 1.2 release is `v0.1.7`, the annotated-tag release workflow passed on all five targets, and fresh-runner public bootstrap validation passed for the full R2 matrix plus the documented GitHub fallback subset. Those immutable hosted results remain external evidence; this audit did not republish or dispatch anything.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verification or Wave 0 dependencies.
- [x] Sampling continuity: no three consecutive implementation tasks lack an automated check.
- [x] Wave 0 covers every missing test/fixture named above.
- [x] No watch-mode flags are used.
- [x] Focused feedback latency stays below 120 seconds on warm builds.
- [x] Local full suite is green; exact zero exits are in `11-ARTIFACT-READINESS.md`.
- [x] `11-14-T1 operator gate`: five-platform publication and Phase 06.1 public bootstrap validation are green; evidence is in `11-14-SUMMARY.md`.
- [x] `nyquist_compliant: true` and `wave_0_complete: true` are set in frontmatter.

**Approval:** source-gate approved — evidence: `11-ARTIFACT-READINESS.md`; hosted release proof: `11-14-SUMMARY.md`; Nyquist re-audit passed 2026-07-30.

## Validation Audit 2026-07-30

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Requirements green | 6/6 |
| Plans newly represented in this map | 4 (`11-15` through `11-18`) |

Current-source evidence:

- `make verify-contract` passed 91 Rust tests, deterministic header regeneration, 25 ABI/header audits, and the C layout compile.
- `make verify-docs` passed.
- A fresh release build plus `go test ./... -race -count=1 -timeout=180s` passed all four Go packages.
- The four release/bootstrap Python suites passed 43/43 tests.
- `TestJSONTestSuiteOracle`, the no-baseline PR benchmark signal, and non-strict release readiness all passed.
- No missing or partial requirement-level validation was found, so no test file was generated and no auditor repair task was required.
