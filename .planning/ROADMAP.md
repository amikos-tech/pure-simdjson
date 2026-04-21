# Roadmap: pure-simdjson v0.1 (MVP)

**Created:** 2026-04-14
**Milestone:** v0.1 — DOM API on five platforms with bootstrap, benchmarks, and ad-hoc-signed release
**Granularity:** standard (5-8 phases, 3-5 plans each)
**Parallelization:** enabled
**Coverage:** 64/64 v1 requirements mapped

## Core Value Anchor

Replace `encoding/json` + `any` in parse-heavy Go workloads with a >=3x faster, precision-preserving JSON parser that does not require cgo at the consumer build. The non-negotiable happy path: `[]byte -> Doc -> typed accessors` on linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.

## Phases

- [x] **Phase 1: FFI Contract Design** — Lock the C ABI, error-code space, handle format, and ownership rules in a committed contract document before any code is written
- [x] **Phase 2: Rust Shim + Minimal Parse Path** — Build the Rust cdylib with vendored simdjson and the smallest end-to-end parse path (parser_new -> parse -> doc_root -> get_int64)
- [x] **Phase 3: Go Public API + purego Happy Path** — Wire Go's `purejson` package to the shim with handle lifecycle, ParserPool, typed errors, and one accessor as smoke test
- [x] **Phase 4: Full Typed Accessor Surface** — Complete the DOM accessor surface (uint64/float64/string/bool/null) and cursor-pull iteration over arrays and objects
- [x] **Phase 5: Bootstrap + Distribution** — Implement R2 download with GitHub fallback, SHA-256 verification, OS cache, env overrides, and the bootstrap CLI
- [ ] **Phase 6: CI Release Matrix + Platform Coverage** — Build, sign, and publish artifacts for all five targets plus Alpine smoke-test, with cosign and ad-hoc macOS codesign
- [ ] **Phase 7: Benchmarks + v0.1 Release** — Three-tier benchmark harness vs `encoding/json`, `simdjson-go`, `sonic`, `goccy/go-json`; correctness oracle; documentation; v0.1 tag

## Phase Details

### Phase 1: FFI Contract Design

**Goal:** A committed, reviewable C ABI contract that gates every implementation phase. Addresses 7 of 12 P0 pitfalls at the contract level so they cannot be re-introduced later.

**Depends on:** Nothing (gates everything else)

**Requirements:** FFI-01, FFI-02, FFI-03, FFI-04, FFI-05, FFI-06, FFI-07, FFI-08, DOC-02

**Must-haves:**
- Single committed `include/pure_simdjson.h` (cbindgen-generated, diff-checked in CI)
- Error-code numeric space defined; every export returns `int32` with out-params for results (pitfall 8)
- Generation-stamped opaque handles `{slot: u32, gen: u32}` packed into `u64` (pitfall 1, 10)
- Ownership rule: input buffer copied into Rust-owned padded arena on every parse (pitfall 2, 3)
- `ffi_wrap` helper contract: `catch_unwind` + error-code return on every export; `panic = "abort"` (pitfall 4)
- C++ `.get(err)` form mandated; grep-CI rule documented (pitfall 5)
- ABI version function `get_abi_version()` and Go-side `^0.1.x` constraint
- No-float/int-mixing rule documented for Windows/arm64 calling convention (pitfall 9)
- Cursor/pull iteration signatures fixed (no Go callbacks across FFI) (pitfall 7)
- Number-accessor split documented: distinct int64/uint64/float64 with overflow + precision-loss errors (pitfall 12)

**Nice-to-haves:**
- Worked examples in the contract doc showing happy-path and error-path call sequences
- Annotated invariants table mapping each rule back to the originating pitfall

**Success Criteria** (what must be TRUE):
1. `docs/ffi-contract.md` is committed and reviewable; every subsequent phase can be implemented from it without ambiguity
2. Every C ABI function signature in the contract obeys: `int32` return, no struct-by-value, no float/int mixing in the same arglist
3. Handle format, error-code space, and ABI version handshake are fully specified with byte-level layouts
4. The 7 P0 pitfalls (1, 2, 7, 8, 9, 10, 12) each map to a documented contract rule that prevents recurrence

**Plans:** 3 plans

