---
gsd_state_version: 1.0
milestone: v0.1
milestone_name: Release
status: ready_to_plan
stopped_at: Completed 05-06
last_updated: "2026-04-20T12:28:36.979Z"
last_activity: 2026-04-20
progress:
  total_phases: 10
  completed_phases: 6
  total_plans: 22
  completed_plans: 22
  percent: 60
---

# Project State

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-15)

**Core value:** Replace `encoding/json` + `any` in parse-heavy Go workloads with a >=3x faster, precision-preserving parser that does not require cgo at consumer build time.
**Current focus:** Phase 06 — ci-release-matrix-platform-coverage

## Current Position

Phase: 06 (ci-release-matrix-platform-coverage) — READY TO PLAN
Plan: Not started
Status: Ready to plan
Last activity: 2026-04-20
Shipping: Phase 05 complete — bootstrap pipeline wired end-to-end (human UAT partial: E2E against live R2+GH requires Phase 06 CI-05)

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 32
- Average duration: 11.2m
- Total execution time: 1.3 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| Phase 01 | 3 | 28m | 9.3m |
| Phase 02 | 3 | 39m | 13.0m |
| 03 | 5 | - | - |
| 04 | 5 | - | - |
| 05 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: 04-01, 04-02, 04-03, 04-04, 04-05
- Trend: Stable

| Phase 04 P01 | 16m | 2 tasks | 7 files |
| Phase 04-full-typed-accessor-surface P02 | 8m | 2 tasks | 2 files |
| Phase 04 P03 | 4m | 2 tasks | 8 files |
| Phase 04-full-typed-accessor-surface P04 | 8m | 2 tasks | 3 files |
| Phase 04-full-typed-accessor-surface P05 | 11m | 2 tasks | 7 files |
| Phase 05 P01 | 3min | 2 tasks | 9 files |
| Phase 05 P02 | 7min | 2 tasks | 6 files |
| Phase 05 P03 | 3min | 1 tasks | 3 files |
| Phase 05 P04 | 8min | 1 tasks | 3 files |
| Phase Phase 05 PP05 | 5min | 2 tasks | 7 files |
| Phase Phase 05 PP06 | 9min | 2 tasks tasks | 5 files files |

## Accumulated Context

### Decisions

Decisions are logged in `.planning/PROJECT.md`. Recent decisions affecting current work:

