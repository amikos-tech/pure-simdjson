---
phase: 03-go-public-api-purego-happy-path
verified: 2026-04-16T08:37:57Z
status: passed
score: 10/10 must-haves verified
overrides_applied: 0
---

# Phase 3: Go Public API + purego Happy Path Verification Report

**Phase Goal:** Wire Go's `purejson` package to the shim with handle lifecycle, `ParserPool`, typed errors, and one accessor as the smoke-tested happy path.
**Verified:** 2026-04-16T08:37:57Z
**Status:** passed
**Re-verification:** No - initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | The repository is a real Go module at `github.com/amikos-tech/pure-simdjson`, and the public package boundary is `purejson`. | ✓ VERIFIED | `go.mod` declares the locked module path; `purejson.go` documents the import-path/package-name split explicitly. |
| 2 | The hidden `internal/ffi` layer binds only the real Phase 3 happy-path native surface through purego `RegisterFunc`, not the broader Phase 4 accessor/iterator surface. | ✓ VERIFIED | `internal/ffi/bindings.go` binds ABI/version helpers, implementation-name helpers, parser lifecycle/diagnostic helpers, `doc_root`, `element_type`, and `element_get_int64`; no broader accessor bindings were added. |
| 3 | Library loading remains deterministic and local-only: explicit full paths, `PURE_SIMDJSON_LIB_PATH` first, then the locked repo-local `target` candidates in order. | ✓ VERIFIED | `library_loading.go`, `library_unix.go`, and `library_windows.go` implement the exact ordered path resolution without bootstrap, globbing, or recursive scans. |
| 4 | Typed Go-side errors exist for the required Phase 3 failure modes, and the structured wrapper preserves native code/message/offset detail without reusing stale diagnostics. | ✓ VERIFIED | `errors.go` defines the sentinel error set plus `type Error struct { Code int32; Offset uint64; Message string; Err error }`; `TestStructuredErrorDetails` passed under `go test ./...`. |
| 5 | The public happy path `NewParser() -> Parse(data) -> doc.Root().GetInt64()` works against the real shim, and `NewParser()` rejects ABI mismatches before parser allocation. | ✓ VERIFIED | `TestHappyPathGetInt64` and `TestABIMismatchAtNewParser` passed; `make phase3-go-test` exited 0 after building the local release shim. |
| 6 | Parser/doc lifecycle behavior is deterministic: parser/doc close is idempotent, use-after-close returns `ErrClosed`, and busy ownership stays visible instead of being silently repaired. | ✓ VERIFIED | `TestParserDoubleClose`, `TestDocDoubleClose`, `TestParseAfterClose`, `TestAccessorAfterClose`, `TestParserBusy`, and `TestParserCloseWhileDocLive` all passed under `go test ./...`. |
| 7 | `ParserPool` exists as the goroutine-per-parser reuse primitive, rejects nil/closed/busy parsers, and finalizer behavior is split correctly between production and `purejson_testbuild`. | ✓ VERIFIED | `pool.go`, `finalizer_prod.go`, and `finalizer_testbuild.go` implement the pool/finalizer contract; `TestParserPoolRoundTrip`, `TestParserPoolRejectsBusy`, `TestParserPoolRejectsClosed`, `TestPooledParserEvictionCleansUp`, `TestLeakWarningSilentProd`, `TestLeakWarningTestBuild`, and `TestLeakWarningMassLeak10000` all passed in prior phase-local verification. |
| 8 | Source docs describe the exact Phase 3 API that exists now, and `docs/concurrency.md` matches the implemented single-doc, parser-pool, and leak-warning behavior. | ✓ VERIFIED | `purejson.go`, `errors.go`, `parser.go`, `doc.go`, `element.go`, and `pool.go` contain exported comments for the current surface; `docs/concurrency.md` documents the single-doc invariant, goroutine-per-parser model, `ParserPool.Put` rejection rules, and prod vs `purejson_testbuild` leak-warning split. |
| 9 | The repo has fresh local verification targets for the wrapper and they passed after the final Phase 3 implementation landed. | ✓ VERIFIED | Fresh regression and phase checks passed: `make verify-contract`, `make verify-docs`, `cargo test`, `make phase2-smoke-linux`, `make phase3-go-test`, and `make phase3-go-race` all exited 0 on 2026-04-16. |
| 10 | The narrow five-target wrapper-smoke workflow proved the Go wrapper remotely on Linux amd64, Linux arm64, Darwin amd64, Darwin arm64, and Windows amd64. | ✓ VERIFIED | GitHub Actions run `24500326284` on branch `gsd/phase-03-go-public-api-purego-happy-path` and head `9e158a1c7b39812948bca23e84fcaf8b798b46a3` concluded `success` with all five required jobs: `linux-amd64-go-race`, `linux-arm64-go-race`, `darwin-amd64-go-race`, `darwin-arm64-go-race`, and `windows-amd64-go-race`. |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `go.mod` | Locked Go module definition for the wrapper | ✓ VERIFIED | Declares module `github.com/amikos-tech/pure-simdjson` and Go 1.24 toolchain line. |
| `internal/ffi/types.go` | Go mirror of the Phase 3 transport types/constants | ✓ VERIFIED | Mirrors the Phase 3 handles, value-view transport, ABI constant, implemented value kinds, and public error-code constants from the committed header. |
| `internal/ffi/bindings.go` | Exact purego bindings for the Phase 3 happy path | ✓ VERIFIED | Uses `purego.RegisterFunc` for the real implemented Phase 3 symbols only. |
| `errors.go` | Typed sentinel errors plus structured native error wrapper | ✓ VERIFIED | Contains the required sentinel set and `Error` wrapper with `Error()`/`Unwrap()`. |
| `library_loading.go`, `library_unix.go`, `library_windows.go` | Deterministic repo-local loader | ✓ VERIFIED | Resolve only explicit full paths and preserve the locked search order. |
| `parser.go`, `doc.go`, `element.go` | Public parser/doc/element happy-path API | ✓ VERIFIED | Provide `NewParser`, `Parse`, `Close`, `Root`, and `GetInt64` with explicit busy/closed semantics and liveness guards. |
| `pool.go`, `finalizer_prod.go`, `finalizer_testbuild.go` | Parser pool and build-tag-specific finalizer behavior | ✓ VERIFIED | Implement the reuse primitive, finalizer split, and pool-safe cleanup behavior. |
| `parser_test.go`, `pool_test.go` | Semantic Phase 3 behavior proof | ✓ VERIFIED | Cover happy path, ABI mismatch, close/busy semantics, pool rejection/eviction, and leak-warning split. |
| `purejson.go` | Package-level documentation | ✓ VERIFIED | Makes the import-path/package-name split explicit for consumers. |
| `docs/concurrency.md` | User-facing concurrency and leak-warning contract | ✓ VERIFIED | Documents the exact shipped parser/doc/pool behavior. |
| `Makefile` | Local verification targets | ✓ VERIFIED | Adds `phase3-go-test`, `phase3-go-race`, and `phase3-go-wrapper-remote`. |
| `.github/workflows/phase3-go-wrapper-smoke.yml` and `scripts/phase3-go-wrapper-smoke.sh` | Narrow remote wrapper proof | ✓ VERIFIED | Workflow stays limited to local shim build plus Go race tests; helper script finds the branch run id and validates named jobs explicitly. |

