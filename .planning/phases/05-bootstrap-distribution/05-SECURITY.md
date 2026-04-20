---
phase: 05
slug: bootstrap-distribution
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-20
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Consolidated from the six plan-level threat models (05-01 through 05-06) and verified by `gsd-security-auditor` against the executed implementation.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| CLI / library caller → `internal/bootstrap` | `WithMirror` option and CLI flags accept user-supplied URLs and paths | mirror URL string, cache dir path, lib path |
| `PURE_SIMDJSON_BINARY_MIRROR` env var | Attacker-controlled if environment is compromised | URL base |
| `PURE_SIMDJSON_CACHE_DIR` env var | Attacker-controlled; dictates where downloaded artifacts land on disk | filesystem path |
| `PURE_SIMDJSON_LIB_PATH` env var | Attacker-controlled; passed directly to `dlopen`/`LoadLibrary` | filesystem path |
| HTTP response body (R2 primary, GH fallback) | MITM-capable if a corporate TLS proxy / hostile network sits between client and origin | binary artifact bytes |
| Downloaded file → `dlopen` | File at rest in cache; OS filesystem is the trust boundary for persistence | native library bytes |
| `checksums.go` embedded table | Compiled into Go source at build time; attacker must compromise the release build to alter it | SHA-256 digests |
| `docs/bootstrap.md` cosign recipe | External tool (cosign) not invoked by any Go code | user-operated |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-05-01 | Tampering | `internal/bootstrap/download.go::downloadAndVerify` | mitigate | SHA-256 verified against compile-time `Checksums` before `atomicInstall`; mismatch → `ErrChecksumMismatch` (permanent, no retry, no dlopen). Evidence: download.go:182-196; `TestChecksumMismatchIsPermanent` PASS. | closed |
| T-05-02 | Tampering | `internal/bootstrap/cache.go` flock + `atomicInstall` | mitigate | Cache dirs created with `0700`; flock serialization via `unix.Flock`/`LockFileEx`; atomic rename prevents torn files. Evidence: cache.go:40,65,109,117; `TestCacheDirPerms`, `TestCacheDirTempDirFallbackPerms`, `TestConcurrentBootstrap` PASS. | closed |
| T-05-03 | Elevation of Privilege | `library_loading.go::resolveLibraryPath` + `cmd/pure-simdjson-bootstrap/verify.go` | mitigate (resolve) / accept (Stage 3 SHA-layer) | `filepath.Abs(envPath)` on env path; `bootstrap.CachePath` absolute by construction; CLI uses identical `CachePath`/`filepath.Join(dest, ...)`. Evidence: library_loading.go:121,133; verify.go:64-73; `TestResolveLibraryPathAbsolute` PASS. | closed |
| T-05-04 | Tampering | `internal/bootstrap/download.go::rejectHTTPSDowngrade` | mitigate | `http.Client.CheckRedirect` wired to `rejectHTTPSDowngrade`; blocks any HTTPS→HTTP hop as ladder-fatal. Evidence: download.go:62-67,70-83,87-102,254-265; `TestRedirectDowngradeUnit`, `TestRedirectDowngradeWired`, `TestHTTPSDowngradeRejected` PASS. | closed |
| T-05-05 | Information Disclosure | `internal/bootstrap/bootstrap.go::WithMirror` + `resolveConfig` | mitigate | `validateBaseURL` rejects HTTP for non-loopback hosts at BOTH entry points (option + env). Evidence: url.go:99-114; bootstrap.go:58,126; mirror-URL rejection regressions documented in 05-03-PLAN. | closed |
| T-05-06 | Tampering | `internal/bootstrap/checksums.go` + `download.go` GH fallback | mitigate | `downloadAndVerify` is URL-agnostic — primary (R2) and fallback (GH) flow through the same `io.MultiWriter` hashing pipeline and the same `Checksums` lookup. Evidence: download.go:162-197,210-247,323; `TestFallback404R2Then200GH`, `TestFallback503R2Then200GH` PASS. | closed |
| T-05-10 | Information Disclosure | `cmd/pure-simdjson-bootstrap/version.go` | accept | Prints only `bootstrap.Version`, `runtime.Version()`, and `debug.ReadBuildInfo().Main.Version` — no secrets, env vars, or build flags. Evidence: version.go:27-35. | closed |
| T-05-11 | Spoofing | `internal/bootstrap/url.go::githubArtifactURL` (H1) | mitigate | `githubAssetName` emits five pairwise-distinct platform-tagged filenames (`libpure_simdjson-<goos>-<goarch>.<ext>` + Windows MSVC variant). Evidence: url.go:48-62,79-87; `TestGitHubAssetNames` PASS (explicit pairwise-distinctness assertion). | closed |
| T-05-12 | Denial of Service | `internal/bootstrap/bootstrap.go` failure memoization (M2) | mitigate | 30-second TTL cache on bootstrap failures; blocked networks no longer stall every `NewParser()`. Evidence: bootstrap.go:33,152-180,199,210,213; `TestBootstrapFailureMemoized` PASS (second call <50 ms, HTTP hit-counter unchanged). | closed |
| T-05-13 | Elevation of Privilege | `internal/bootstrap/cache.go::defaultCacheDir` TempDir fallback (L6) | mitigate | UID-scoped `pure-simdjson-<uid>` subdirectory under `os.TempDir()` with `0700` perms — non-world-writable even when `UserCacheDir` is unavailable. Evidence: cache.go:29-42; `TestCacheDirTempDirFallbackPerms` PASS. | closed |
| T-05-14 | Denial of Service | `library_loading.go::activeLibrary` (M1) | mitigate | Double-checked locking: `resolveLibraryPath` + `loadLibrary` + `ffi.Bind` all run OUTSIDE `libraryMu`; final lock only guards the cached-pointer install. Evidence: library_loading.go:48-97; `TestActiveLibraryLockScope` PASS (source-inspection test fails if network call moves inside the mutex). | closed |
| T-05-15 | Tampering | `cmd/pure-simdjson-bootstrap/verify.go::runVerify --all-platforms --dest` (M4) | mitigate | Bundle-level integrity check — any single-platform substitution in an offline bundle fails the gate before the bundle is trusted. Evidence: verify.go:95-126; `TestVerifyAllPlatformsDest`, `TestVerifyAllPlatformsDestMismatchFails` PASS. | closed |
| DIST-10 | Tampering | CLI `verify` verb + `docs/bootstrap.md` cosign recipe | accept | No sigstore import anywhere in the module; cosign is docs-only per D-29/D-30; SHA-256 is the mandatory always-on integrity layer. Evidence: `grep -rn "sigstore\|cosign" . --include="*.go"` → 0 hits; docs/bootstrap.md:162-164 explicit acceptance paragraph. | closed |
| L5 | Tampering | `docs/bootstrap.md` §Testing and Release Scope | accept | Real end-to-end test against R2/GH deferred to Phase 6 CI-05; explicitly documented so consumers understand the v0.1 "verified" boundary. Evidence: docs/bootstrap.md:204-229; 05-06-SUMMARY.md:131-133. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-03 (Stage 3) | Stage-3 of `resolveLibraryPath` reads a cache path whose bytes were SHA-256-verified inside `BootstrapSync` (covered by T-05-01). Adding a second verification layer would duplicate the same check without increasing assurance. | author | 2026-04-20 |
| AR-05-02 | T-05-10 | `bootstrap version` prints only library version, Go toolchain version, and main-module build-info version. No secrets, paths, or environment values are surfaced. | author | 2026-04-20 |
| AR-05-03 | DIST-10 | Cosign verification is retained as a **docs-only** recipe (see `docs/bootstrap.md` §Verifying Artifact Integrity). No Go code in the module imports sigstore. SHA-256 against the compile-time `Checksums` table is the always-on integrity layer; cosign is an optional user-run supplemental verification for air-gapped or high-assurance consumers. Accepted per D-29/D-30. | author | 2026-04-20 |
| AR-05-04 | L5 | The v0.1 test suite exercises every code path via `httptest` (URL construction, retry cascade, GH fallback, SHA-256 verify, atomic rename, flock concurrency, ctx cancel, env overrides). End-to-end verification against real R2-hosted artifacts with populated `Checksums` digests is deferred to Phase 6 (CI-05 — release workflow + checksum backfill + optional cosign signing job). This boundary is declared in `docs/bootstrap.md:204-229`. | author | 2026-04-20 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-20 | 14 | 14 | 0 | gsd-security-auditor (State B — created from artifacts) |

### 2026-04-20 — Initial audit

- Input state: **B** (no prior SECURITY.md; six PLAN.md + six SUMMARY.md present).
- Consolidated 14 unique threats from the six plan-level threat registers (T-05-01 through T-05-15 plus DIST-10 and L5).
- Auditor verified each mitigation against implementation files with file:line citations and ran the phase test suite targeted at the mitigation regressions (`TestRedirectDowngrade*`, `TestChecksumMismatchIsPermanent`, `TestGitHubAssetNames`, `TestBootstrapFailureMemoized`, `TestFallback*R2Then200GH`, `TestConcurrentBootstrap`, `TestCacheDirPerms`, `TestCacheDirTempDirFallbackPerms`, `TestVerifyAllPlatformsDest*`, `TestResolveLibraryPathAbsolute`, `TestActiveLibraryLockScope`) — all pass.
- Gate check for DIST-10: `grep -rn "sigstore" . --include="*.go"` returned zero hits; docs-only acceptance stands.
- Result: `## SECURED` — 14/14 closed, no open threats, no escalations.
- Deferred (not a threat, tracked for Phase 6): `checksums.go` ships commented placeholders only; the `ErrNoChecksum` fail-closed path is tested and is the correct behaviour for a pre-release.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-20