Plans:
- [x] `01-01-PLAN.md` — Bootstrap the ABI-source crate and reproducible `cbindgen` header pipeline
- [x] `01-02-PLAN.md` — Define the stable C ABI surface in `src/lib.rs` and regenerate `include/pure_simdjson.h`
- [x] `01-03-PLAN.md` — Write the normative contract and static verification checks for header drift and ABI rules

**Research flag:** YES — spawn `/gsd-research-phase` during planning. No prior `pure-*` library has compiled C++ inside a Rust build, and the contract is expensive to walk back after code lands. Research should validate signature shapes against purego v0.10.0 across all five targets.

---

### Phase 2: Rust Shim + Minimal Parse Path

**Goal:** A buildable Rust cdylib + staticlib that compiles vendored simdjson via the `cc` crate and exposes the smallest end-to-end parse path. Proves the FFI contract holds in code on at least one platform.

**Depends on:** Phase 1 (contract must be locked)

**Requirements:** SHIM-01, SHIM-02, SHIM-03, SHIM-04, SHIM-05, SHIM-06, SHIM-07

**Must-haves:**
- Crate `pure_simdjson` with `crate-type = ["cdylib", "staticlib"]`
- `build.rs` driving `cc` crate over simdjson v4.6.1 single-file amalgamation; `-static-libstdc++ -static-libgcc`; C++17
- simdjson vendored as a git submodule at `third_party/simdjson` with pinned commit
- `ffi_wrap` helper (shipped in Phase 1) applied to every export (catch_unwind + error code)
- All 17 exports from SHIM-06 implemented with stub-or-real semantics so the header compiles
- Runtime kernel dispatch left to simdjson auto-detection (no `-march=native`)
- `get_implementation_name()` exposes selected kernel; `parser_new` returns `ERR_CPU_UNSUPPORTED` when `fallback` is selected (with documented bypass for testing)
- cbindgen-generated header committed and matches the Phase 1 contract (CI diff check)
- Per-platform FFI smoke test runs on at least linux/amd64 and one of darwin or windows as Phase 2 exit gate (pitfall: Windows MSVC simdjson compile is risky — verify early per ARCHITECTURE.md guidance)

**Nice-to-haves:**
- Stress test for input-arena padding under `GOGC=1`-style GC pressure simulated from a test harness
- Allocation-failure injection test for the C++ exception path

**Success Criteria** (what must be TRUE):
1. `cargo build --release` produces `libpure_simdjson.{so,dylib,dll}` on linux/amd64, darwin (one arch), and windows/amd64
2. A C-language smoke test loads the library, calls `parser_new -> parser_parse -> doc_root -> element_get_int64` on a literal `42` document, and gets back 42
3. `get_abi_version()` returns the value committed in Phase 1's contract
4. CI diff check confirms the cbindgen-generated header matches the committed contract header byte-for-byte
5. On a CPU forced to the `fallback` kernel, `parser_new` returns `ERR_CPU_UNSUPPORTED`

**Plans:** 3 plans

Plans:
- [x] `02-01-PLAN.md` — Build the vendored simdjson shim and wire the minimal ABI surface
- [x] `02-02-PLAN.md` — Implement the real minimal parse path with lifecycle-safe runtime state
- [x] `02-03-PLAN.md` — Add smoke verification and fallback-kernel proof paths

---

### Phase 3: Go Public API + purego Happy Path

**Goal:** Go consumers can `NewParser() -> Parse(data) -> doc.Root().GetInt64()` with full lifecycle safety, typed errors, and the ParserPool concurrency primitive in place. Single accessor proves the bind, not the breadth.

