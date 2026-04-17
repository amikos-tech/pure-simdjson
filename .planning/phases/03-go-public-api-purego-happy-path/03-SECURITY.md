---
phase: 03
slug: go-public-api-purego-happy-path
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-16
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| repo-local file system -> purego loader | Library resolution must stay deterministic and avoid bare-name loading. | Absolute shared-library paths, native library handle |
| Go-owned buffers -> native string-copy helpers | Buffer-backed purego calls must not observe reclaimed Go memory. | Byte slices, temporary copy buffers |
| native ABI results -> Go errors | Native failures must map to the right sentinel without stale detail reuse. | Status codes, error messages, offsets |
| Go public API -> native parser/doc handles | Parser/doc lifecycle misuse must not silently corrupt native state. | Parser handles, doc handles, lifecycle state |
| Go-owned buffers and receiver objects -> purego calls | Receiver-owned handles and input slices must remain live across native calls. | Input JSON slices, parser/doc/element references |
| Go wrapper lifecycle -> native parser/doc handles | Pool and finalizer behavior must not leak or double-free native state. | Pooled parser instances, finalizer cleanup paths |
| local branch -> GitHub remote | Remote wrapper verification must require explicit authenticated operator action. | Git push, workflow run creation |
| workflow YAML -> actual proof | Workflow presence is not enough; named jobs must be observed green on a specific run. | Run IDs, job names, job conclusions |
| user-facing docs -> actual wrapper behavior | Docs must not teach unsafe parser sharing or wrong cleanup semantics. | Package contract, concurrency guidance |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-03-01-01 | T | local loader | mitigate | Resolve only explicit absolute paths from `PURE_SIMDJSON_LIB_PATH` or the ordered local `target` candidates; do not fall back to bare-name loading or recursive discovery. | closed |
| T-03-01-02 | T | purego string-copy helpers | mitigate | Keep Go-owned buffers and receiver-owned values alive immediately after native copy helpers return. | closed |
| T-03-01-03 | T | Go FFI mirrors | mitigate | Mirror the implemented header surface exactly before adding higher-level API logic. | closed |
| T-03-01-04 | I | structured error wrapping | mitigate | Build errors from the failing call's immediate diagnostics only and treat empty detail as absent. | closed |
| T-03-02-01 | T | parser/doc lifecycle | mitigate | Keep close semantics explicit, reject busy parser frees, and surface `ErrClosed` without hidden state repair. | closed |
| T-03-02-02 | T | ABI mismatch handling | mitigate | Probe ABI compatibility before parser allocation and keep a deterministic mismatch test hook. | closed |
| T-03-02-03 | T | purego liveness | mitigate | Call `runtime.KeepAlive(...)` after native calls that depend on input slices or receiver-owned handles. | closed |
| T-03-02-04 | I | structured error behavior | mitigate | Preserve `errors.Is` sentinel matching while keeping detail fields scoped to the failing call path. | closed |
| T-03-03-01 | T | `ParserPool` | mitigate | Reject nil, closed, and busy parsers at `Put()` instead of auto-closing or replacing them. | closed |
| T-03-03-02 | T | pool eviction | mitigate | Keep finalizers armed while parsers sit in `sync.Pool` and prove eviction cleanup with an explicit test. | closed |
| T-03-03-03 | I | finalizer instrumentation | mitigate | Build-tag leak warnings so only `purejson_testbuild` emits `purejson leak:` while production stays silent. | closed |
| T-03-04-01 | R | remote verification | mitigate | Require explicit `gh` authentication and a manual branch push to trigger remote wrapper proof. | closed |
| T-03-04-02 | T | workflow targeting | mitigate | Pin the intended Linux/macOS runners and configure Windows MSVC setup before build/test. | closed |
| T-03-04-03 | T | run observation | mitigate | Bind verification to the matching branch-scoped run ID captured after push instead of a latest-run heuristic. | closed |
| T-03-05-01 | I | package docs | mitigate | Make the import-path/package-name split explicit in package and symbol docs. | closed |
| T-03-05-02 | I | concurrency docs | mitigate | Document the single-doc invariant, pool rejection rules, and leak-warning split exactly as implemented. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Threat Flags

No explicit `## Threat Flags` sections were present in the Phase 03 summary files. The plan threat models remained the source of truth for this audit.

---

## Verification Notes

- `T-03-01-01` is closed by `library_loading.go`, which resolves `PURE_SIMDJSON_LIB_PATH` to an absolute path, enumerates only four ordered local `target/...` candidates, records attempted paths, and passes an explicit path into the platform loaders.
- `T-03-01-02`, `T-03-01-03`, `T-03-01-04`, `T-03-02-03`, and `T-03-02-04` are closed by `internal/ffi/bindings.go`, `parser.go`, `doc.go`, `element.go`, and `errors.go`, which use explicit `purego.RegisterFunc`, widespread `runtime.KeepAlive(...)`, exact error/symbol mirrors, and per-call diagnostic wrapping.
- `T-03-02-01` and `T-03-02-02` are closed by `parser.go` and `parser_test.go`: `NewParser()` checks ABI before parser allocation, busy/closed semantics stay explicit, and the mismatch path is exercised by `TestABIMismatchAtNewParser`.
- `T-03-03-01`, `T-03-03-02`, and `T-03-03-03` are closed by `pool.go`, `parser.go`, `doc.go`, `finalizer_prod.go`, `finalizer_testbuild.go`, `parser_test.go`, and `pool_test.go`; `ParserPool.Put()` rejects misuse, production finalizers stay silent, test-build finalizers emit `purejson leak:`, and eviction cleanup is covered by explicit tests.
- `T-03-04-01`, `T-03-04-02`, and `T-03-04-03` are closed by `.github/workflows/phase3-go-wrapper-smoke.yml` and `scripts/phase3-go-wrapper-smoke.sh`, which require `gh auth status`, push the current branch explicitly, capture a post-push timestamp, locate the matching branch run ID, and verify all five required job names concluded `success`.
- `T-03-05-01` and `T-03-05-02` are closed by `purejson.go`, `parser.go`, `doc.go`, `element.go`, `pool.go`, and `docs/concurrency.md`, which document the import contract, the one-live-doc invariant, pool rejection semantics, and the production vs `purejson_testbuild` leak-warning split.
- Fresh local evidence from this session: `make phase3-go-test`, `go test ./... -run '^TestLeakWarningSilentProd$'`, `go test ./... -tags purejson_testbuild -run '^TestLeakWarning(TestBuild|MassLeak10000)$'`, and `make phase3-go-race` all passed on 2026-04-16.
- Fresh remote evidence from this session: `gh run view 24500326284 --json name,headBranch,headSha,conclusion,jobs` confirmed the recorded branch-scoped proof for head `9e158a1c7b39812948bca23e84fcaf8b798b46a3` completed successfully with `linux-amd64-go-race`, `linux-arm64-go-race`, `darwin-amd64-go-race`, `darwin-arm64-go-race`, and `windows-amd64-go-race`.

---

## Accepted Risks Log

No accepted risks.

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-16 | 16 | 16 | 0 | Codex (direct audit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-16
