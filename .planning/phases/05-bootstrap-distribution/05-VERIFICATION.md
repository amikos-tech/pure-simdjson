---
phase: 05-bootstrap-distribution
verified: 2026-04-20T15:45:00Z
status: human_needed
score: 10/10 must-haves verified (automated); 2 items require human/Phase-6 validation
overrides_applied: 0
human_verification:
  - test: "Fresh-machine end-to-end bootstrap against live R2 + GitHub Releases"
    expected: "rm -rf ~/Library/Caches/pure-simdjson; NewParser() downloads from releases.amikos.tech, verifies SHA-256, caches, and parses successfully on all 5 target platforms"
    why_human: "Checksums.go map is intentionally empty during development; live artifacts + real SHA-256 values are populated by CI-05 in Phase 6. Cannot be exercised in this phase without the CI release pipeline. This matches Success Criterion 1 from ROADMAP.md."
  - test: "Corporate-firewall workaround against a real proxy blocking releases.amikos.tech"
    expected: "With PURE_SIMDJSON_BINARY_MIRROR set to internal mirror, bootstrap succeeds; with GH fallback reachable, R2-blocked environment still bootstraps"
    why_human: "Requires corporate network environment and cannot be automated meaningfully in CI. Documented in 05-VALIDATION.md Manual-Only Verifications section. Deferred to Phase 7 or user-reported validation."
---

# Phase 5: Bootstrap + Distribution Verification Report

**Phase Goal:** First `NewParser()` on a fresh machine downloads, verifies, caches, and loads the right shared library — or honors a user-provided path for air-gapped deployments. Ten distinct distribution requirements all met.

**Verified:** 2026-04-20T15:45:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                           | Status     | Evidence                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | R2 URL layout `releases.amikos.tech/pure-simdjson/v<version>/<os>-<arch>/lib<name>.<ext>` with GitHub Releases mirror           | VERIFIED   | `url.go:10` defaultR2BaseURL + `url.go:11` defaultGitHubBaseURL; `r2ArtifactURL` and `githubArtifactURL` construct per-platform URLs; `TestURLConstruction` + `TestGitHubArtifactURL` assert layouts |
| 2   | `internal/bootstrap/checksums.go` with SHA-256 entry per artifact; verification before any dlopen                               | VERIFIED   | `checksums.go:7` defines `var Checksums map[string]string` keyed by ChecksumKey format; `download.go:182-194` verifies SHA-256 before `atomicInstall`; cache-hit path does NOT re-verify (D-04). Map is empty pending CI-05 — documented and tested via `TestNoChecksumReturnsSentinel` |
| 3   | `BootstrapSync(ctx)` callable as preflight; auto-trigger on first `NewParser()` if cache miss                                   | VERIFIED   | `bootstrap.go:190` `BootstrapSync(ctx, opts...)`; `library_loading.go:141` calls `bootstrap.BootstrapSync(ctx)` on cache miss with internal 5-min timeout |
| 4   | OS-user-cache-dir storage with `0700` perms on unix                                                                             | VERIFIED   | `cache.go:29-42` `defaultCacheDir` uses `os.UserCacheDir()`; `cache.go:40` + `bootstrap.go:227` + `download.go:301` all use `0700`; `TestCacheDirPerms` + `TestCacheDirTempDirFallbackPerms` assert 0700 |
| 5   | `PURE_SIMDJSON_LIB_PATH` env override + `PURE_SIMDJSON_BINARY_MIRROR` env override                                              | VERIFIED   | `library_loading.go:120` reads `PURE_SIMDJSON_LIB_PATH`; `bootstrap.go:26` + `resolveConfig:118` read `PURE_SIMDJSON_BINARY_MIRROR`; `TestLibPathEnvBypassesDownload` + `TestMirrorOverride` + `TestResolveConfigEnvMirror` |
| 6   | `cmd/pure-simdjson-bootstrap` CLI for offline pre-fetch                                                                         | VERIFIED   | `cmd/pure-simdjson-bootstrap/{main,fetch,verify,platforms,version}.go` — four cobra verbs build and run; `go build ./cmd/pure-simdjson-bootstrap` succeeds; `TestFetchCmd` + `TestFetchCmdSingleTarget` + `TestVerifyAllPlatformsDest` |
| 7   | Windows `LoadLibrary` always uses full absolute path — never bare filename                                                      | VERIFIED   | `library_loading.go:120-128` calls `filepath.Abs` on env input; `library_loading.go:133-135` cache path built from `bootstrap.CachePath` (absolute); `library_windows.go:11-12` `windows.LoadLibrary(path)` receives the resolved absolute path; `TestResolveLibraryPathAbsolute` asserts every attempted path is absolute |
| 8   | `docs/bootstrap.md` covering env vars, mirror setup, air-gapped flow, corporate firewall workaround                             | VERIFIED   | `docs/bootstrap.md:36-40` env var table with all 4 vars; §Air-Gapped Deployment (L46); §Corporate Firewall / Custom Mirror (L67); §Verifying Artifact Integrity (Cosign) (L136); §Retry and Error Behavior (L166) |
| 9   | Cosign keyless OIDC signing; verification documented as optional but recommended (DIST-10 docs-only per D-29/D-30)              | VERIFIED   | `docs/bootstrap.md:136-164` cosign verify-blob recipe using keyless OIDC; no Go code imports sigstore (`grep -r sigstore *.go` returns no matches). Matches D-29/D-30 docs-only decision. **Note:** actual signing-in-CI is deferred to Phase 6 per ROADMAP.md §Phase 6 must-haves |
| 10  | Retry with exponential backoff; honor context cancellation; surface clear errors on egress block                                | VERIFIED   | `download.go:107-129` `sleepWithJitter` implements Full-Jitter D-13 with ctx-aware select; `download.go:210-247` `downloadWithRetry` drives R2→GH ladder; `errors.go:17-20` `ErrAllSourcesFailed` + `download.go:245-246` wraps with `PURE_SIMDJSON_LIB_PATH` hint; `TestBootstrapSyncCancellation` + `TestBootstrapSyncCtxCancelDuringSleep` + `TestRetryOn429ThenSuccess` + `TestFallback404R2Then200GH` |