**Depends on:** Phase 2 (Go can't bind nonexistent symbols)

**Requirements:** API-01, API-02, API-03, API-09, API-10, API-11, API-12, DOC-03 (partial — exported types from this phase), DOC-04

**Must-haves:**
- Package `purejson` with `Parser`, `Doc`, `Element`, `Array`, `Object`, `ParserPool` types declared (Element/Array/Object stubs ok, filled in Phase 4)
- `library_unix.go` / `library_windows.go` / `library_loading.go` for purego symbol binding (use `RegisterFunc`, never `SyscallN` — pitfall 27)
- `NewParser()`, `Parser.Parse([]byte) (*Doc, error)`, `Parser.Close()`, `Doc.Close()` — all idempotent on double-close (pitfall 10)
- `Element.GetInt64()` as the single smoke accessor
- `ParserPool` on `sync.Pool` with Get/Put; documented goroutine-per-parser model
- `runtime.SetFinalizer` on `Parser`/`Doc` logging a leak warning in test builds; silent in production (pitfall 31)
- All typed error vars from API-12 wired to FFI error codes
- `docs/concurrency.md` written explaining per-parser single-doc invariant and ParserPool pattern
- Per-platform FFI smoke test green on all five targets (Phase 6 will productionize CI; this phase needs at least manual verification)

**Nice-to-haves:**
- Race-detector test for `TestDoubleClose` and concurrent `Parser` access returning `ErrParserBusy`
- Godoc rendering check (output committed under `docs/` for review)

**Success Criteria** (what must be TRUE):
1. `go test ./...` with `-race` passes on linux/amd64, darwin/arm64, and windows/amd64 with the locally-built shim
2. Calling `Close()` twice on a Parser or Doc returns nil; calling any method after Close returns `ErrClosed`
3. ABI version mismatch between Go constants and the loaded shim returns `ErrABIVersionMismatch` at `NewParser()`
4. A program that allocates 10,000 Parsers and forgets to Close produces leak warnings in test builds (and only in test builds)
5. ParserPool round-trips a Parser across goroutines without `ErrParserBusy` when used as documented

**Plans:** 5 plans

Plans:
- [x] `03-01-PLAN.md` — Create the Go module, deterministic loader, and low-level purego happy-path bindings
- [x] `03-02-PLAN.md` — Implement parser/doc lifecycle, typed happy-path access, and semantic tests
- [x] `03-03-PLAN.md` — Add `ParserPool` plus build-tag-specific finalizer behavior and lifetime tests
- [x] `03-04-PLAN.md` — Add local verification targets and the observed five-target wrapper-smoke workflow
- [x] `03-05-PLAN.md` — Document the public API and concurrency contract for the exact Phase 3 surface

**UI hint:** no

---

### Phase 4: Full Typed Accessor Surface

**Goal:** Complete the DOM accessor surface so consumers can extract every JSON value type and walk arrays and objects via cursor-pull iteration. This is the v0.1 API users will see.

**Depends on:** Phase 3 (needs the binding skeleton). Can run in parallel with Phase 5.

**Requirements:** API-04, API-05, API-06, API-07, API-08, DOC-03 (full coverage)

**Must-haves:**
- `Element.GetUint64() (uint64, error)`, `GetFloat64() (float64, error)` with `ErrNumberOutOfRange` and `ErrPrecisionLoss` wired to FFI codes
- `Element.GetString() (string, error)` returning a Go string copy (zero-copy view deferred to v0.2)
- `Element.GetBool() (bool, error)`, `IsNull() bool`, `Type() ElementType`
- `Array.Iter() *ArrayIter` and `Object.Iter() *ObjectIter` with `Next()`, `Value()`, `Key()` — Go drives, no FFI callbacks
- `Object.GetField(key string) (Element, error)` for direct lookup
- Boundary-value tests for number accessors (max int64, max uint64, values requiring float64 precision loss) — pitfall 12
- Malformed-UTF-8 fuzz corpus in tests; validating accessors return `ErrInvalidJSON` not garbage strings — pitfall 11
- Godoc on every exported symbol (DOC-03 complete)

**Nice-to-haves:**
- Batch accessor `Object.GetStringField(name) (string, error)` in one FFI call (architecture note: this is the predicted first bottleneck)
- Number-classification helper exposed via `Element.NumberKind()` for callers who want to branch before extraction

**Success Criteria** (what must be TRUE):
1. Every JSON value type defined by the spec can be extracted from a parsed Doc with a typed Go accessor that returns the precise type
2. Iterating an array or object of N elements requires no Go callbacks across FFI and produces results in iteration order
3. Calling `GetInt64()` on `9223372036854775808` returns `ErrNumberOutOfRange`; calling `GetInt64()` on `1e20` returns `ErrWrongType`; calling `GetFloat64()` on `9007199254740993` returns `ErrPrecisionLoss`
4. Parsing a buffer with invalid UTF-8 in a string value surfaces `ErrInvalidJSON` from the accessor, not a corrupted Go string
5. Godoc renders cleanly with examples on every exported type

**Plans:** 5 plans

Plans:
- [x] `04-01-PLAN.md` — Lock descendant-view encoding and activate the native scalar/string/bool/null substrate
- [x] `04-02-PLAN.md` — Expose the public scalar/type/string/bool/null API and semantic tests
- [x] `04-03-PLAN.md` — Implement inline-only native iterator/object-lookup transport and hidden Go mirrors
- [x] `04-04-PLAN.md` — Expose the public scanner-style array/object traversal and field helpers
- [x] `04-05-PLAN.md` — Close Phase 4 with DOC-03, examples, fuzz entrypoint, and full semantic verification

**UI hint:** no

---

### Phase 5: Bootstrap + Distribution

**Goal:** First `NewParser()` on a fresh machine downloads, verifies, caches, and loads the right shared library — or honors a user-provided path for air-gapped deployments. Ten distinct distribution requirements all met.

**Depends on:** Phase 3 (needs the loader). Can run in parallel with Phase 4.

**Requirements:** DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06, DIST-07, DIST-08, DIST-09, DIST-10, DOC-05

**Must-haves:**
- R2 URL layout `releases.amikos.tech/pure-simdjson/v<version>/<os>-<arch>/lib<name>.<ext>` with GitHub Releases mirror
- `internal/bootstrap/checksums.go` with a SHA-256 entry per artifact; verification before any `dlopen` (pitfall 17)
- `BootstrapSync(ctx)` callable as preflight; auto-trigger on first `NewParser()` if cache miss
- OS-user-cache-dir storage with `0700` perms on unix (pitfall 16 — non-world-writable)
- `PURE_SIMDJSON_LIB_PATH` env override (full path) and `PURE_SIMDJSON_BINARY_MIRROR` env override (R2 base URL)
- `cmd/pure-simdjson-bootstrap` CLI for offline pre-fetch
- Windows `LoadLibrary` always uses full absolute path — never bare filename (pitfall 16)
- `docs/bootstrap.md` covering env vars, mirror setup, air-gapped flow, corporate firewall workaround
- Cosign keyless OIDC signing; verification documented as optional but recommended (DIST-10)
- Retry with exponential backoff on download; honor context cancellation; surface clear errors on egress block (pitfall 20)

**Nice-to-haves:**
- Cold-start benchmark to characterize first-parse latency including download (pitfall 19)
- Integration test that runs in an air-gapped Docker network with `PURE_SIMDJSON_LIB_PATH` set

**Success Criteria** (what must be TRUE):
1. On a fresh machine with internet, `NewParser()` downloads, verifies SHA-256, caches to OS user-cache dir, loads, and parses successfully
2. With `PURE_SIMDJSON_LIB_PATH` set to an absolute path, no network call is made and the specified library is loaded
3. With `PURE_SIMDJSON_BINARY_MIRROR` set to an alternate base URL, the download targets that mirror
4. A corrupted artifact (mismatched SHA-256) is rejected before `dlopen` with a clear error
5. `pure-simdjson-bootstrap` CLI pre-fetches all platform artifacts for offline distribution

**Plans:** 6 plans

Plans:
- [x] `05-01-PLAN.md` — Package scaffold: version/checksums/url/flock/error sentinels + cobra dep (Wave 1)
- [x] `05-02-PLAN.md` — HTTP download pipeline: cache layout, Full-Jitter retry, SHA-256 verify, BootstrapSync API (Wave 2)
- [x] `05-03-PLAN.md` — Bootstrap test suite: URL/cache unit tests + fault injection stubs (Wave 3)
- [x] `05-04-PLAN.md` — Loader integration: rewrite resolveLibraryPath 4-stage chain + activeLibrary double-checked locking, delete legacy candidates (Wave 4)
- [x] `05-05-PLAN.md` — Bootstrap CLI: four cobra verbs + fetch integration test + verify --dest --all-platforms (Wave 4)
- [x] `05-06-PLAN.md` — Remaining fault injection tests + docs/bootstrap.md (Wave 5)

**UI hint:** no

---

### Phase 6: CI Release Matrix + Platform Coverage

**Goal:** A tag push produces signed, verified shared libraries for all five targets (plus an Alpine smoke-test signal) uploaded to R2 and GitHub Releases with a generated checksum manifest. CI is the only path to a release.

**Depends on:** Phases 2 + 3 + 4 + 5 (needs a complete codebase to build, smoke-test, bootstrap, and ship)

**Requirements:** PLAT-01, PLAT-02, PLAT-03, PLAT-04, PLAT-05, PLAT-06, CI-01, CI-02, CI-03, CI-04, CI-05, CI-06, CI-07

**Must-haves:**
- `.github/workflows/release-prepare.yml` prepares version/checksum source state before tagging, and `.github/workflows/release.yml` builds all five target artifacts on tag push
- linux/amd64 and linux/arm64: glibc baseline ≤ 2.17 (manylinux2014 base or equivalent); `objdump -T` verification step (PLAT-01, PLAT-02, pitfall 14)
- darwin/amd64 and darwin/arm64: macOS 11+; ad-hoc codesign via `codesign -s - --force`; thin per-arch (no `lipo`) (pitfall 15, 30)
- windows/amd64: MSVC toolchain; artifact named `pure_simdjson-msvc.dll`; long-paths enabled in CI (PLAT-05, pitfall 29)
- Per-platform FFI smoke test job: load the artifact, call every exported symbol once, parse one literal document — gate the release on this (CI-04, pitfall 8, 9)
- Alpine smoke-test job (`alpine:latest` container) loads via `PURE_SIMDJSON_LIB_PATH` with a user-built `.so`; documents the chosen musl strategy (PLAT-06, CI-07, pitfall 21)
- Cosign keyless OIDC signing on every artifact
- SHA-256 manifest computed in CI and committed back as `internal/bootstrap/checksums.go` in the tagged-commit path (CI-05)
- GitHub Release asset upload step renames platform binaries to their platform-tagged form (`libpure_simdjson-<goos>-<goarch>.ext` / `pure_simdjson-windows-amd64-msvc.dll`) per Phase 5 H1 contract to avoid flat-namespace collision (CI-05)
- Pre-tag prep flow handles version/checksum source updates; tag workflow generates release notes and publishes that exact prepared state (CI-06)
- `-static-libstdc++ -static-libgcc` verified via `nm` showing only `extern "C"` exports (pitfall 22)

**Nice-to-haves:**
- 64K-page-size arm64 runner integration test (pitfall 23)
- Reproducibility check: rebuild the same tag twice and diff the binaries

**Success Criteria** (what must be TRUE):
1. A tag push produces five signed, checksummed artifacts uploaded to both R2 and GitHub Releases without manual intervention
2. The per-platform FFI smoke test must pass on all five targets before any artifact is published
3. The Alpine smoke-test job runs on every release and produces a clear pass/fail signal for the documented musl strategy
4. `objdump -T` on the linux artifacts shows no glibc symbols newer than 2.17
5. macOS artifacts open without Gatekeeper blocking after the documented `xattr -d com.apple.quarantine` workaround

**Plans:** 6 plans

**Research flag:** YES — spawn `/gsd-research-phase` during planning. The musl/Alpine strategy (static-link-into-glibc-so vs ship `.a` with documented relink vs smoke-test-only with escape hatch) is unresolved per SUMMARY.md decision 5; manylinux vs zig-cc choice and final target matrix also need a focused study before CI is written.

Plans:
- [x] `06-01-PLAN.md` — Shared release tooling scaffold: composite actions, packaging helpers, and bootstrap-state generator
- [ ] `06-02-PLAN.md` — Linux GNU release builds in manylinux containers with glibc-floor proof
- [ ] `06-03-PLAN.md` — macOS and Windows release builds with codesign, long-path handling, and export verification
- [ ] `06-04-PLAN.md` — Native + Go packaged-artifact smoke gates, including Alpine escape-hatch validation
- [ ] `06-05-PLAN.md` — Release-prep and tag-publish workflows with checksum/tag coherence, cosign, and R2/GitHub publish
- [ ] `06-06-PLAN.md` — Release runbook, readiness gate, and repo-local release skill

---

### Phase 06.1: Fresh-machine end-to-end bootstrap UAT against live R2 + GitHub Releases (INSERTED)

**Goal:** Execute the Phase 5 human UAT that could not be exercised during Phase 5 because the `internal/bootstrap/checksums.go` map is populated only at release time by Phase 6 CI-05. On a fresh machine with `~/Library/Caches/pure-simdjson` cleared, `NewParser()` should download a real artifact from `releases.amikos.tech`, verify SHA-256 against the populated `Checksums` map, cache with 0700 perms, and parse a sample document successfully on each of the 5 target platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64). Validates Success Criterion 1 from ROADMAP.md.

