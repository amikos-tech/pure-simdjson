---
phase: 11
slug: upstream-simdjson-refresh-bigint-and-diagnostics
status: source-gate-approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-22
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for the upstream refresh, exact BigInt behavior, production diagnostics, parser limits, ABI 1.2, and coordinated bootstrap artifact state.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing`/fuzz with race detection; Rust unit and integration tests; Python `unittest`; C ABI compile/smoke checks; GitHub Actions five-platform matrix. |
| **Config file** | `Makefile`, `Cargo.toml`, `go.mod`, and `.github/workflows/*.yml`; no separate test configuration. |
| **Quick run command** | `cargo test --locked --test rust_shim_accessors --test rust_shim_minimal -- --test-threads=1 && cargo build --release --locked && go test . -run 'Test(BigInt|ParserOption|ParserPoolOption|Kernel|ErrorOffset|Capacity|Depth)' -count=1` |
| **Full suite command** | `make verify-contract && cargo build --release --locked && go test ./... -race && python3 scripts/release/test_check_bootstrap_abi_state.py && python3 scripts/release/test_release_workflow_contracts.py && python3 scripts/release/test_public_bootstrap_validation_contracts.py && python3 scripts/release/test_render_release_notes.py` |
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

*Status: ✅ green — the exact commands and zero exits are recorded in `11-ARTIFACT-READINESS.md`.*

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
- [x] Extended release/bootstrap policy tests for recovery version `0.1.6` and ABI 1.2.
- [x] Updated every deliberate `NewParserPool` source-break call site, including tests, examples, and `docs/concurrency.md`.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Select and approve the intermediate ABI 1.2 artifact semantic version | D-17 | Operator authority was required | Resolved by `11-01`: recovery version `0.1.6` supersedes the failed, unpublished `v0.1.5` attempt without consuming Phase 16's final v0.2 label. |
| Publish matching ABI 1.2 artifacts through the supported path | UP-01, D-17, D-18 | Requires `origin/main`, an annotated tag, hosted runners, signing, and external publication | `11-14-T1` must run `bash scripts/release/check_readiness.sh --strict --version 0.1.6` on the approved `main` commit, create/push the annotated tag, and require `.github/workflows/release.yml` to pass. Never hand-upload artifacts. |
| Prove fresh-runner default bootstrap after publication | D-17, D-18 | Requires live R2/GitHub release assets and hosted target runners | `11-14-T1` must dispatch `.github/workflows/public-bootstrap-validation.yml` for `0.1.6` and require the full R2 matrix plus documented GitHub fallback subset to pass. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verification or Wave 0 dependencies.
- [x] Sampling continuity: no three consecutive implementation tasks lack an automated check.
- [x] Wave 0 covers every missing test/fixture named above.
- [x] No watch-mode flags are used.
- [x] Focused feedback latency stays below 120 seconds on warm builds.
- [x] Local full suite is green; exact zero exits are in `11-ARTIFACT-READINESS.md`.
- [ ] `11-14-T1 operator gate`: five-platform publication and Phase 06.1 public bootstrap validation are green.
- [x] `nyquist_compliant: true` and `wave_0_complete: true` are set in frontmatter.

**Approval:** source-gate approved — evidence: 11-ARTIFACT-READINESS.md; hosted release proof: 11-14-T1