**Score:** 10/10 truths verified (automated)

### Required Artifacts

| Artifact                                       | Expected                                                | Status     | Details                                                                                                                                           |
| ---------------------------------------------- | ------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/bootstrap/version.go`                | compile-time Version constant                           | VERIFIED   | `const Version = "0.1.0"`; imported by 9 sites (cache.go, download.go, bootstrap.go, library_loading.go, cmd/*)                                   |
| `internal/bootstrap/checksums.go`              | SHA-256 map placeholder                                 | VERIFIED   | `var Checksums = map[string]string{}` with 5 commented example entries; referenced from download.go:183 + verify.go:99+111                         |
| `internal/bootstrap/errors.go`                 | canonical sentinels                                     | VERIFIED   | `ErrChecksumMismatch`, `ErrAllSourcesFailed`, `ErrNoChecksum`; re-exported via aliasing in root `errors.go:48-52`                                 |
| `internal/bootstrap/url.go`                    | URL + platform + ChecksumKey                            | VERIFIED   | All expected exports present; `SupportedPlatforms` is the 5-entry target matrix; `validateBaseURL` rejects non-HTTPS for non-loopback             |
| `internal/bootstrap/cache.go`                  | cache layout + flock + atomic rename                    | VERIFIED   | `CachePath`, `withProcessFileLock`, `atomicInstall`, `defaultCacheDir`; PURE_SIMDJSON_CACHE_DIR env override wired (L2)                           |
| `internal/bootstrap/download.go`               | HTTP + retry + SHA-256                                  | VERIFIED   | Full-Jitter backoff + User-Agent + CheckRedirect + one-pass SHA-256 via MultiWriter + oversize response guard + cleanup of orphan tmp files       |
| `internal/bootstrap/bootstrap.go`              | BootstrapSync orchestrator                              | VERIFIED   | Signature matches D-03; resolveConfig reads env → applies opts; failure memoization TTL=30s (M2); all BootstrapOption setters defined              |
| `internal/bootstrap/bootstrap_lock_unix.go`    | unix flock                                              | VERIFIED   | `unix.Flock(LOCK_EX\|LOCK_NB)`; EWOULDBLOCK/EAGAIN classified as would-block                                                                       |
| `internal/bootstrap/bootstrap_lock_windows.go` | Windows LockFileEx                                      | VERIFIED   | `windows.LockFileEx` with LOCKFILE_EXCLUSIVE_LOCK\|LOCKFILE_FAIL_IMMEDIATELY; cross-compile `go vet` clean                                          |
| `internal/bootstrap/export_test.go`            | test seams                                              | VERIFIED   | ResolveConfig, WithHTTPClient, WithGitHubBaseURL, DefaultCacheDir, RegisterChecksumForTest, ResetBootstrapFailureCacheForTest, and 7 others        |
| `internal/bootstrap/bootstrap_test.go`         | full fault injection coverage                           | VERIFIED   | 34 Test functions; all Fault Injection Matrix items covered                                                                                        |
| `internal/bootstrap/cache_test.go`             | cache dir + atomic install tests                        | VERIFIED   | 7 Test functions covering perms, env override, fallback, lock, atomic install                                                                      |
| `library_loading.go`                           | 4-stage resolveLibraryPath + M1 double-checked locking  | VERIFIED   | resolveLibraryPath chains env → cache → BootstrapSync → cache; activeLibrary runs path resolution + dlopen OUTSIDE libraryMu (lines 48-97)         |
| `library_loading_test.go`                      | DIST-09 + DIST-06 + M1 coverage                         | VERIFIED   | 8 Test functions; TestResolveLibraryPathAbsolute (3 sub-tests) + TestLibPathEnvBypassesDownload + TestActiveLibraryLockScope                       |
| `errors.go` (root)                             | bootstrap sentinel re-exports (H2 pointer identity)     | VERIFIED   | `var ErrChecksumMismatch = bootstrap.ErrChecksumMismatch` etc. — aliasing preserves pointer identity for errors.Is across package boundary         |
| `cmd/pure-simdjson-bootstrap/main.go`          | cobra root with 4 verbs                                 | VERIFIED   | SilenceUsage=true + SilenceErrors=true; adds newFetchCmd, newVerifyCmd, newPlatformsCmd, newVersionCmd                                             |
| `cmd/pure-simdjson-bootstrap/fetch.go`         | fetch verb + L4 per-platform progress                   | VERIFIED   | All 5 flags wired (--all-platforms, --target, --dest, --version, --mirror); progress lines to stderr                                                |
| `cmd/pure-simdjson-bootstrap/verify.go`        | verify --all-platforms --dest (M4)                      | VERIFIED   | --dest directs artifactPath; --all-platforms iterates SupportedPlatforms; returns ErrChecksumMismatch on mismatch                                  |
| `cmd/pure-simdjson-bootstrap/platforms.go`     | platforms verb lists 5 targets with cache status        | VERIFIED   | Iterates SupportedPlatforms; os.Stat on CachePath produces cached/missing indicator                                                                |
| `cmd/pure-simdjson-bootstrap/version.go`       | version verb with ReadBuildInfo                         | VERIFIED   | runtime.Version() + debug.ReadBuildInfo() + bootstrap.Version all printed                                                                          |
| `cmd/pure-simdjson-bootstrap/fetch_test.go`    | fetch integration via httptest                          | VERIFIED   | TestFetchCmd + TestFetchCmdSingleTarget                                                                                                             |
| `cmd/pure-simdjson-bootstrap/verify_test.go`   | verify --all-platforms --dest integration               | VERIFIED   | TestVerifyAllPlatformsDest + TestVerifyAllPlatformsDestMismatchFails + TestVerifyCurrentPlatformDefault                                             |
| `docs/bootstrap.md`                            | env vars + mirror + air-gapped + corporate + cosign     | VERIFIED   | 230 lines covering all required flows + Supported Platforms table + Testing/Release Scope honesty note (L5)                                        |
| `go.mod`                                       | cobra dep pinned to v1.10.2                             | VERIFIED   | `github.com/spf13/cobra v1.10.2 // indirect` present; `golang.org/x/sys v0.31.0` direct                                                            |

