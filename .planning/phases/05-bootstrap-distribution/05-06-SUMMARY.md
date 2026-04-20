---
phase: 05-bootstrap-distribution
plan: 06
subsystem: testing
tags: [bootstrap, httptest, fault-injection, redirect-downgrade, concurrency, flock, ctx-cancel, dist-09, dist-10, doc-05, l5]

# Dependency graph
requires:
  - phase: 05-bootstrap-distribution
    plan: 03
    provides: Plan 03 fault matrix coverage (items 1, 2, 9, 11) plus URL/Checksum/CacheDir unit tests, M3 export seams, isLadderFatalError ladder-vs-URL classification
  - phase: 05-bootstrap-distribution
    plan: 04
    provides: 4-stage resolveLibraryPath, double-checked locking, DIST-09 absolute-path invariant
  - phase: 05-bootstrap-distribution
    plan: 05
    provides: pure-simdjson-bootstrap CLI with fetch/verify/platforms/version verbs
provides:
  - Complete Fault Injection Matrix coverage (items 3, 4, 5, 6, 7, 8, 10, 11) — every row in 05-VALIDATION.md is now backed by a passing automated test
  - TestFallback503R2Then200GH — 503 retry-cascade + GH fallback with H1 platform-tagged path assertion baked in (regression guard for Plan 02 wiring)
  - TestDisableGHFallbackWith404 — DISABLE_GH_FALLBACK env-var integration coverage
  - TestBootstrapSyncCancellation + TestBootstrapSyncCtxCancelDuringSleep — context cancellation propagates within milliseconds, no orphan *.tmp files left behind
  - TestRedirectDowngradeUnit + TestRedirectDowngradeWired — T-05-04 covered both at the policy function and at the wiring site of newHTTPClient
  - TestConcurrentBootstrap — 8-goroutine flock contention with L1 cross-process scope rationale documented inline
  - TestGitHub403RateLimit — body-sniff classification proven through the retry ladder end-to-end
  - TestMirrorOverride — env-var driven mirror override exercised through a real httptest download
  - Extended TestResolveLibraryPathAbsolute (DIST-09) with three sub-tests covering the success path, env-override-missing-absolute, and env-override-missing-relative branches
  - Production-bug fix in downloadOnce: named-return-zeroing was leaking *.tmp files on cancellation; cleanup defer now closes over a captured local
  - docs/bootstrap.md — DOC-05 user-facing distribution documentation with all 4 env vars, air-gapped/corporate/mirror flows, CLI reference, GH asset naming table (H1), cosign recipe (D-29 docs-only), and L5 Phase-6 honesty note