**Requirements:** TBD — lifted from backlog item 999.4 after Phase 5 deferred the live-artifact bootstrap UAT pending Phase 6 CI-05. See `.planning/phases/05-bootstrap-distribution/05-HUMAN-UAT.md` for original context.
**Depends on:** Phase 6
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 06.1 to break down)

### Phase 7: Benchmarks + v0.1 Release

**Goal:** A credible, reproducible benchmark story documenting the >=3x speedup vs `encoding/json` + `any` claim, plus the README, changelog, license/notice, and the v0.1 tag.

**Depends on:** Phase 6 (credible benchmarks need the final `.so` on the final channel)

**Requirements:** BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05, BENCH-06, BENCH-07, DOC-01, DOC-06, DOC-07

**Must-haves:**
- Three-tier benchmark harness: full-parse walk, typed field extraction, selective-path placeholder for v0.2 (BENCH-01)
- Canonical corpus vendored from simdjson test data: `twitter.json`, `canada.json`, `citm_catalog.json`, `mesh.json`, `numbers.json` (BENCH-02)
- Baselines: `encoding/json` + `any`, `encoding/json` + struct, `minio/simdjson-go`, `bytedance/sonic`, `goccy/go-json` (BENCH-03)
- `benchstat` reports; cold-start (first `Parse` after `NewParser`) reported separately from warm (BENCH-04, pitfall 25)
- Native allocator stats reported alongside Go alloc counts (BENCH-05, pitfall 26)
- Correctness oracle: parse every file in simdjson's `jsontestsuite`; accept/reject must match upstream (BENCH-06)
- README documents the >=3x vs `encoding/json` + `any` headline number and within-2x of `minio/simdjson-go` on x86_64 (BENCH-07, DOC-01)
- `CHANGELOG.md` in Keep-a-Changelog format (DOC-06)
- `LICENSE` (MIT) and `NOTICE` (simdjson Apache-2.0) committed (DOC-07)
- v0.1 tag pushed; CI release pipeline runs to completion; bootstrap from a fresh machine works against the v0.1 tag