### Key Link Verification

| From                                             | To                                          | Via                                                                          | Status     | Details                                                                                                       |
| ------------------------------------------------ | ------------------------------------------- | ---------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------- |
| `library_loading.go::resolveLibraryPath`         | `internal/bootstrap/bootstrap.go`            | calls `bootstrap.BootstrapSync(ctx)` outside libraryMu (M1)                  | WIRED      | library_loading.go:141 calls BootstrapSync with ctx from context.WithTimeout(5m); outside libraryMu per M1    |
| `library_loading.go::activeLibrary`              | `library_loading.go::resolveLibraryPath`     | resolves path OUTSIDE libraryMu; re-check under lock before install          | WIRED      | lines 48-97 — fast path reads cached under lock then releases; slow path runs resolve + dlopen without lock    |
| `internal/bootstrap/bootstrap.go::ensureArtifact` | `internal/bootstrap/cache.go`                | withProcessFileLock + artifactCachePath                                      | WIRED      | bootstrap.go:219-238 calls both; flock guards install so concurrent downloads collapse                         |
| `internal/bootstrap/bootstrap.go::ensureArtifact` | `internal/bootstrap/download.go`             | calls downloadAndVerify inside lock                                          | WIRED      | bootstrap.go:237 calls downloadAndVerify(ctx, cfg, cachePath)                                                 |
| `internal/bootstrap/download.go::downloadOnce`   | `internal/bootstrap/url.go`                  | r2ArtifactURL + githubArtifactURL                                            | WIRED      | download.go:168 + 172 construct primary/fallback URLs; ChecksumKey on line 182                                 |
| `internal/bootstrap/download.go::downloadAndVerify` | `internal/bootstrap/checksums.go`         | Checksums[key] lookup before atomic install                                  | WIRED      | download.go:183-194; mismatch returns permanent ErrChecksumMismatch                                             |
| `cmd/pure-simdjson-bootstrap/fetch.go`           | `internal/bootstrap/bootstrap.go`            | runFetch calls BootstrapSync with WithDest/WithMirror/WithVersion/WithTarget | WIRED      | fetch.go:78/85 invokes BootstrapSync; buildOpts maps flags 1:1 to BootstrapOption setters                       |
| `cmd/pure-simdjson-bootstrap/verify.go`          | `internal/bootstrap/checksums.go`            | Checksums + ChecksumKey for SHA-256 round-trip                               | WIRED      | verify.go:98-124 compares hashed file against Checksums[key]                                                    |
| `cmd/pure-simdjson-bootstrap/platforms.go`       | `internal/bootstrap/url.go`                  | SupportedPlatforms + CachePath                                               | WIRED      | platforms.go:25-33 iterates SupportedPlatforms, stats each CachePath                                             |
| `errors.go` (root)                               | `internal/bootstrap/errors.go`               | pointer-identity re-exports (H2 aliasing)                                    | WIRED      | `var ErrChecksumMismatch = bootstrap.ErrChecksumMismatch` — errors.Is matches across both package surfaces    |
| `internal/bootstrap/bootstrap_test.go`           | `internal/bootstrap/export_test.go`          | test seams consumed by external bootstrap_test package (M3)                  | WIRED      | bootstrap_test.go uses bootstrap.ResolveConfig, bootstrap.WithHTTPClient, bootstrap.ResetBootstrapFailureCacheForTest |

