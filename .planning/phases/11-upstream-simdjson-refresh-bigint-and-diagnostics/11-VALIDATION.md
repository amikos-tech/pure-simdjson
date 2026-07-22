---
phase: 11
slug: upstream-simdjson-refresh-bigint-and-diagnostics
status: draft
nyquist_compliant: false
wave_0_complete: false
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-W0-01 | W0 | 0 | UP-01 | — | Exact v4.6.4 pin and unchanged build contract | contract + integration | `git -C third_party/simdjson rev-parse HEAD && make verify-contract && cargo build --release --locked` | ❌ W0 pin assertion | ⬜ pending |
| 11-W0-02 | W0 | 0 | NUM-01, NUM-02 | T-11-04 | Boundary classification, strict getters, copied lifetime, and materializer propagation | native + Go unit | `cargo test --locked --test rust_shim_bigint -- --test-threads=1 && go test . -run '^TestBigInt' -count=1` | ❌ W0 | ⬜ pending |
| 11-W0-03 | W0 | 0 | DIAG-01 | T-11-02, T-11-06 | Kernel validation, auto reset, unsupported selection, and creation lock | subprocess integration | `cargo test --locked --test rust_shim_kernel -- --test-threads=1 && go test . -run '^TestKernel' -count=1` | ❌ W0 | ⬜ pending |
| 11-W0-04 | W0 | 0 | DIAG-02 | T-11-05 | Known nonzero, known zero, unknown, and stale-detail reset remain truthful end to end | native + Go unit | `cargo test --locked --test rust_shim_diagnostics -- --test-threads=1 && go test . -run '^TestError(HasOffset|OffsetKnownZero|OffsetUnknown|OffsetCorpus)' -count=1` | ❌ W0 | ⬜ pending |
| 11-W0-05 | W0 | 0 | LIMIT-01 | T-11-01 | Immutable options validate once; capacity rejects before copy; depth boundaries are exact | Rust white-box + Go integration | `cargo test registry::tests::capacity -- --test-threads=1 && go test . -run '^Test(ParserOption|ParserCapacity|ParserDepth)' -count=1` | ❌ W0 | ⬜ pending |
| 11-W0-06 | W0 | 0 | LIMIT-01 | T-11-01, T-11-06 | Pool misses preserve normalized config and mismatched `Put` fails | Go race unit | `go test . -run '^TestParserPool.*(Option|Config)' -race -count=1` | ❌ W0 | ⬜ pending |
| 11-W0-07 | W0 | 0 | D-15..D-18 | T-11-03 | ABI 1.1 mismatches; complete 1.2 binds; incomplete 1.2 fails closed | loader + ABI contract | `go test . -run '^TestABI' -count=1 && go test ./internal/ffi -run '^TestBind' -count=1 && make verify-contract` | ⚠️ partial | ⬜ pending |
| 11-W0-08 | W0 | 0 | D-17, D-18 | T-11-03 | Bootstrap policy/canary and release workflow require the same ABI 1.2 artifact state | Python contract | `python3 scripts/release/test_check_bootstrap_abi_state.py && python3 scripts/release/test_release_workflow_contracts.py && python3 scripts/release/test_public_bootstrap_validation_contracts.py && python3 scripts/release/test_render_release_notes.py` | ⚠️ extend existing | ⬜ pending |
| 11-W1-01 | TBD | 1+ | UP-01 | — | Existing JSON accept/reject behavior remains aligned with the committed oracle | correctness | `go test . -run '^TestJSONTestSuiteOracle$' -count=1` | ✅ | ⬜ pending |
| 11-W1-02 | TBD | 1+ | UP-01 | — | Representative Tier 1/2/3 paths receive a regression signal | benchmark smoke | `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir /tmp/phase11-pr-bench` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ partial/flaky*

---

## Wave 0 Requirements

- [ ] `tests/rust_shim_bigint.rs` — native BigInt boundaries, kind `9`, copied text, and strict numeric-accessor status.
- [ ] `tests/rust_shim_diagnostics.rs` — characterization corpus for upstream-proven known/nonzero, known/zero, and unknown error locations.
- [ ] `tests/rust_shim_kernel.rs` — isolated process-global kernel selection, support validation, auto reset, and post-creation lock behavior.
- [ ] Go BigInt tests — root, descendant, iterator, lookup, copied lifetime, numeric wrong-type, and fast-materializer paths.
- [ ] Go parser option/pool tests — omitted/explicit defaults, duplicate behavior, invalid values, pre-copy capacity rejection, depth boundaries, homogeneous misses, mismatched `Put`, and race coverage.
- [ ] Go error-offset tests — `HasOffset`, known zero, unknown, formatting, and no stale details after a Rust-side capacity rejection.
- [ ] ABI binding fixtures — ABI 1.1, complete ABI 1.2, and ABI 1.2 missing one mandatory symbol.
- [ ] Extend `tests/abi/check_header.py`, `tests/abi/test_check_header.py`, `tests/abi/handle_layout.c`, and `tests/smoke/ffi_export_surface.c` for ABI `0x00010002`, appended kind, retained layouts, and mandatory exports.
- [ ] Extend release/bootstrap policy tests for the selected intermediate artifact version and ABI 1.2.
- [ ] Update every deliberate `NewParserPool` source-break call site, including tests, examples, and `docs/concurrency.md`.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Select and approve the intermediate ABI 1.2 artifact semantic version | D-17 | The user did not lock a version, and research/planning does not authorize a tag | Before the release task, record the chosen version in the plan/checkpoint, bootstrap pin, ABI policy, and changelog. Do not consume Phase 16's final v0.2 label by assumption. |
| Publish matching ABI 1.2 artifacts through the supported path | UP-01, D-17, D-18 | Requires `origin/main`, an annotated tag, GitHub-hosted runners, signing, and external publication | On the approved `main` commit run `bash scripts/release/check_readiness.sh --strict --version <version>`, create/push the annotated tag, and require `.github/workflows/release.yml` to pass. Never hand-upload artifacts. |
| Prove fresh-runner default bootstrap after publication | D-17, D-18 | Requires live R2/GitHub release assets and hosted target runners | Dispatch `.github/workflows/public-bootstrap-validation.yml` for the published version and require the full R2 matrix plus documented GitHub fallback subset to pass. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verification or Wave 0 dependencies.
- [ ] Sampling continuity: no three consecutive implementation tasks lack an automated check.
- [ ] Wave 0 covers every missing test/fixture named above.
- [ ] No watch-mode flags are used.
- [ ] Focused feedback latency stays below 120 seconds on warm builds.
- [ ] Local full suite is green.
- [ ] Five-platform publication and Phase 06.1 public bootstrap validation are green.
- [ ] `nyquist_compliant: true` and `wave_0_complete: true` are set in frontmatter.

**Approval:** pending