**Nice-to-haves:**
- Comparison-fairness review note documenting matched workloads across competitors (pitfall 24)
- `Kernel()` / `ImplementationName()` diagnostic output included in benchmark headers

**Success Criteria** (what must be TRUE):
1. `go test -bench=. -benchmem ./...` produces benchstat-formatted output showing >=3x speedup vs `encoding/json` + `any` on at least three of the five corpus files
2. The correctness oracle passes 100% of `jsontestsuite` accept/reject cases against the upstream expected results
3. README installation snippet works end-to-end on a fresh machine for at least linux/amd64, darwin/arm64, and windows/amd64
4. Cold-start and warm benchmarks are reported separately and both are reproducible across two runs
5. v0.1 is tagged, signed artifacts are live on R2 and GitHub Releases, and `BootstrapSync()` against the tag succeeds from a clean cache

**Plans:** TBD

---

## Parallelization Map

| Phase | Runs After | Can Parallelize With |
|-------|------------|----------------------|
| 1 | (start) | — |
| 2 | 1 | — |
| 3 | 2 | — |
| 4 | 3 | **5** |
| 5 | 3 | **4** |
| 6 | 2, 3, 4, 5 | — |
| 7 | 6 | — |

**Parallelization opportunity:** Phases 4 and 5 are the only cross-phase parallel pair — they touch independent code (Go accessor surface vs Go bootstrap pipeline) and both depend only on the Phase 3 binding skeleton. Within phases, plans should be designed to run in parallel where possible (e.g., Phase 6's per-platform builds, Phase 7's per-baseline benchmark suites).