### Data-Flow Trace (Level 4)

| Artifact                                    | Data Variable       | Source                                                      | Produces Real Data | Status                                                                                       |
| ------------------------------------------- | ------------------- | ----------------------------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------- |
| `library_loading.go::activeLibrary`         | loaded library handle | purego.Dlopen(resolveLibraryPath()) → bindings              | YES (when artifact present) | FLOWING — load chain returns real path; dlopen + Bind produce real handle when artifact exists |
| `bootstrap.go::BootstrapSync`               | artifact bytes      | HTTP GET (R2 or GH) → SHA-256 verify → atomic rename        | YES (when live)    | FLOWING at runtime with real endpoints — httptest proves pipeline. Real artifacts pending Checksums population (CI-05). Documented in docs/bootstrap.md §Testing and Release Scope |
| `cmd/.../platforms.go::runPlatforms`        | cached indicator    | os.Stat(CachePath) per platform                             | YES                | FLOWING — executable output confirmed: 5 platforms listed with missing/cached status          |
| `cmd/.../version.go::runVersion`            | version info        | bootstrap.Version + runtime.Version() + ReadBuildInfo       | YES                | FLOWING — executable output confirmed: `library: 0.1.0` + go version + module info           |
| `checksums.go::Checksums`                   | SHA-256 digests     | populated at release time by CI-05                          | NO (by design)     | STATIC placeholder — matches ROADMAP.md §Phase 6 must-haves "SHA-256 manifest computed in CI"; tested for ErrNoChecksum behavior |

