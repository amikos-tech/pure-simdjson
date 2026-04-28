---
gsd_state_version: 1.0
milestone: v0.1
milestone_name: "Tracked in `REQUIREMENTS.md` as v2 — explicitly deferred and will become a separate roadmap:"
status: "Phase 10 shipped -- PR #27"
stopped_at: Completed 09.1-01-PLAN.md
last_updated: "2026-04-27T15:02:46.549Z"
last_activity: 2026-04-28 -- Phase 10 shipped as PR #27
progress:
  total_phases: 16
  completed_phases: 11
  total_plans: 50
  completed_plans: 47
  percent: 94
---

# Project State

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-23)

**Core value:** Ship a precision-preserving, cgo-free simdjson DOM parser for Go with honest benchmark positioning: typed extraction and selective traversal are the primary story, while full `any` materialization is documented without overstating current wins.
**Current focus:** Phase 09.1 — bootstrap-artifact-and-abi-alignment-for-default-installs

## Current Position

Phase: 09.1 (bootstrap-artifact-and-abi-alignment-for-default-installs) — EXECUTING
Plan: 2 of 2
Status: Ready to execute
Last activity: 2026-04-28 -- Completed quick task: PR benchmark nice-to-haves
Shipping: Phase 07 PR: https://github.com/amikos-tech/pure-simdjson/pull/18. Phase 08 PR: https://github.com/amikos-tech/pure-simdjson/pull/19. Phase 09 PR: https://github.com/amikos-tech/pure-simdjson/pull/21. Phase 10 PR: https://github.com/amikos-tech/pure-simdjson/pull/27. `v0.1.0` remains the latest published tag. Phase 09.1 now owns bootstrap artifact/default-install alignment before any later release tag.

Progress: [█████████▊] 98%

## Performance Metrics

**Velocity:**

- Total plans completed: 35
- Average duration: 11.1m
- Total execution time: 1.4 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| Phase 01 | 3 | 28m | 9.3m |
| Phase 02 | 3 | 39m | 13.0m |
| 03 | 5 | - | - |
| 04 | 5 | - | - |
| 05 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: 08-03, 08-04, 08-05, 09-01, 09.1-01
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
| Phase 09 P01 | 7min | 2 tasks | 7 files |
| Phase 09.1 P01 | 4min | 2 tasks | 9 files |

## Accumulated Context

## Quick Tasks Completed

| Date | Slug | Summary |
|------|------|---------|
| 2026-04-24 | phase8-final-polish | Added executable depth-boundary fence, clarified ERR_INTERNAL split rationale at the ABI enum, expanded cross-ABI numeric comments, and rechecked benchmark gates. |
| 2026-04-24 | phase8-depth-doc-followup | Clarified depth-limit defense-in-depth docs, strengthened user-actionable enum comments, pinned the current accepted nesting boundary, and rechecked benchmark gates. |
| 2026-04-24 | phase8-followup-feedback | Added observable depth-limit status/sentinel coverage, tightened materializer comments, filled adversarial string-span coverage, and rechecked benchmark gates. |
| 2026-04-24 | phase8-pr-review-feedback | Applied Phase 8 PR review fixes for materializer depth guarding, optional-symbol/fallback observability, unsafe frame diagnostics, not-implemented telemetry status, span contract tests/docs, and benchmark regression checks. |
| 2026-04-24 | pr19-review-items-1-2-3-5 | Addressed PR #19 polish items 1/2/3/5: documented `InternalMaterializeBuild` frame-span lifecycle, expanded the LIFO defer ordering comment in the fast materializer, added native-side (Rust + C++) size asserts for `psdj_internal_frame_t` (field-width expression, 32-bit safe), and documented `psimdjson_test_hold_materialize_guard`'s by-design `PARSER_BUSY` return. Comments-and-asserts only — Tier 1 diagnostics benchstat shows no regression (B/op and allocs/op identical, geomean sec/op within noise). |
| 2026-04-27 | apply-pr-22-feedback-items-2-4-6-8-and-9 | Applied 5 of 9 PR #22 review items: bidirectional ABI sync comments between `internal/bootstrap/abi_assertion.go` and `scripts/release/check_bootstrap_abi_state.py`, fixed `semver_tuple` return-type to honor `tuple[int, int, int]` annotation, added `0.1.1` stale-version boundary test, documented + tested pre-release semver acceptance (`0.1.2-dev`), and added a clarifying comment on the layered `bootstrap.Version` check in `scripts/release/check_readiness.sh`. Items #1/#3/#5/#7 explicitly skipped per prior `/pr-feedback` analysis. |
| 2026-04-28 | pr-benchmark-review-feedback | Addressed Phase 10 PR benchmark feedback: empty benchmark captures now fail closed, parser metric-section handling is stricter, baseline cache save is success-only, workflow comments are clearer, asymmetric `NO_BASELINE=1` parsing was removed, and focused regression tests cover the new contracts. |
| 2026-04-28 | pr-benchmark-nice-to-haves | Added follow-up Phase 10 PR benchmark nice-to-have coverage: pinned the current non-`vs base` metric-header limitation, required clean `yq` stderr for workflow YAML smoke checks, and made stale-output replacement a true two-run orchestrator test. |