## Research Flags

Per `research/SUMMARY.md`, two phases should spawn `/gsd-research-phase` during their planning step:

- **Phase 1 (FFI Contract Design)** — bespoke contract; no prior `pure-*` library has compiled C++ inside a Rust build, and the contract is expensive to walk back after code lands
- **Phase 6 (CI Release Matrix)** — musl/Alpine strategy, manylinux vs zig-cc, and the final target matrix all need a focused study before CI is committed

The other five phases follow established patterns from `pure-tokenizers`, `pure-onnx`, and `fast-distance` and should not need a research-phase step.

## Coverage Validation

| Category | Requirement IDs | Phase |
|----------|-----------------|-------|
| FFI Contract | FFI-01..08 (8) | 1 |
| Native Shim | SHIM-01..07 (7) | 2 |
| Go API (lifecycle) | API-01, API-02, API-03, API-09, API-10, API-11, API-12 (7) | 3 |
| Go API (accessors) | API-04, API-05, API-06, API-07, API-08 (5) | 4 |
| Distribution | DIST-01..10 (10) | 5 |
| Platform | PLAT-01..06 (6) | 6 |
| CI | CI-01..07 (7) | 6 |
| Benchmarks | BENCH-01..07 (7) | 7 |
| Docs | DOC-02 (1) | 1 |
| Docs | DOC-03 (partial), DOC-04 (2) | 3 |
| Docs | DOC-03 (full) | 4 |
| Docs | DOC-05 (1) | 5 |
| Docs | DOC-01, DOC-06, DOC-07 (3) | 7 |