- [Phase 02] Build the native shim from vendored simdjson `v4.6.1` through `build.rs` and `cc`, without manual kernel-selection flags.
- [Phase 02] Keep parser/doc handles generation-checked and store padded Rust-owned input alongside live docs.
- [Phase 02] Treat observed `windows-smoke` success as part of the exit gate, not just workflow YAML presence.
- [Phase 02] Keep the fallback-kernel override hidden behind test-only environment variables instead of exposing new public ABI controls.
- [Phase 03] Use branch-scoped push observation for wrapper smoke because GitHub cannot dispatch a workflow file that exists only on a non-default branch.
- [Phase 04]: Lock descendant views to PSDJROOT/PSDJDESC with doc+json_index transport and registry validation.
- [Phase 04]: Keep string copy-out ownership in Rust and free only through pure_simdjson_bytes_free.
- [Phase 04]: Use defer-safe purego string cleanup via BytesFree immediately after successful native reads.
- [Phase 04-full-typed-accessor-surface]: Public ElementType numerically mirrors ffi.ValueKind so Type() preserves the exact int64/uint64/float64 split.
- [Phase 04-full-typed-accessor-surface]: GetFloat64 rejects lossy integral conversions in the Go wrapper because native get_double rounds large int64/uint64 values silently.
- [Phase 04-full-typed-accessor-surface]: Integers larger than uint64 max are locked as parse-time ErrInvalidJSON cases because simdjson rejects them before GetUint64 can run.
- [Phase 04]: Iterator tags are locked as AR/OB and every iterator call rejects unknown tags or reserved bits before traversal continues.
- [Phase 04]: Array/object iterator progress stays inline as current and end tape indexes because the public ABI has no iterator free hook.
- [Phase 04-full-typed-accessor-surface]: ObjectIter.Next decodes key views through ElementGetString so Key only returns copied Go strings.
- [Phase 04-full-typed-accessor-surface]: Object.GetStringField stays as GetField plus GetString composition to preserve primitive missing/null/wrong-type semantics without new ABI.
- [Phase 04]: Document the final v0.1 purejson surface only in package docs and examples; do not preview bootstrap or On-Demand behavior.
- [Phase 04]: Lock the numeric boundary contract explicitly: max-int64+1 -> ErrNumberOutOfRange, 1e20 -> ErrWrongType, 9007199254740993 -> ErrPrecisionLoss.
- [Phase 04]: Use a recursive FuzzParseThenGetString DOM walk to validate copied Go strings across successful object and array paths.
- [Phase 05]: Canonical error sentinels (ErrChecksumMismatch, ErrAllSourcesFailed, ErrNoChecksum) live only in internal/bootstrap/errors.go; root errors.go re-exports via pointer alias so errors.Is matches both paths.
- [Phase 05]: GitHub release asset names are platform-tagged (libpure_simdjson-<goos>-<goarch>.ext, pure_simdjson-<goos>-<goarch>-msvc.dll) to avoid flat-namespace collision; cache filename stays platform-independent under <os>-<arch>/ directory in R2.
- [Phase 05]: ChecksumKey helper exported from internal/bootstrap so the Plan 05 CLI (separate cmd/ package) can reuse the Checksums map key format without exposing the map layout.
- [Phase 05]: PURE_SIMDJSON_CACHE_DIR env var takes precedence over os.UserCacheDir in defaultCacheDir so ephemeral-HOME CI runners and t.Setenv+t.TempDir test suites can self-isolate (L2 review resolution).
- [Phase 05]: When os.UserCacheDir fails, fall back to a UID-scoped 0700 subdirectory under os.TempDir (pure-simdjson-<uid>) instead of the bare TempDir path so the cache is never world-writable (L6 + DIST-05 spirit).
- [Phase 05]: BootstrapSync memoizes failures for 30 seconds via a package-level sync.Mutex-guarded cache so blocked-network NewParser() calls short-circuit after the first ladder exhausts; TTL is not configurable in v0.1 (M2 review resolution).
- [Phase 05]: Test seams for the external bootstrap_test package live in internal/bootstrap/export_test.go (compiled only during go test) — re-exports resolveConfig, withHTTPClient, withGitHubBaseURL, defaultCacheDir, and ResetBootstrapFailureCacheForTest (M3 review resolution).
- [Phase 05]: User-Agent 'pure-simdjson-go/v<Version>' is stamped on every outbound HTTP request in download.go so R2/GitHub server-side telemetry can identify the library and version (L3 review resolution).
- [Phase 05]: BootstrapSync checks ctx.Err() BEFORE consulting the failure-memoization cache, so a cancelled ctx returns ctx.Err() even when a memoized failure exists; config errors (bad mirror URL) are NOT memoized because they are caller bugs, not network state.
- [Phase 05]: downloadWithRetry now distinguishes per-URL fatal (404 -> skip remaining retries for that URL, try next URL) from ladder-fatal (checksum/no-checksum/HTTPS-downgrade -> abort all URLs); Fault Injection Matrix item 9 (R2 404 -> GH fallback fires) requires this separation.
- [Phase 05]: internal/bootstrap/export_test.go additionally re-exports r2ArtifactURL, githubArtifactURL, githubAssetName so URL-construction tests assert the exact wire format instead of rebuilding the format string in-test (prevents test/production drift).
- [Phase 05]: library_loading.go::activeLibrary switches to double-checked locking (M1). libraryMu is held only for the fast-path cached-pointer read and the recheck-insert block; resolveLibraryPath, loadLibrary, and ffi.Bind run outside the mutex so first-run bootstrap no longer serializes concurrent NewParser callers on one caller's network bandwidth.
- [Phase 05]: resolveLibraryPath implements a 4-stage chain (env override -> cache hit -> BootstrapSync -> cache hit after bootstrap). Every successful return is absolute via filepath.Abs or bootstrap.CachePath, preserving the DIST-09 Windows full-path invariant. Bootstrap failures are wrapped with a "set PURE_SIMDJSON_LIB_PATH to bypass" hint (D-21) and %w preserves errors.Is matching via the H2 pointer-identity aliasing locked in Plan 01.
- [Phase 05]: bootstrap error translation uses no adapter. Plan 01 H2 aliased root purejson.ErrChecksumMismatch etc. to bootstrap sentinels via pointer identity, so fmt.Errorf("...: %w", err) propagates the full errors.Is chain across the loader boundary without a translation helper.
- [Phase 05]: testmain_test.go seeds PURE_SIMDJSON_LIB_PATH to target/release/<libname> when the cargo artefact is present, so Phase 3/4 tests that relied on implicit candidate discovery continue to pass after Plan 05-04 deleted libraryCandidates(). Tests that exercise the new resolution chain override with t.Setenv to "".
- [Phase 05]: cmd/pure-simdjson-bootstrap is a thin wrapper only — CLI owns no download/checksum/URL logic; cobra flags translate 1:1 to bootstrap.BootstrapOption setters so internal/bootstrap remains the single source of truth.
- [Phase 05]: fetch --all-platforms emits per-platform progress ('fetching <os>/<arch>...' + '  ok <os>/<arch>') to stderr before/after each BootstrapSync call (L4) so users never perceive the CLI as silently hung during multi-platform downloads.
- [Phase 05]: verify supports --dest <dir> and --all-platforms (M4) so offline bundles produced by 'fetch --all-platforms --dest X' can be round-trip verified via 'verify --all-platforms --dest X'; the layout under <dest> is v<Version>/<os>-<arch>/<libname>, identical to what fetch writes.
- [Phase 05]: CLI root command uses SilenceUsage: true and SilenceErrors: true; errors render exactly once via main() to stderr with exit code 1, preventing cobra from drowning error messages in the default usage dump (D-28).
- [Phase 05]: Integration tests mutate the package-level bootstrap.Checksums map via a t.Cleanup-restored override so httptest-served fake bytes can hash-match; the map is empty in dev (pre-CI-05), the override is the M3-spirit test seam for the cmd/ package.
- [Phase 05]: downloadOnce captures the temp path in a local createdTmp before the cleanup defer; named-return-zeroing on early return "", "", err otherwise leaves orphan *.tmp files in the cache dir on every cancelled/failed bootstrap (Plan 06 Rule 1 fix surfaced by TestBootstrapSyncCancellation).
- [Phase 05]: T-05-04 redirect-downgrade defence is covered by three layered tests — TestRedirectDowngradeUnit (calls rejectHTTPSDowngrade with synthetic via-chain), TestRedirectDowngradeWired (asserts newHTTPClient().CheckRedirect points at the policy), and the existing TestHTTPSDowngradeRejected end-to-end via httptest.NewTLSServer; preferred over a brittle two-server httptest topology.
- [Phase 05]: Cross-process flock test (Fault Injection Matrix item 8) is intentionally NOT added in v0.1 — flock/LockFileEx correctness is OS code, pure-onnx ships without one, and subprocess tests are flaky on Windows CI; rationale comment lives at TestConcurrentBootstrap so future contributors find it without re-discovering.

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 02 advisory] Review whether parse-time `simdjson::UNSUPPORTED_ARCHITECTURE` should map to `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` instead of `PURE_SIMDJSON_ERR_INTERNAL`.
- [Phase 02 advisory] Clean up stale public comments for now-live exports and decide whether `last_error_offset` should remain sentinel-only or surface real offsets.

## Session Continuity

Last session: 2026-04-20T12:28:36.969Z
Stopped at: Completed 05-06
Resume file: None

**Planned Phase:** 05 (bootstrap-distribution) — 6 plans — 2026-04-20T11:21:15.134Z
