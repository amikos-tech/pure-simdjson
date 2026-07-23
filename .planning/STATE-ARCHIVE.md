# STATE Archive

Pruned entries from STATE.md. Recoverable but no longer loaded into agent context.

## Pruned 2026-07-23 (phases 1-8, kept recent 3)

### Decisions

- [Phase 07]: Benchmark helpers must load only the committed `testdata/bench/` and `testdata/jsontestsuite/` assets; Phase 7 runtime must not depend on `third_party/` paths or the network.
- [Phase 07]: `TestJSONTestSuiteOracle` treats `expectations.tsv` as the only runtime source of truth and fails on both missing and extra vendored case files before parsing.
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
- [Phase 07]: Benchmark fixtures must be loaded only by exact filename from testdata/bench so later plans cannot drift back to third_party or network inputs.
- [Phase 07]: The JSONTestSuite oracle uses expectations.tsv as the only runtime source of truth and fails on both missing and extra vendored case files.
- [Phase 07]: Tier 1 benchmarks use per-fixture top-level benchmark functions with comparator sub-benchmarks to keep names stable for benchstat and README reporting.
- [Phase 07]: Cold-start means first Parse after NewParser inside an already loaded process; bootstrap and download time stay out of this benchmark family.
- [Phase 07]: Comparator availability is registered once and split by build tags so unsupported libraries are omitted structurally with human-readable reasons.
- [Phase 07]: Native allocator telemetry is epoch-based: reset excludes pre-existing live allocations from later snapshots instead of claiming process-wide totals.
- [Phase 07]: The allocator stats surface remains diagnostic-only and is published strictly as reset/snapshot exports plus a fixed four-field struct.
- [Phase 07]: Header-audit verification must work both through Makefile rules and the planner's direct python3 tests/abi/check_header.py include/pure_simdjson.h command.
- [Phase 07]: Tier 2 uses shared schema structs across supported comparators; pure-simdjson reaches them through DOM traversal only.
- [Phase 07]: Tier 3 remains explicitly scoped as a DOM-era placeholder and does not imply a v0.1 On-Demand API.
- [Phase 07]: Tier 1 and cold/warm benchmark outputs publish native-bytes/op, native-allocs/op, and native-live-bytes beside Go benchmem data.
- [Phase 08]: `make verify-contract` passes `--rule no-internal-symbols` explicitly because its explicit rule list bypasses default header-audit rules.
- [Phase 08]: FastMaterializer oversized-literal parse-rejection tests use `18446744073709551616` as the current public `ErrInvalidJSON` fixture; larger BIGINT-style literals remain separate precision-loss behavior for later implementation plans.
- [Phase 08]: `psdj_internal_materialize_build` validates `ValueView` once in the Rust registry, then traverses a root or subtree into doc-owned native frame scratch guarded by `materialize_in_progress`.
- [Phase 08]: Oversized integer literals now normalize to parse-time `PURE_SIMDJSON_ERR_INVALID_JSON` at `psimdjson_parser_parse`, so the internal frame builder never exposes BIGINT nodes or partial frame spans.
- [Phase 08]: Go mirrors `psdj_internal_frame_t` as a 72-byte `ffi.InternalFrame`, binds `psdj_internal_materialize_build`, and consumes the borrowed frame slice without copying it in `internal/ffi`.
- [Phase 08]: `fastMaterializeElement` stays internal, holds `doc.mu` while consuming borrowed frames, copies keys/strings at the Go value boundary, and rejects leftover or under-consumed frame spans with `ErrInternal`.
- [Phase 08]: `Doc.isClosed()` now uses a non-blocking mutex check so fast-materializer contention surfaces `ErrParserBusy` instead of deadlocking before the `TryLock` guard.
- [Phase 08]: Tier 1 full and materialize-only benchmark helpers now delegate to `fastMaterializeElement`, with literal diagnostic row labels and an explicit no-cache comment preserving Phase 7 benchstat continuity.
- [Phase 08]: Native frame scratch growth must stay geometric; per-container reserve churn caused the first same-host Canada gate attempt to regress by orders of magnitude before the Phase 8 evidence rerun fixed it.

### Performance Metrics

| Phase 01 | 3 | 28m | 9.3m |
| Phase 02 | 3 | 39m | 13.0m |
| 03 | 5 | - | - |
| 04 | 5 | - | - |
| 05 | 6 | - | - |
| Phase 04 P01 | 16m | 2 tasks | 7 files |
| Phase 04-full-typed-accessor-surface P02 | 8m | 2 tasks | 2 files |
| Phase 04 P03 | 4m | 2 tasks | 8 files |
| Phase 04-full-typed-accessor-surface P04 | 8m | 2 tasks | 3 files |
| Phase 04-full-typed-accessor-surface P05 | 11m | 2 tasks | 7 files |
| Phase 05 P01 | 3min | 2 tasks | 9 files |
| Phase 05 P02 | 7min | 2 tasks | 6 files |
| Phase 05 P03 | 3min | 1 tasks | 3 files |
| Phase 05 P04 | 8min | 1 tasks | 3 files |
| Phase 06 P01 | 5min | 2 tasks | 10 files |
| Phase 06 P02 | 11min | 2 tasks | 5 files |
| Phase 06 P03 | 15min | 2 tasks | 5 files |
| Phase 06 P04 | 44min | 2 tasks | 8 files |
| Phase 06 P05 | 15min | 2 tasks | 6 files |
| Phase 06 P06 | 7min | 2 tasks | 4 files |
| Phase 07 P01 | 12 min | 2 tasks | 328 files |
| Phase 07 P02 | 15min | 2 tasks | 12 files |
| Phase 07 P03 | 20min | 2 tasks | 17 files |
| Phase 07 P04 | 4min | 2 tasks | 8 files |
| Phase 08 P01 | 8min | 2 tasks | 5 files |
| Phase 08 P02 | 12min | 2 tasks | 7 files |
| Phase 08 P03 | 9min | 2 tasks | 6 files |
| Phase 08 P04 | 6min | 2 tasks | 2 files |
| Phase 08 P05 | 29min | 2 tasks | 7 files |