**Total v1 requirements:** 64 (8 + 7 + 7 + 5 + 10 + 6 + 7 + 7 + 7 docs distributed)
**Mapped:** 64
**Orphans:** 0

## Out of Scope for v0.1

Tracked in `REQUIREMENTS.md` as v2 — explicitly deferred and will become a separate roadmap:

- **OD-01..04** (On-Demand API + JSON Pointer + JSONPath subset)
- **STREAM-01..03** (NDJSON streaming + parallel `iterate_many`)
- **ZC-01..02** (zero-copy string views + `ParsePinned` via `runtime.Pinner`)
- **DIAG-01..02** (kernel override + macOS notarization)

Out-of-scope items from PROJECT.md (JSON encoding, struct-reflection Unmarshal, full JSONPath, JSON Schema validation, cgo fallback, silent SIMD fallback, linux/arm 32-bit, in-place mutation, visitor callbacks, mmap input) are explicitly excluded from all phases.

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. FFI Contract Design | 3/3 | Complete | 2026-04-14 |
| 2. Rust Shim + Minimal Parse | 3/3 | Complete | 2026-04-15 |
| 3. Go API + purego Happy Path | 5/5 | Complete | 2026-04-16 |
| 4. Full Typed Accessor Surface | 5/5 | Complete | 2026-04-17 |
| 5. Bootstrap + Distribution | 6/6 | Complete | 2026-04-20 |
| 6. CI Release Matrix | 0/6 | Not started | — |
| 7. Benchmarks + v0.1 Release | 0/? | Not started | — |

Plan counts populated by `/gsd-plan-phase`.

## Backlog

Parking lot for ideas not yet scheduled. Promote with `/gsd-review-backlog`.

### Phase 999.1: Local pre-commit and pre-push verification hooks (BACKLOG)

**Goal:** [Captured for future planning] Add lefthook (or equivalent — e.g., `pre-commit`, git-native hooks) so `make verify-contract` and `make verify-docs` (and future lint/format gates) run locally before code reaches CI. Prevents drift between local and CI verification and catches header/doc regressions at the dev's machine instead of after push.

**Requirements:** TBD

**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.2: Unified cleanup-failure reporting surface (BACKLOG)

**Goal:** [Captured for future planning] Replace the ad-hoc cleanup-failure `eprintln!` dispatch in `src/runtime/mod.rs` and `src/runtime/registry.rs` with a deliberate reporting surface that can propagate diagnostics to Go/test callers without forcing unconditional stderr output from the Rust runtime.

**Requirements:** TBD

**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.3: Encapsulate exported internal/ffi layout types (BACKLOG)

**Goal:** [Captured for future planning] Reshape the exported `internal/ffi` layout carriers so purego bindings can preserve ABI/layout guarantees without exposing field-level coupling as de facto public API.

**Requirements:** TBD

**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.5: Corporate-firewall bootstrap workaround verification (BACKLOG)

**Goal:** [Captured for future planning] Verify the corporate-firewall bootstrap workaround under a real proxy that blocks `releases.amikos.tech`. Two scenarios: (a) `PURE_SIMDJSON_BINARY_MIRROR` points at an internal mirror and bootstrap succeeds; (b) mirror unset but GitHub Releases fallback is reachable, and bootstrap succeeds via the GH ladder. Cannot be automated meaningfully in CI — needs a real corporate network or proxy emulation. Deferred from Phase 5 HUMAN-UAT per `05-VALIDATION.md` Manual-Only Verifications. See `.planning/phases/05-bootstrap-distribution/05-HUMAN-UAT.md` for original context.

**Requirements:** TBD — promote when user-reported issue or internal corp-network testbed is available.

**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)

---
*Roadmap created: 2026-04-14 from PROJECT.md, REQUIREMENTS.md, and research/SUMMARY.md*