### Learning Extractions

| Date | Phase | Output |
|------|-------|--------|
| 2026-04-24 | 08 | `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-LEARNINGS.md` |
| 2026-04-24 | 09 | `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-LEARNINGS.md` |

### Roadmap Evolution

- Phase 06.1 inserted after Phase 06: Fresh-machine end-to-end bootstrap UAT against live R2 + GitHub Releases (promoted from backlog item 999.4)
- Phase 06.1 execution produced the public bootstrap wrapper, hosted-runner validation workflow, contract tests, and operator runbook updates, and was shipped in PR #17; hosted GitHub Actions execution remains pending
- Phase 07 is now planned as six plans: corpus/oracle foundation, Tier 1 + cold/warm harness, allocator telemetry surface, Tier 2/Tier 3 benchmark consumers, public docs/legal artifacts with committed evidence, and a closeout handoff that defers public patch-release work until the later benchmark-positioning phases
- Phase 07 completed on 2026-04-23 as a benchmark/docs/legal baseline rather than a forced patch release: README, methodology doc, results snapshot, changelog, LICENSE, and NOTICE are now committed, and the closeout explicitly routes Tier 1 ABI work to Phase 08 and benchmark/release recalibration to Phase 09
- Phase 07 learnings were extracted on 2026-04-23 into `.planning/phases/07-benchmarks-v0.1-release/07-LEARNINGS.md`, preserving benchmark positioning decisions, execution lessons, reusable patterns, and surprises for Phase 08 and Phase 09 planning
- Phase 08 added: Low-overhead DOM traversal ABI and specialized Go any materializer. This folds the old 999.6 DOM-materialization ABI idea into the active milestone after Tier 1 diagnostics showed materialization, not parse, dominates the current full-`any` path
- Phase 09 added: Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh. This phase exists to replace the invalidated BENCH-07 headline with a measured benchmark story after Phase 08 lands
- Phase 09.1 inserted after Phase 09: Bootstrap artifact and ABI alignment for default installs (URGENT)
- Phase 10 added: Lightweight PR benchmark regression signal, promoted from backlog item 999.8 and explicitly scoped to a cheap Tier 1/Tier 2/Tier 3 `pull_request` benchmark check rather than the heavier Phase 9 release-evidence capture
- Backlog items 999.6, 999.7, and 999.8 were retired from the parking lot: 999.6 is now active milestone work under Phase 08, 999.7's diagnostic split was implemented during Phase 07 investigation to justify the new direction, and 999.8 is now active milestone work under Phase 10

### Decisions