**Note:** The empty `Checksums` map is not a defect — it is the intentional Phase 5 / Phase 6 seam documented in CONTEXT D-08 and 05-06-SUMMARY L5. Phase 5 wires the verification pipeline; Phase 6 populates the map at release time.

### Behavioral Spot-Checks

| Behavior                             | Command                                                          | Result                                                  | Status |
| ------------------------------------ | ---------------------------------------------------------------- | ------------------------------------------------------- | ------ |
| CLI builds                           | `go build ./cmd/pure-simdjson-bootstrap`                         | exits 0                                                 | PASS   |
| CLI help lists 4 verbs               | `./pure-simdjson-bootstrap --help`                               | fetch, verify, platforms, version                       | PASS   |
| CLI version subcommand runs          | `./pure-simdjson-bootstrap version`                              | `library: 0.1.0\ngo: go1.26.2\nmodule: ...`             | PASS   |
| CLI platforms subcommand runs        | `./pure-simdjson-bootstrap platforms`                            | 5 lines: linux/amd64..windows/amd64 with missing/cached | PASS   |
| `go vet ./...` clean                 | `go vet ./...`                                                   | no output                                               | PASS   |
| Bootstrap unit tests pass            | `go test ./internal/bootstrap/... -count=1 -timeout 60s`         | `ok` 16.2s                                              | PASS   |
| CLI tests pass                       | `go test ./cmd/pure-simdjson-bootstrap/... -count=1 -timeout 60s` | `ok` 0.6s                                               | PASS   |
| Full test suite passes               | `go test ./... -count=1 -timeout 120s`                           | all 4 packages `ok`                                     | PASS   |
| No sigstore/cosign imports in Go     | `grep -r sigstore *.go` / `grep -r cosign *.go`                  | No matches found                                        | PASS (DIST-10 docs-only) |
| docs/bootstrap.md contains env var   | `grep -q 'PURE_SIMDJSON_LIB_PATH' docs/bootstrap.md`             | match found                                             | PASS (DOC-05) |

### Requirements Coverage

Cross-referenced PLAN frontmatter `requirements:` fields against REQUIREMENTS.md.