### Behavioral Spot-Checks

| Behavior | Command / Run | Result | Status |
| --- | --- | --- | --- |
| Phase 1 contract gate still passes | `make verify-contract` | Exit 0 on 2026-04-16 | ✓ PASS |
| Phase 1 docs gate still passes | `make verify-docs` | Exit 0 on 2026-04-16 | ✓ PASS |
| Phase 2 Rust shim suites still pass | `cargo test` | 32 tests passed (10 unit + 2 fallback gate + 20 minimal shim) | ✓ PASS |
| Phase 2 public-header smoke still works locally | `make phase2-smoke-linux` | Printed `phase2 smoke passed` | ✓ PASS |
| Phase 3 local wrapper suite passes | `make phase3-go-test` | Exit 0 on 2026-04-16 | ✓ PASS |
| Phase 3 race suite passes | `make phase3-go-race` | Exit 0 on 2026-04-16 | ✓ PASS |
| Phase 3 remote wrapper matrix passes | GitHub Actions run `24500326284` | Five required jobs concluded `success` | ✓ PASS |

### Requirements Coverage

| Requirement | Description | Status | Evidence |
| --- | --- | --- | --- |
| `API-01` | Package `purejson` with `Parser`, `Doc`, `Element`, `Array`, `Object`, `ParserPool` types | ✓ SATISFIED | Public types exist across `parser.go`, `doc.go`, `element.go`, and `pool.go`. |
| `API-02` | `NewParser() (*Parser, error)` allocates a reusable parser | ✓ SATISFIED | `NewParser` exists and is exercised by the happy-path and lifecycle tests. |
| `API-03` | `Parser.Parse(data []byte) (*Doc, error)` parses JSON and returns typed errors | ✓ SATISFIED | `Parse` exists, the happy path returns a live `Doc`, and typed error paths are covered by `parser_test.go`. |
| `API-09` | `Parser.Close()` and `Doc.Close()` are idempotent; use-after-close returns `ErrClosed` | ✓ SATISFIED | Double-close and use-after-close tests passed. |
| `API-10` | `ParserPool` wraps `sync.Pool` with Get/Put and finalization; goroutine-per-parser is documented | ✓ SATISFIED | `ParserPool` implementation exists and `docs/concurrency.md` documents the intended model. |
| `API-11` | Finalizers warn in test builds and stay silent in production | ✓ SATISFIED | `finalizer_prod.go` and `finalizer_testbuild.go` implement the split; helper-process tests passed. |
| `API-12` | Typed error vars wrap the FFI error codes | ✓ SATISFIED | Sentinel errors plus structured wrapper exist in `errors.go`, and tests exercise `errors.Is` against them. |
| `DOC-03` | Godoc on every exported type/function in `purejson` (Phase 3 portion) | ✓ PARTIAL SATISFIED | Exported docs were added for the full Phase 3 surface in `purejson.go`, `errors.go`, `parser.go`, `doc.go`, `element.go`, and `pool.go`; Phase 4 completes this requirement for the symbols added there. |
| `DOC-04` | `docs/concurrency.md` documents the parser/doc/pool contract | ✓ SATISFIED | `docs/concurrency.md` contains the invariant, shareability, pool pattern, rejection rules, and leak warnings. |

No orphaned Phase 3 requirements were found: the union of `requirements:` across `03-01-PLAN.md`, `03-02-PLAN.md`, `03-03-PLAN.md`, `03-04-PLAN.md`, and `03-05-PLAN.md` maps cleanly to the nine Phase 3 requirement IDs in `.planning/REQUIREMENTS.md`.

### Disconfirmation Pass Notes

- The original 03-04 task text assumed branch-local `workflow_dispatch`, but GitHub rejects dispatch for workflow files absent from the default branch. The implemented `push` trigger plus explicit branch/run-id polling preserves the requirement-level outcome and was observed passing end to end.
- Rust still emits the pre-existing `input_storage` dead-code warning from `src/runtime/registry.rs` during regression runs. This warning predates Phase 3 and does not invalidate the Go wrapper goal.

### Gaps Summary

No blocking gaps found. The phase goal is achieved: the Go wrapper exposes the required Phase 3 public surface, preserves the intended lifecycle and typed-error semantics, documents the concurrency contract, and has both fresh local proof and a verified five-platform remote wrapper-smoke run.

---
_Verified: 2026-04-16T08:37:57Z_  
_Verifier: Codex (direct phase verification)_  