Decisions are logged in `.planning/PROJECT.md`. Recent decisions affecting current work:

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
- Pinned Rust setup lives in a local setup-rust composite action that reads rust-toolchain.toml directly.
- verify-shared-artifact hard-fails when native ABI or minimal_parse smoke commands are missing so export audits stay supplemental.
- Bootstrap release-state rewrites are tested in copied TemporaryDirectory workspaces so unittest never mutates the real repo.
- The shared build action now hands manylinux execution to scripts/release/build_linux_manylinux.sh so workflow YAML does not duplicate docker mount logic or arm64 page-size enforcement.
- linux/arm64 page-size proof runs as both an explicit workflow step and a builder-side guard; the prep workflow also uploads linux-arm64-pagesize.txt with the staged artifact bundle.
- verify_glibc_floor.sh derives the expected pure_simdjson export set from include/pure_simdjson.h instead of freezing a separate symbol list in CI.
- The darwin workflow matrix now carries the expected public asset names and asserts them after packaging, so the bootstrap naming contract is executable in CI.
- The windows release bundle preserves pure_simdjson.dll.lib and a dumpbin /DEPENDENTS report alongside the staged DLL so later plans can reuse that evidence without rebuilding.
- The shared release helpers now emit forward-slash artifact paths and Python-created temp directories so the same bash-based composite actions work on windows runners without a separate packaging path.
- CI-04 now runs through scripts/release/run_native_smoke.sh so every platform executes one shared audit -> ffi_export_surface.c -> minimal_parse.c sequence.
- Staged bootstrap smoke consumes one exact v<version>/<os>-<arch>/<libname> tree assembled from per-platform manifest rows and staged artifacts.
- Both staging jobs rewrite bootstrap release state from the combined manifest before go run so packaged-artifact smoke uses real checksum data.
- Release prep now rewrites version.go, checksums.go, and CHANGELOG.md on a release-prep/v<version> branch before any tag is created.
- Tag publication now starts with a verify-tag-source gate that rejects off-main tags and validates committed bootstrap source state before any build begins.
- The publish workflow signs and verifies the raw staged blobs first, then copies those bytes into flat GitHub Release asset names so R2 and GitHub Releases carry the same signed payload.
- docs/releases.md is the single human-readable source of truth for the Phase 6 release-prep -> main -> tag sequence, required repo configuration, artifact layout, and cosign verification commands.
- scripts/release/check_readiness.sh --strict reuses assert_prepared_state.py --check-source and adds origin/main ancestry checks instead of re-implementing release-state validation in shell.
- scripts/release/check_readiness.sh --strict now also delegates bootstrap/Go/Rust ABI source-state validation to scripts/release/check_bootstrap_abi_state.py using an explicit ABI_MINIMUM_VERSION policy table.
- bootstrap.Version is pinned to 0.1.2 for ABI 0x00010001; internal/bootstrap/checksums.go remains empty in source and runtime digest verification still resolves published SHA256SUMS.
- internal/bootstrap/abi_assertion.go provides a bidirectional compile-time array canary so go test ./internal/bootstrap fails if ffi.ABIVersion drifts from the ABI expected by bootstrap version 0.1.2.
- docs/bootstrap.md now points operators at the release runbook and mirrors the exact xattr Gatekeeper workaround, while Phase 06.1 owns the fresh-runner public validation boundary.
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

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 09.1] Plan 02 remains the release gate: land the prepared source on origin/main, run strict readiness there, publish v0.1.2 through CI, and dispatch public bootstrap validation before any default-install claim.
- [Phase 02 advisory] Review whether parse-time `simdjson::UNSUPPORTED_ARCHITECTURE` should map to `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` instead of `PURE_SIMDJSON_ERR_INTERNAL`.
- [Phase 02 advisory] Clean up stale public comments for now-live exports and decide whether `last_error_offset` should remain sentinel-only or surface real offsets.

## Session Continuity

Last session: 2026-04-25T05:54:57Z
Stopped at: Completed 09.1-01-PLAN.md
Resume file: .planning/phases/09.1-bootstrap-artifact-and-abi-alignment-for-default-installs/09.1-01-SUMMARY.md

**Planned Phase:** 09.1 (Bootstrap artifact and ABI alignment for default installs) — context ready, planning next — 2026-04-24T21:30:00Z

**Ready to Execute:** 09.1 (Bootstrap artifact and ABI alignment for default installs) — 2 verified plans in 2 waves — 2026-04-24T22:15:00Z