| Requirement | Source Plan(s)         | Description                                                                                                         | Status       | Evidence                                                                                                          |
| ----------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------- |
| DIST-01     | 05-01, 05-03           | Pre-built libs uploaded to CloudFlare R2 at `releases.amikos.tech/pure-simdjson/v<version>/<os>-<arch>/lib<name>.<ext>` | SATISFIED    | `url.go::r2ArtifactURL` constructs exact layout; `TestURLConstruction` asserts for all 5 platforms                 |
| DIST-02     | 05-01, 05-02, 05-03    | GitHub Releases mirror as fallback                                                                                  | SATISFIED    | `url.go::githubArtifactURL` + `download.go::downloadWithRetry` ladder; `TestFallback404R2Then200GH` + `TestFallback503R2Then200GH` |
| DIST-03     | 05-01, 05-02, 05-03    | SHA-256 table in Go source; embedded in `internal/bootstrap/checksums.go`; generated at release time                | SATISFIED    | `checksums.go` exists with ChecksumKey-format placeholder map; verified before `atomicInstall`; `TestChecksumMismatchIsPermanent` |
| DIST-04     | 05-02, 05-03           | `BootstrapSync(ctx)` downloads, verifies, caches; callable for preflight                                            | SATISFIED    | `bootstrap.go:190`; `TestBootstrapSync` + `TestBootstrapSyncCancellation` + `TestConcurrentBootstrap`              |
| DIST-05     | 05-02, 05-03, 05-04    | Library auto-downloaded on first `NewParser()`; cached to OS user-cache-dir with 0700 perms on unix                 | SATISFIED    | `library_loading.go::resolveLibraryPath` stage 3; `cache.go::defaultCacheDir`; `TestCacheDirPerms`                  |
| DIST-06     | 05-01, 05-04           | `PURE_SIMDJSON_LIB_PATH` overrides download entirely                                                                | SATISFIED    | `library_loading.go:120-128`; `TestLibPathEnvBypassesDownload` asserts no network call                             |
| DIST-07     | 05-02, 05-03           | `PURE_SIMDJSON_BINARY_MIRROR` overrides R2 base URL                                                                 | SATISFIED    | `bootstrap.go::resolveConfig:118`; `TestMirrorOverride` proves download hits override URL                          |
| DIST-08     | 05-05                  | `cmd/pure-simdjson-bootstrap` CLI pre-downloads artifacts                                                           | SATISFIED    | 4-verb cobra CLI; `TestFetchCmd` confirms --all-platforms downloads each platform's artifact                       |
| DIST-09     | 05-01, 05-04, 05-06    | Windows `LoadLibrary` uses full path                                                                                | SATISFIED    | resolveLibraryPath returns absolute path only; `TestResolveLibraryPathAbsolute` (3 sub-tests) asserts every attempted path is absolute |
| DIST-10     | 05-01, 05-06           | Release artifacts cosign-signed with keyless OIDC; verification documented but optional                             | SATISFIED (docs-only portion) | `docs/bootstrap.md:136-164` cosign recipe; no sigstore Go imports. **Actual signing-in-CI deferred to Phase 6** per ROADMAP §Phase 6 must-haves and 05-CONTEXT D-29/D-30. |
| DOC-05      | 05-06                  | `docs/bootstrap.md` covers env vars, mirror setup, air-gapped install flow                                          | SATISFIED    | 230 lines covering all required flows + Supported Platforms table + corporate firewall + cosign recipe + Phase 6 honesty note |

**Orphaned Requirements:** NONE. All DIST-01..10 + DOC-05 present in at least one plan's `requirements:` field. No requirement mapped to Phase 5 in REQUIREMENTS.md is missing from any plan.

### Anti-Patterns Found

Identified via grep on files listed in summaries. Categorized per review and verifier rules.