affects: [06-ci-release-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Captured-local pattern for defer cleanup in functions with named returns: rebind tmp path to a local before the cleanup closure so explicit `return \"\", \"\", err` cannot zero the path the defer needs"
    - "Two-tier T-05-04 coverage: unit-test the redirect policy function in isolation, then wire-test the policy is installed on newHTTPClient.CheckRedirect — pairs with the existing httptest end-to-end downgrade test"
    - "L1 multi-process lock scope: intra-process flock concurrency tested via 8 goroutines; cross-process behaviour delegated to OS flock/LockFileEx semantics, rationale captured inline above the test"

key-files:
  created:
    - docs/bootstrap.md
    - .planning/phases/05-bootstrap-distribution/05-06-SUMMARY.md
  modified:
    - internal/bootstrap/bootstrap_test.go
    - internal/bootstrap/download.go
    - internal/bootstrap/export_test.go
    - library_loading_test.go

key-decisions:
  - "TestRedirectDowngrade is split into TestRedirectDowngradeUnit (calls rejectHTTPSDowngrade directly with a synthetic via-chain) and TestRedirectDowngradeWired (asserts newHTTPClient().CheckRedirect points at a function that rejects the same downgrade) — preferred over option (b) two-server httptest topology because plumbing httptest.NewTLSServer to redirect into an http:// scheme is brittle and the existing TestHTTPSDowngradeRejected (Plan 02) already covers the end-to-end behaviour"
  - "The L1 cross-process flock test was deliberately not added in v0.1; the rationale (OS-level flock/LockFileEx correctness, pure-onnx precedent, brittle Windows path quoting in subprocess CI tests) is captured in a comment block above TestConcurrentBootstrap so a future contributor finds the reasoning before re-discovering it"
  - "TestNewParserCacheHit was NOT added: the closest substitute is TestResolveLibraryPathCacheHit (Plan 04, already in library_loading_test.go) which proves the loader returns the cached path without invoking BootstrapSync. Adding a second test that wraps activeLibrary() to count network requests would either need a real shared library on disk (host-dependent) or test infrastructure (shared library mock) that does not exist in v0.1; deferred"
  - "downloadOnce captures the temp path in a local createdTmp before the cleanup defer so an early `return \"\", \"\", err` cannot zero the named-return tmpPath the defer reads. Without this fix TestBootstrapSyncCancellation observed orphan *.tmp files in the cache dir after ctx cancel; this is a v0.1-shipping bug the integration test exposed"
  - "TestResolveLibraryPathAbsolute was extended into three sub-tests (success cache hit, missing absolute env path, missing relative env path) so DIST-09's absolute-path invariant is enforced on both happy and error paths — the original Plan 04 test only covered the happy path"

patterns-established:
  - "Named-return + defer cleanup anti-pattern: when a function has a named return and uses defer to clean up an intermediate resource referenced by that return, an explicit `return \"\", \"\", err` zeroes the named-return inside the defer. Always capture the resource path in a local before the cleanup closure"
  - "Documenting scope decisions inline at the test site: when a reviewer-suggested test is intentionally NOT added, leave a comment at the closest analogous test explaining the rationale and the typical Go pattern future contributors would use, so the decision is not silently re-discovered"

requirements-completed: [DIST-02, DIST-04, DIST-05, DIST-07, DIST-09, DIST-10, DOC-05]

# Metrics
duration: 9min
completed: 2026-04-20
---

# Phase 5 Plan 6: Test Matrix Closeout + Bootstrap Documentation Summary

**Final plan in Phase 5: every Fault Injection Matrix row in 05-VALIDATION.md
is now backed by a passing automated test, including the matrix items that
required library_loading.go (Plan 04) and the CLI (Plan 05) to land first.
Two T-05-04 redirect-downgrade tests cover the policy function and its wiring
site individually, complementing the existing end-to-end test from Plan 02.
A production-side bug surfaced by TestBootstrapSyncCancellation is fixed in the
same commit (Rule 1 — named-return-zeroing in downloadOnce was orphaning *.tmp
files). DOC-05 ships docs/bootstrap.md with all 4 env vars, the CLI reference,
the H1 platform-tagged asset table, the cosign verify-blob recipe, and the L5
Phase-6 honesty note. DIST-10 grep gate (no sigstore Go imports) is verified
clean repo-wide.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-04-20T12:17:01Z
- **Completed:** 2026-04-20T12:25:41Z
- **Tasks:** 2
- **Commits:** 2
- **Files created:** 1 (docs/bootstrap.md) + this SUMMARY.md
- **Files modified:** 4 (bootstrap_test.go, download.go, export_test.go, library_loading_test.go)

## Accomplishments

- **Closed out the Fault Injection Matrix.** All 11 rows from 05-VALIDATION.md
  now map to passing automated tests. Plans 02/03 covered items 1, 2, 5, 7,
  9, 11 (unit subset); this plan adds items 3, 4, 6, 8 (intra-process), 10,
  and the integration coverage of item 11 — leaving item 8 (cross-process
  flock contention) explicitly delegated to OS semantics per L1.
- **Production-bug fix in downloadOnce.** TestBootstrapSyncCancellation
  surfaced a real defer-vs-named-return interaction: the cleanup defer
  captured the named return `tmpPath`, but explicit `return "", "", err` calls
  earlier in the function zero it before the defer fires, so `os.Remove("")`
  ran instead of `os.Remove(actualTempFile)`. The fix captures the temp path
  in a local before the defer closure. This is a Rule 1 fix — without it the
  cache dir would accumulate `*.tmp` debris on every cancelled bootstrap.
- **T-05-04 two-tier coverage.** Added TestRedirectDowngradeUnit (calls
  rejectHTTPSDowngrade with a synthetic via-chain — proves the policy
  function rejects HTTPS→HTTP) and TestRedirectDowngradeWired (asserts
  newHTTPClient().CheckRedirect points at that policy function). Together
  with the existing TestHTTPSDowngradeRejected (Plan 02), this gives behaviour
  + wiring + end-to-end coverage of the redirect-downgrade defence without
  needing a brittle two-server httptest topology.
- **DIST-09 hardening.** TestResolveLibraryPathAbsolute now has three
  sub-tests instead of one. The two new sub-tests
  (`env-override-missing-absolute-input` and
  `env-override-missing-relative-input`) cover the error branches where a
  regression could leak a relative path into Windows LoadLibrary, which is
  exactly the DLL-hijack vector pitfall #29 documents.
- **L1 cross-process scope decision recorded in source.** A comment block
  above TestConcurrentBootstrap explains why intra-process concurrency is
  covered but inter-process flock semantics is delegated to the OS — pure-onnx
  precedent + Windows path-quoting brittleness in subprocess tests + the fact
  that flock/LockFileEx is OS code, not application code. Future contributors
  see the reasoning at the test site.
- **DOC-05: docs/bootstrap.md** covers the full v0.1 distribution surface:
  five-stage resolution flow, all four env vars (including L2's
  `PURE_SIMDJSON_CACHE_DIR`), air-gapped + corporate-firewall + custom-mirror
  workflows, the CLI reference (`fetch`/`verify`/`platforms`/`version` with
  `--all-platforms` and `--dest`), the GitHub asset-naming table that
  documents H1 (platform-tagged flat namespace) vs the R2 directory layout,
  the cosign verify-blob recipe (D-29, docs-only), retry/error semantics, the
  supported-platform list, and an explicit L5 honesty note distinguishing
  what v0.1 verifies via httptest from what Phase 6 CI-05 will verify
  end-to-end against the live CDN.

## Fault Injection Matrix → Test Mapping (final)

| #  | Fault                                                       | Covered By                                              | Plan      |
| -- | ----------------------------------------------------------- | ------------------------------------------------------- | --------- |
| 1  | Checksum corruption → ErrChecksumMismatch                   | `TestChecksumMismatchIsPermanent`                       | 05-02     |
| 2  | HTTP 429 then 200 → retry succeeds                          | `TestRetryOn429ThenSuccess`                             | 05-03     |
| 3  | HTTP 503 all R2 attempts, 200 GH → fallback succeeds        | `TestFallback503R2Then200GH` (H1 path-asserting)        | **05-06** |
| 4  | ctx cancel mid-download → context.Canceled, temp cleaned up | `TestBootstrapSyncCancellation`                         | **05-06** |
| 5  | ctx cancel during retry sleep → returns within 50ms         | `TestBootstrapSyncCtxCancelDuringSleep` + `TestSleepWithJitterCtxCancel` | **05-06** + 05-02 |
| 6  | Concurrent bootstrap (8 goroutines) → exactly one download  | `TestConcurrentBootstrap`                               | **05-06** |
| 7  | HTTPS→HTTP redirect rejected                                | `TestRedirectDowngradeUnit` + `TestRedirectDowngradeWired` + `TestHTTPSDowngradeRejected` | **05-06** + 05-02 |
| 8  | Lock file contention                                        | `TestConcurrentBootstrap` (intra-process); cross-process delegated to OS per L1 | **05-06** |
| 9  | 404 on R2, 200 on GH → GH fallback fires                    | `TestFallback404R2Then200GH`                            | 05-03     |
| 10 | DISABLE_GH_FALLBACK=1 + R2 404 → ErrAllSourcesFailed        | `TestDisableGHFallbackWith404`                          | **05-06** |
| 11 | GitHub 403 rate-limit body classified retryable             | `TestGitHub403RateLimit` (integration) + `TestIsRetryable` (unit) | **05-06** + 05-02 |

Bonus integration test (not a matrix row but a DIST-07 deliverable):
`TestMirrorOverride` proves the `PURE_SIMDJSON_BINARY_MIRROR` env var routes
downloads to the configured mirror without an explicit `WithMirror` option.

## Task Commits

1. **Task 1: Round out Fault Injection Matrix integration tests + Rule 1 fix
   for orphan-tmp on cancel** — `379fc1c` (test)
2. **Task 2: docs/bootstrap.md (DOC-05)** — `0d0c247` (docs)

## Files Created/Modified

**Created:**

- `docs/bootstrap.md` — DOC-05 distribution documentation (229 lines).
- `.planning/phases/05-bootstrap-distribution/05-06-SUMMARY.md` (this file).

**Modified:**

- `internal/bootstrap/bootstrap_test.go` — appended 9 integration test
  functions (`TestFallback503R2Then200GH`, `TestDisableGHFallbackWith404`,
  `TestBootstrapSyncCancellation`, `TestBootstrapSyncCtxCancelDuringSleep`,
  `TestRedirectDowngradeUnit`, `TestRedirectDowngradeWired`,
  `TestConcurrentBootstrap`, `TestGitHub403RateLimit`, `TestMirrorOverride`)
  + `bootstrapRetryAttempts` mirror constant + L1 rationale comment block.
  Imports gained `strings`, `sync`. Total `func Test*` count: **34** (Plan 03
  baseline 25 + 9 added).
- `internal/bootstrap/download.go` — Rule 1 fix in `downloadOnce`: rebind
  `f.Name()` to a captured local `createdTmp` before the cleanup defer so
  early `return "", "", err` paths cannot zero the path the defer needs.
- `internal/bootstrap/export_test.go` — added `RejectHTTPSDowngradeForTest`
  and `NewHTTPClientForTest` seams so the new T-05-04 unit + wiring tests
  can exercise the policy function directly without spinning up an HTTP
  server topology.
- `library_loading_test.go` — extended `TestResolveLibraryPathAbsolute`
  into three sub-tests (success cache hit + env-override-missing-absolute +
  env-override-missing-relative) sharing a new `assertAllAbsoluteOrEmpty`
  helper.

## Decisions Made

- **Two-tier T-05-04 coverage instead of two-server httptest:** Plan
  recommended option (a). Wiring an httptest.NewTLSServer to redirect into
  an http:// scheme would require an extra non-TLS server and careful URL
  surgery to thread through `Location:` headers. The unit test of
  `rejectHTTPSDowngrade` plus the wiring test of `newHTTPClient().CheckRedirect`
  is functionally equivalent and brittle-free, and pairs cleanly with the
  pre-existing `TestHTTPSDowngradeRejected` (Plan 02) which already exercises
  the end-to-end behaviour. Net coverage is better than the alternative.
- **Cross-process flock test deferred per L1:** the comment block at
  TestConcurrentBootstrap explains the rationale (OS semantics + Windows path
  quoting brittleness + pure-onnx precedent). A subprocess-spawning test
  using `exec.Command(os.Args[0])` with a `TEST_SUBPROCESS=1` env var is the
  typical Go pattern; future contributors find that hint at the test site
  rather than re-discovering it.
- **TestNewParserCacheHit not added:** the matrix doesn't require it
  (DIST-05 row "Second NewParser() call (cache hit) makes no HTTP requests"
  is satisfied by `TestResolveLibraryPathCacheHit` from Plan 04). The plan's
  DIST-05 row is met without adding a second test that would either need a
  loadable shared library on disk (host-dependent) or new mock infrastructure.
  Documented in the test list and in this SUMMARY for traceability.
- **`bootstrapRetryAttempts` mirror constant in the test file:** the test
  needs to assert R2 hits == 3 (the production value of `bootstrapRetryAttempt`
  inside the bootstrap package). Rather than re-export the unexported
  constant, the test file declares its own `bootstrapRetryAttempts = 3` next
  to a comment that flags the duplication. If `download.go` ever changes the
  value, the test will fail loudly and the comment points at the fix.
- **DIST-10 grep gate verified at the close of the plan, not just at planning
  time:** `grep -r 'sigstore' . --include='*.go'` returns exit 1 (zero
  matches). cosign verification is documented in `docs/bootstrap.md` as a
  user-side optional step using the cosign CLI; no Go code in this repo
  imports any sigstore package.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Orphan `*.tmp` files left behind on context cancellation
in `downloadOnce`**

- **Found during:** Task 1 verification — `TestBootstrapSyncCancellation`
  failed on the first run with `orphan temp file left behind after
  cancellation: .../v0.1.0/linux-amd64/pure-simdjson-3326042956.tmp`.
- **Issue:** `downloadOnce` declares named returns
  `(tmpPath, digest string, err error)`, assigns `tmpPath = f.Name()` after
  `os.CreateTemp`, then installs a cleanup defer that runs
  `os.Remove(tmpPath)` when `success == false`. But every error path uses
  explicit `return "", "", err`, which sets the named return `tmpPath` to
  `""` _before_ the deferred function executes. The defer therefore calls
  `os.Remove("")` (a no-op), and the temp file leaks into the cache dir on
  every cancelled or failed download. This is a v0.1-shipping bug — without
  it the air-gapped flow's cache dir would silently accumulate `.tmp`
  garbage in any environment with intermittent network.
- **Fix:** Capture `f.Name()` into a local `createdTmp := f.Name()` before
  the cleanup defer; the closure now references that local instead of the
  named return. The successful return path uses `return createdTmp, digest, nil`
  for consistency.
- **Files modified:** `internal/bootstrap/download.go`.
- **Verification:** `TestBootstrapSyncCancellation` walks the cache dir
  after a cancelled bootstrap and asserts no `*.tmp` files remain — passes
  after the fix. All 34 bootstrap tests still pass under `-race`.
- **Committed in:** `379fc1c` (bundled with the integration tests because
  the test that surfaces the bug is the test that verifies the fix; splitting
  them would leave a red intermediate commit).

**2. [Rule 3 — Blocking Issue] Added `RejectHTTPSDowngradeForTest` and
`NewHTTPClientForTest` seams to `export_test.go`**

- **Found during:** Drafting `TestRedirectDowngradeUnit` and
  `TestRedirectDowngradeWired`.
- **Issue:** Both functions need to exercise the unexported
  `rejectHTTPSDowngrade` policy function and inspect the
  `*http.Client.CheckRedirect` field that `newHTTPClient` installs. Neither
  was previously exposed via `export_test.go` (Plan 02 only exposed
  resolveConfig + WithHTTPClient + WithGitHubBaseURL et al.). Without these
  seams the new tests cannot live in the external `bootstrap_test` package.
- **Fix:** Added two re-exports:
  - `RejectHTTPSDowngradeForTest(req, via)` — direct call into
    `rejectHTTPSDowngrade`.
  - `NewHTTPClientForTest()` — direct call into `newHTTPClient`.
  Both compile only during `go test` (the `_test.go` suffix gates them) and
  are the idiomatic Go test-seam pattern (cf. `net/http/export_test.go` in
  the standard library).
- **Files modified:** `internal/bootstrap/export_test.go`.
- **Committed in:** `379fc1c`.

---

**Total deviations:** 2 (1 Rule 1 bug fix, 1 Rule 3 blocking)
**Impact on plan:** Both deviations were anticipated by the plan's
scaffolding — the Rule 3 seams were implicit prerequisites for the
plan's preferred option (a) for T-05-04, and the Rule 1 fix is a real
correctness bug the plan's matrix coverage was designed to expose.

## Authentication Gates

None — every test uses `httptest.NewServer` / `httptest.NewTLSServer` or
synthetic in-memory request objects. No external services were touched.

## Issues Encountered

The orphan `*.tmp` cancellation bug was the only surprise; documented above as
deviation #1. Everything else compiled and passed on first run.

## Known Stubs

None. `grep -ri 'TODO|FIXME|placeholder|coming soon|not available'` over the
files modified by this plan returns no in-source matches (the docs file does
contain prose like "is not supported" describing the linux/arm exclusion,
which is intentional documentation rather than a stub marker).

## User Setup Required

None — the env vars documented in `docs/bootstrap.md` are user-facing
configuration knobs; no setup is required for the library to function with
defaults.

## Next Phase Readiness

- **Phase 6 — CI release pipeline + cosign signing job:** every grep gate
  Plan 06 will assert against the released artifacts is now provably tested
  here. The cosign verify-blob recipe in `docs/bootstrap.md` is the user-side
  contract Phase 6's release.yml workflow must satisfy. The L5 honesty note
  spells out exactly what flips from "tested via httptest" to "tested
  end-to-end against the live CDN" once CI-05 populates `checksums.go`.
- **Air-gapped consumers:** the documented flow (`fetch --all-platforms
  --dest <dir>` → transport → `PURE_SIMDJSON_LIB_PATH=...`) is verifiable
  with the existing CLI today; the only piece missing for production use is
  Phase 6's release artifacts. No code change needed in `pure-simdjson` to
  unblock air-gapped use.

## Self-Check: PASSED

Created/modified files all exist and the task commits are present on the
branch:

- FOUND: `docs/bootstrap.md`
- FOUND: `internal/bootstrap/bootstrap_test.go` (34 `func Test` definitions)
- FOUND: `internal/bootstrap/download.go` (with `createdTmp` local capture)
- FOUND: `internal/bootstrap/export_test.go` (with `RejectHTTPSDowngradeForTest`
  and `NewHTTPClientForTest`)
- FOUND: `library_loading_test.go` (with `assertAllAbsoluteOrEmpty` helper +
  three-sub-test `TestResolveLibraryPathAbsolute`)
- FOUND: commit `379fc1c` (Task 1)
- FOUND: commit `0d0c247` (Task 2)

Plan-level verification:

- `go build ./...` exit 0
- `go vet ./...` exit 0
- `go test ./internal/bootstrap/... -count=1 -timeout 60s` — PASS (~17s, 34 test functions)
- `go test ./internal/bootstrap/... -count=1 -race -timeout 120s` — PASS (~18s)
- `go test ./... -count=1 -race -timeout 180s` — PASS
  (root purejson 10.2s, cmd/pure-simdjson-bootstrap 3.3s, internal/bootstrap 18.1s, internal/ffi 2.1s)
- Targeted plan-06 tests all pass: `TestFallback503R2Then200GH`,
  `TestDisableGHFallbackWith404`, `TestBootstrapSyncCancellation`,
  `TestBootstrapSyncCtxCancelDuringSleep`, `TestRedirectDowngradeUnit`,
  `TestRedirectDowngradeWired`, `TestConcurrentBootstrap`,
  `TestGitHub403RateLimit`, `TestMirrorOverride`, `TestResolveLibraryPathAbsolute`
  (3 sub-tests).
- Every grep acceptance criterion from the plan matches:
  - `head -1 internal/bootstrap/bootstrap_test.go` → `package bootstrap_test`
  - `grep -c "^func Test" internal/bootstrap/bootstrap_test.go` → 34 (>= 25)
  - `grep 'libpure_simdjson-linux-amd64.so' internal/bootstrap/bootstrap_test.go` → 4 hits (H1 fixture in TestFallback503R2Then200GH + URL/asset-name tests)
  - `grep 'bootstrap\.ResetBootstrapFailureCacheForTest' internal/bootstrap/bootstrap_test.go` → 2 hits (clearBootstrapEnv helper)
  - `grep "05-REVIEWS.md L1" internal/bootstrap/bootstrap_test.go` → 1 hit (rationale comment)
  - `test -f docs/bootstrap.md` → exists
  - All 9 doc gates from Task 2 pass: `PURE_SIMDJSON_LIB_PATH`,
    `PURE_SIMDJSON_BINARY_MIRROR`, `PURE_SIMDJSON_DISABLE_GH_FALLBACK`,
    `PURE_SIMDJSON_CACHE_DIR`, `cosign verify-blob`, `Air-Gapped`, `Corporate`,
    `pure-simdjson-bootstrap`, `Phase 6` / `CI-05`, the H1 GH asset table,
    and `verify --all-platforms --dest`.
  - `grep -r 'sigstore' . --include='*.go'` → 0 hits (DIST-10).

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
