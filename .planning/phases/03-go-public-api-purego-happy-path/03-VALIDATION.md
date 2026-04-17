---
phase: 03
slug: go-public-api-purego-happy-path
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-16
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` with repo-local native library builds (`cargo build --release`) |
| **Config file** | `go.mod` |
| **Quick run command** | `cargo build --release && go test ./...` |
| **Full suite command** | `cargo build --release && go test ./... -race` plus the observed `scripts/phase3-go-wrapper-smoke.sh` proof on `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64` |
| **Estimated runtime** | ~90 seconds local plus GitHub Actions wrapper-smoke runtime |

---

## Sampling Rate

- **After every task commit:** Run the task's `<automated>` verification command.
- **After every plan wave:** Run `cargo build --release && go test ./... -race`.
- **Before `/gsd-verify-work`:** Full suite plus the observed five-target wrapper-smoke proof must be green.
- **Max feedback latency:** 90 seconds for local loops; the GitHub Actions wrapper-smoke proof is the explicit later-wave exception.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | API-03 / API-12 | T-03-01-02 / T-03-01-03 | The hidden purego layer binds only the real Phase 3 ABI surface and keeps Go-backed copy buffers alive across native calls | build/static | `cargo build --release && go test ./... && rg 'ABIVersion|ErrInvalidJSON|ValueKindInt64' internal/ffi/types.go && rg 'RegisterFunc|runtime.KeepAlive|pure_simdjson_(get_abi_version|get_implementation_name_len|copy_implementation_name|parser_new|parser_free|parser_parse|parser_get_last_error_len|parser_copy_last_error|parser_get_last_error_offset|doc_free|doc_root|element_type|element_get_int64)' internal/ffi/bindings.go && ! rg 'SyscallN|pure_simdjson_element_get_uint64|pure_simdjson_element_get_float64|pure_simdjson_object_get_field|pure_simdjson_object_iter_next' internal/ffi/bindings.go` | `go.mod`, `internal/ffi/types.go`, `internal/ffi/bindings.go` | ✅ green |
| 03-01-02 | 01 | 1 | API-03 / API-12 | T-03-01-01 / T-03-01-04 | Local loading is deterministic and full-path-only, and native error detail never bleeds across unrelated later failures | build/unit | `cargo build --release && go test ./... && rg 'PURE_SIMDJSON_LIB_PATH|target/release|target/debug|attempted' library_loading.go && ! rg 'WalkDir|filepath.Walk|Glob\\(|BootstrapSync|PURE_SIMDJSON_BINARY_MIRROR|releases\\.amikos\\.tech' library_loading.go library_unix.go library_windows.go && rg 'type Error struct|ErrInvalidHandle|ErrClosed|ErrParserBusy|ErrNumberOutOfRange|ErrPrecisionLoss|ErrCPUUnsupported|ErrABIVersionMismatch|ErrInvalidJSON|ErrElementNotFound|ErrWrongType|Code int32|Offset uint64|Message string|Unwrap\\(' errors.go` | `errors.go`, `library_loading.go`, `library_unix.go`, `library_windows.go` | ✅ green |
| 03-02-01 | 02 | 2 | API-01 / API-02 / API-03 / API-09 | T-03-02-01 / T-03-02-03 | Parser/doc lifecycle is explicit in code: ABI mismatch is early, busy-close stays visible, and purego liveness guards are present before semantic tests land | build/static | `cargo build --release && go test ./... && rg 'runtime.KeepAlive|ErrABIVersionMismatch|ErrParserBusy' parser.go doc.go element.go` | `parser.go`, `doc.go`, `element.go` | ✅ green |
| 03-02-02 | 02 | 2 | API-01 / API-02 / API-03 / API-09 / API-12 | T-03-02-01 / T-03-02-04 | The public happy path, busy-close rule, closed-state behavior, and structured error semantics are proven with named tests | unit | `cargo build --release && go test ./... -run 'Test(HappyPathGetInt64|ABIMismatchAtNewParser|ParserDoubleClose|DocDoubleClose|ParserCloseWhileDocLive|ParseAfterClose|AccessorAfterClose|ParserBusy|StructuredErrorDetails)'` | `parser_test.go` | ✅ green |
| 03-03-01 | 03 | 3 | API-10 / API-11 | T-03-03-01 / T-03-03-03 | `ParserPool` and build-tag-specific finalizer helpers compile and are wired before stress tests run | build/static | `cargo build --release && go test ./... && rg 'type ParserPool struct|func NewParserPool\\(|func \\(p \\*ParserPool\\) Put\\(' pool.go && rg 'attachParserFinalizer|attachDocFinalizer|clearParserFinalizer|clearDocFinalizer' parser.go doc.go finalizer_prod.go finalizer_testbuild.go` | `pool.go`, `finalizer_prod.go`, `finalizer_testbuild.go` | ✅ green |
| 03-03-02 | 03 | 3 | API-10 / API-11 | T-03-03-02 / T-03-03-03 | Pool reuse, pool-eviction cleanup, race safety, and leak warnings are proven, including the roadmap's 10,000-parser warning scenario in test builds only | unit/race/stress | `cargo build --release && go test ./... -run 'Test(ParserPoolRoundTrip|ParserPoolRejectsBusy|ParserPoolRejectsClosed|PooledParserEvictionCleansUp|LeakWarningSilentProd)' && cargo build --release && go test ./... -race -run 'Test(ParserPoolRoundTrip|ParserPoolRejectsBusy)' && cargo build --release && go test ./... -tags purejson_testbuild -run 'Test(LeakWarningTestBuild|LeakWarningMassLeak10000)'` | `pool_test.go`, `parser_test.go` | ✅ green |
| 03-04-01 | 04 | 4 | API-03 / API-09 / API-10 | T-03-04-02 / T-03-04-03 | Local verification targets and workflow definition stay narrow, use pinned cross-arch runner labels for all five Phase 3 targets, and prepare Windows MSVC explicitly | static | `test -f .github/workflows/phase3-go-wrapper-smoke.yml && rg '^phase3-go-test|^phase3-go-race|^phase3-go-wrapper-remote' Makefile && rg 'workflow_dispatch|linux-amd64-go-race|linux-arm64-go-race|darwin-amd64-go-race|darwin-arm64-go-race|windows-amd64-go-race|ubuntu-24.04-arm|macos-15-intel|macos-15|windows-latest|ilammy/msvc-dev-cmd@v1|cargo build --release|go test ./\\.\\.\\. -race' .github/workflows/phase3-go-wrapper-smoke.yml && ! rg 'upload-artifact|cosign|PURE_SIMDJSON_BINARY_MIRROR|releases\\.amikos\\.tech|cmd/pure-simdjson-bootstrap|BootstrapSync' .github/workflows/phase3-go-wrapper-smoke.yml Makefile` | `Makefile`, `.github/workflows/phase3-go-wrapper-smoke.yml` | ✅ green |
| 03-04-02 | 04 | 4 | API-03 / API-09 / API-10 | T-03-04-01 / T-03-04-03 | The wrapper-smoke proof is observed on a specific branch-scoped run id instead of a timing heuristic, and all five target jobs conclude success | integration/CI | `gh auth status -h github.com && gh run view 24500326284 --json conclusion,headBranch,headSha,jobs,name,workflowName,url` | `scripts/phase3-go-wrapper-smoke.sh`, `.github/workflows/phase3-go-wrapper-smoke.yml` | ✅ green |
| 03-05-01 | 05 | 4 | DOC-03 | T-03-05-01 | Source docs describe the exact Phase 3 API, including the package-name/module-path split and pool semantics | static/doc | `rg '^// Package purejson|^// Error|^// Parser|^// Doc|^// Element|^// Array|^// Object|^// ParserPool|^// NewParser|^// Parse|^// Close|^// Root|^// GetInt64|^// NewParserPool|^// Get|^// Put' purejson.go errors.go parser.go doc.go element.go pool.go` | `purejson.go`, commented Go source files | ✅ green |
| 03-05-02 | 05 | 4 | DOC-04 | T-03-05-02 | `docs/concurrency.md` states the single-doc invariant, pool rejection rules, and leak-warning behavior exactly as implemented | static/doc | `rg '## Invariant|## Why Parsers Are Not Shareable|## ParserPool Pattern|## Put Rejection Rules|## Leak Warnings|single-doc invariant|goroutine-per-parser|ParserPool.Put|purejson_testbuild' docs/concurrency.md` | `docs/concurrency.md` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `go.mod` is created in Plan `03-01-01`
- [x] Deterministic library-loading helpers are created in Plan `03-01-02`
- [x] Initial wrapper test scaffolding is created in Plan `03-02-02`

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or audited automation evidence
- [x] Sampling continuity preserved across waves
- [x] Wave 0 gaps resolved by Plans 01-02
- [x] No watch-mode flags
- [x] Local feedback latency < 90s
- [x] Cross-platform wrapper smoke is explicit and does not rely on bootstrap/download behavior
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** refreshed 2026-04-16

---

## Validation Audit 2026-04-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Fresh audit evidence:

- `cargo build --release && go test ./...` passed.
- `cargo build --release && go test ./... -run 'Test(HappyPathGetInt64|ABIMismatchAtNewParser|ParserDoubleClose|DocDoubleClose|ParserCloseWhileDocLive|ParseAfterClose|AccessorAfterClose|ParserBusy|StructuredErrorDetails)'` passed.
- `cargo build --release && go test ./... -run 'Test(ParserPoolRoundTrip|ParserPoolRejectsBusy|ParserPoolRejectsClosed|PooledParserEvictionCleansUp|LeakWarningSilentProd)'` passed.
- `cargo build --release && go test ./... -race` passed.
- `cargo build --release && go test ./... -race -run 'Test(ParserPoolRoundTrip|ParserPoolRejectsBusy)'` passed.
- `cargo build --release && go test ./... -tags purejson_testbuild -run 'Test(LeakWarningTestBuild|LeakWarningMassLeak10000)'` passed.
- Static audit confirmed the FFI bindings, loader/error scaffolding, parser/pool/finalizer hooks, source docs, concurrency guide, and wrapper-smoke workflow assets.
- `gh run view 24500326284 --json conclusion,headBranch,headSha,jobs,name,workflowName,url` confirmed `success` for branch `gsd/phase-03-go-public-api-purego-happy-path` at head `9e158a1c7b39812948bca23e84fcaf8b798b46a3`, with `linux-amd64-go-race`, `linux-arm64-go-race`, `darwin-amd64-go-race`, `darwin-arm64-go-race`, and `windows-amd64-go-race` all concluding `success`.
- The audit corrected stale validation commands for the FFI/static rows so they now match the shipped Go identifiers and the audited GitHub run evidence.