| File                                     | Line      | Pattern                                                                                      | Severity   | Impact                                                                                                       |
| ---------------------------------------- | --------- | -------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------ |
| `internal/bootstrap/cache.go`            | 90-92     | `nextLogAt` bookkeeping exists but no log statement fires (WR-01 from review)                | Warning    | Silent 2-minute lock wait UX gap. Non-blocking; documented in 05-REVIEW.md as post-v0.1 polish               |
| `internal/bootstrap/bootstrap.go`        | 199-212   | `BootstrapSync` failure memoization ignores config key (WR-02 from review)                   | Warning    | Stale error leaks across config changes within 30s TTL. Non-blocking; documented post-v0.1 polish             |
| `internal/bootstrap/cache.go`            | 61-103    | Lock-acquire loop is not context-cancellable (WR-03 from review)                             | Warning    | 2-min max-wait even on ctx.Done. Non-blocking; documented post-v0.1 polish                                    |
| `cmd/pure-simdjson-bootstrap/fetch_test.go` | 33-40  | `hits` counter modified from httptest goroutines without sync (WR-04 from review)            | Warning    | Sharp edge for future `--parallel` flag in Phase 6. Non-blocking; test-only                                   |
| `internal/bootstrap/checksums.go`        | 7         | Map intentionally empty: `var Checksums = map[string]string{}`                               | Info       | By design — populated by CI-05 in Phase 6; tests prove ErrNoChecksum path; documented in 05-CONTEXT D-08 and docs/bootstrap.md |
| `internal/bootstrap/download.go`         | 37        | Unused constant `bootstrapRetryBaseMS = 500` (IN-01)                                         | Info       | Dead code; review recommends deleting or wiring to sleepWithJitter                                            |

**No blockers.** All 4 warnings are post-v0.1 polish explicitly accepted in 05-REVIEW.md summary: "Most impactful: WR-01 and WR-02. Both are post-v0.1 polish, not ship-blockers." The empty Checksums map is intentional design per 05-CONTEXT D-08.

### Human Verification Required

**1. Fresh-machine end-to-end bootstrap against live infrastructure**

**Test:** `rm -rf ~/Library/Caches/pure-simdjson` (or OS-equivalent), then run a program that calls `NewParser()` on a fresh machine with internet access on each of the 5 target platforms.

**Expected:** Artifact downloads from `releases.amikos.tech`, SHA-256 verifies, caches to OS user-cache-dir with 0700 perms (unix), and parser opens successfully.

**Why human:** The `Checksums` map is intentionally empty during development. Live R2 artifacts + real SHA-256 digests are generated by CI-05 in **Phase 6**. Until CI-05 lands and populates `checksums.go`, the fresh-machine E2E flow cannot be exercised. This is Success Criterion 1 from ROADMAP.md §Phase 5 and is explicitly deferred to Phase 6 per `docs/bootstrap.md:212-223` §Testing and Release Scope.

**2. Corporate-firewall workaround against a real proxy**

**Test:** Configure a network environment that blocks `releases.amikos.tech`. Set `PURE_SIMDJSON_BINARY_MIRROR=<internal-mirror>` and confirm bootstrap succeeds.

**Expected:** Download hits the internal mirror URL; if that fails, GH fallback fires unless `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` is set.

**Why human:** Requires corporate network environment; cannot be automated meaningfully in CI. Documented in 05-VALIDATION.md §Manual-Only Verifications row 3: "Deferred to Phase 7 or user-reported validation."

### Gaps Summary

No gaps block the goal. The phase's 10 must-have truths are all VERIFIED with evidence:

- All 10 DIST-* requirements and DOC-05 implemented with test coverage.
- The 4-stage loader chain (env → cache → bootstrap → cache) is wired with M1 double-checked locking so downloads run outside `libraryMu`.
- The 24 declared artifacts all exist, are substantive (non-stub), are wired end-to-end, and (where applicable) have data flowing through them at runtime.
- 11 key links verified WIRED.
- 34 bootstrap unit tests + 3 CLI test files + 8 library_loading tests pass under `go test ./... -count=1 -timeout 120s`.
- The 4 advisory warnings from 05-REVIEW.md are documented post-v0.1 polish items and do not affect the phase goal.

**Remaining items are environmental and require real infrastructure/human testing in Phase 6:**

1. Live-endpoint E2E bootstrap — gated on CI-05 populating `checksums.go`.
2. Corporate-firewall verification — gated on access to a real corporate network.

Both items are pre-documented as deferred to Phase 6/7 in CONTEXT.md, VALIDATION.md, and docs/bootstrap.md. The automated verification surface for Phase 5 is complete.

---

_Verified: 2026-04-20T15:45:00Z_
_Verifier: Claude (gsd-verifier)_
