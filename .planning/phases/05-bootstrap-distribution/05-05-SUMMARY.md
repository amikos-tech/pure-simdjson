---
phase: 05-bootstrap-distribution
plan: 05
subsystem: infra
tags: [cli, cobra, bootstrap, fetch, verify, all-platforms, offline-bundle, dist-08, m4, l4]

# Dependency graph
requires:
  - phase: 05-bootstrap-distribution
    plan: 01
    provides: SupportedPlatforms, ChecksumKey, Checksums map, Version const, error sentinels (ErrChecksumMismatch, ErrNoChecksum)
  - phase: 05-bootstrap-distribution
    plan: 02
    provides: BootstrapSync entry point, BootstrapOption surface (WithMirror/WithDest/WithVersion/WithTarget), CachePath
provides:
  - cmd/pure-simdjson-bootstrap CLI binary with four verbs (fetch, verify, platforms, version)
  - fetch --all-platforms offline pre-fetch capability for air-gapped deployments (DIST-08)
  - verify --all-platforms + --dest round-trip integrity gate for offline bundles (M4)
  - Per-platform progress lines on stderr during fetch --all-platforms so the CLI never looks silently hung (L4)
  - Integration tests exercising DIST-08 fetch flow and M4 verify round-trip via httptest
affects: [05-06-tests-ci-matrix, 06-ci-release-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Thin CLI wrapper pattern: cmd/ subcommands translate cobra flags into bootstrap.BootstrapOption setters and delegate all domain logic to internal/bootstrap"
    - "SilenceUsage + SilenceErrors on cobra root so errors go to stderr via main() instead of cobra's default usage dump (D-28)"
    - "Per-platform progress line before each BootstrapSync call so --all-platforms never looks silently hung (L4)"
    - "Integration-test seam: replace package-level bootstrap.Checksums via t.Cleanup-restored override map so httptest-served fake artifacts verify end-to-end"

key-files:
  created:
    - cmd/pure-simdjson-bootstrap/main.go
    - cmd/pure-simdjson-bootstrap/fetch.go
    - cmd/pure-simdjson-bootstrap/verify.go
    - cmd/pure-simdjson-bootstrap/platforms.go
    - cmd/pure-simdjson-bootstrap/version.go
    - cmd/pure-simdjson-bootstrap/fetch_test.go
    - cmd/pure-simdjson-bootstrap/verify_test.go
  modified:
    - .gitignore

key-decisions:
  - "CLI is a thin wrapper. No new download, checksum, or URL-construction logic lives in cmd/; all domain logic stays in internal/bootstrap. The CLI only translates cobra flags into BootstrapOption values and formats output."
  - "SilenceUsage and SilenceErrors are both true on the root command; errors are rendered exactly once by main() to stderr. Cobra's default would print usage on every error, drowning the real failure (D-28)."
  - "verify --all-platforms uses the first-error-wins semantics (records firstErr but keeps iterating) so users see every platform's status in one pass; exit code is non-zero when any platform fails."
  - "platformLibraryNameForCLI in verify.go duplicates the filename switch from internal/bootstrap/url.go instead of exporting platformLibraryName. The name set is locked by D-10, and exporting would expose an unstable public surface. The duplicated switch is five lines; drift risk is low."
  - "Integration tests mutate the package-level bootstrap.Checksums map via a t.Cleanup-restored override. The map is empty in dev (pre-CI-05) so httptest-served fake bytes cannot verify without this override. Tests are sequential by default — no locking needed."
  - "The root-level pure-simdjson-bootstrap binary produced by `go build ./cmd/...` is added to .gitignore so it is never accidentally committed when developers run the build from the repo root."

patterns-established:
  - "cmd/<cli>/ subcommand layout: one file per verb (fetch.go, verify.go, platforms.go, version.go), each exporting a newXxxCmd() *cobra.Command factory; main.go wires them under the root command"
  - "L4 progress style: 'fetching <goos>/<goarch>...\\n' before each platform's BootstrapSync; '  ok <goos>/<goarch>\\n' after success. Indentation disambiguates the confirmation line from the next platform's start line."
  - "M4 round-trip test: stage fake body + sha256 sum under <dest>/v<ver>/<os>-<arch>/<libname> for all 5 platforms, then runVerify(true, dest, &stdout, &stderr) and assert PASS count == 5"

requirements-completed: [DIST-08]

# Metrics
duration: 5min
completed: 2026-04-20
---

# Phase 5 Plan 5: CLI Bootstrap Summary

**cmd/pure-simdjson-bootstrap now ships with four verbs (fetch, verify, platforms, version) wrapping internal/bootstrap — fetch --all-platforms writes a round-trippable offline bundle with per-platform progress on stderr; verify --all-platforms --dest hashes every file in that bundle against the embedded Checksums map. DIST-08 is deliverable end-to-end via httptest integration coverage, and the CLI introduces zero new download/verify logic — it is a thin translation from cobra flags to BootstrapOption values.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-04-20T12:02:26Z
- **Completed:** 2026-04-20T12:07:25Z
- **Tasks:** 2
- **Commits:** 2
- **Files created:** 7
- **Files modified:** 1

## Accomplishments

- **main.go** wires the cobra root with `SilenceUsage: true` and `SilenceErrors: true` (D-28). Errors returned from `rootCmd.Execute()` are rendered exactly once to stderr and exit with code 1. Subcommands are added via a flat `AddCommand(...)` block — no nested groups.
- **fetch.go** implements the DIST-08 verb. `--all-platforms` iterates `bootstrap.SupportedPlatforms`, emits a per-platform `fetching <os>/<arch>...` line to stderr before each `BootstrapSync` call (L4), and a `  ok <os>/<arch>` confirmation after. `--target os/arch` (repeatable) supports selective pre-fetch, `--dest` routes artifacts into a caller-supplied directory, and `--version`/`--mirror` pass through to `WithVersion`/`WithMirror`.
- **verify.go** extends the legacy current-platform verify with M4's `--all-platforms` and `--dest` flags. `artifactPath(dest, goos, goarch)` returns either `bootstrap.CachePath(goos, goarch)` (no dest) or `<dest>/v<Version>/<os>-<arch>/<libname>` so offline bundles produced by `fetch --all-platforms --dest ./offline` can be round-trip verified with `verify --all-platforms --dest ./offline`. `verifyOne` hashes via `sha256` + `io.Copy`, prints `PASS <path>` to stdout on match, `FAIL <path>: expected ... got ...` to stderr on mismatch. First-error-wins semantics in the `--all-platforms` loop so every platform is reported in one pass.
- **platforms.go** iterates `bootstrap.SupportedPlatforms` and prints `<os>/<arch>  cached` or `<os>/<arch>  missing` based on `os.Stat(bootstrap.CachePath(...))` (D-26).
- **version.go** prints the library `Version`, `runtime.Version()`, and — when `debug.ReadBuildInfo()` succeeds — the module version of the CLI binary itself (D-27).
- **fetch_test.go** ships `TestFetchCmd` (all 5 platforms, httptest.Server serving R2-path-layout URLs, hit-count assertion = 5, filesystem assertion that every expected path exists) and `TestFetchCmdSingleTarget` (single `--target linux/amd64` produces exactly 1 download).
- **verify_test.go** ships `TestVerifyAllPlatformsDest` (M4 happy path, stage 5 fake artifacts, expect 5 PASS lines), `TestVerifyAllPlatformsDestMismatchFails` (corrupt one platform, assert `errors.Is(err, bootstrap.ErrChecksumMismatch)`), and `TestVerifyCurrentPlatformDefault` (no-flag path, cache dir redirected via `PURE_SIMDJSON_CACHE_DIR`).
- **.gitignore** now excludes the top-level `pure-simdjson-bootstrap` binary that `go build ./cmd/...` produces when invoked from the repo root.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement CLI — main.go + fetch.go + verify.go + platforms.go + version.go (with M4 verify extension and L4 fetch progress)** — `3ddfa00` (feat)
2. **Task 2: Write fetch_test.go + verify_test.go — integration tests for DIST-08 and M4** — `a5dd36e` (test)

## Files Created/Modified

**Created:**

- `cmd/pure-simdjson-bootstrap/main.go` — cobra root + AddCommand wiring.
- `cmd/pure-simdjson-bootstrap/fetch.go` — fetch verb with L4 per-platform progress.
- `cmd/pure-simdjson-bootstrap/verify.go` — verify verb with M4 `--dest` + `--all-platforms`.
- `cmd/pure-simdjson-bootstrap/platforms.go` — platforms verb with cache indicator.
- `cmd/pure-simdjson-bootstrap/version.go` — version verb via ReadBuildInfo.
- `cmd/pure-simdjson-bootstrap/fetch_test.go` — DIST-08 httptest integration test.
- `cmd/pure-simdjson-bootstrap/verify_test.go` — M4 round-trip + mismatch + default-path tests.

**Modified:**

- `.gitignore` — exclude root-level `pure-simdjson-bootstrap` binary (build artifact).

## Decisions Made

- **Thin-wrapper discipline:** The CLI owns no download, checksum, HTTP, flock, or URL logic. Every subcommand translates cobra flags into `bootstrap.BootstrapOption` values or calls an exported helper (`bootstrap.CachePath`, `bootstrap.Checksums`, `bootstrap.ChecksumKey`, `bootstrap.ErrChecksumMismatch`). This keeps the test surface of internal/bootstrap as the single source of truth for integrity and network correctness; cmd/ only tests translation.
- **`SilenceUsage` + `SilenceErrors`:** Cobra's default would print the full usage text on every RunE error, drowning the real failure message. We render errors exactly once via `fmt.Fprintln(os.Stderr, err)` in main() and exit 1 — D-28 exit-code contract preserved and user output stays legible.
- **First-error-wins in `verify --all-platforms`:** If darwin/amd64 is corrupted and linux/arm64 is missing, the user still sees `FAIL` for the first and `MISS` for the second in the same run. The returned error carries the first mismatch so `errors.Is(err, ErrChecksumMismatch)` remains truthful, while every platform's status is visible on the terminal.
- **`platformLibraryNameForCLI` duplicated in verify.go:** `platformLibraryName` is unexported in internal/bootstrap/url.go and the filename set is locked by D-10. Exporting it to cmd/ would broaden a public surface unnecessarily for a 5-line switch. Drift risk is low because the filename set has been stable since Plan 01 and any change there is a breaking release event that touches url.go anyway.
- **Integration-test checksum seam:** The package-level `bootstrap.Checksums` map is empty in development (pre-CI-05). Tests replace it with a fake-populated map via `t.Cleanup`-restored override so httptest-served bytes can hash-match embedded values. No extra test-only API needed — the map is already exported for CLI verify to read.
- **`.gitignore` update:** Running `go build ./cmd/pure-simdjson-bootstrap/...` from the repo root drops a 9 MiB `pure-simdjson-bootstrap` binary at the top level. Adding `/pure-simdjson-bootstrap` to .gitignore prevents a future developer from accidentally staging and committing it. This is a Rule 3 (blocking issue) mitigation — not in the original plan, but required to keep the repo clean for future contributors.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Issue] Added `/pure-simdjson-bootstrap` to .gitignore**

- **Found during:** Task 1 verification — `git status` showed an untracked `pure-simdjson-bootstrap` Mach-O binary at the repo root after running `go build ./cmd/pure-simdjson-bootstrap/...`.
- **Issue:** The plan's build verification (`go build ./cmd/pure-simdjson-bootstrap/...`) was copied verbatim from the plan's `<verify>` block. Running that command from the repo root drops the output binary at the top level. Without a .gitignore entry every future developer (and every CI run) would accumulate 9 MiB of untracked noise, and `git add -A` would silently commit the binary.
- **Fix:** Added `/pure-simdjson-bootstrap` to `.gitignore` alongside the existing Rust `/target/` entry. Root-anchored so it matches only the top-level binary, not `cmd/pure-simdjson-bootstrap/` the directory.
- **Files modified:** `.gitignore`.
- **Commit:** `3ddfa00` (bundled with Task 1 because the binary appears only after the CLI source exists; splitting the .gitignore into its own commit would require an intermediate commit with a broken state).

**2. [Rule 2 - Critical Functionality] Trimmed valid-target validation in `runFetch`**

- **Found during:** Task 1 implementation of `runFetch`.
- **Issue:** The plan's sketch splits `--target` on `/` and accepts any 2-part result. A user passing `--target /amd64` or `--target linux/` would get an empty goos or goarch fed into `bootstrap.WithTarget`, which would silently try to fetch from `https://.../v0.1.0/-amd64/libpure_simdjson.so` and produce a confusing HTTP 404 instead of an argument error.
- **Fix:** Added `parts[0] == "" || parts[1] == ""` guards to `runFetch`'s target parser so an empty os or arch is rejected up-front with a clear `invalid target ...: expected os/arch format` error.
- **Files modified:** `cmd/pure-simdjson-bootstrap/fetch.go`.
- **Commit:** `3ddfa00`.

## Authentication Gates

None — no external services touched. All integration tests use `httptest.NewServer`; the `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` env var keeps tests entirely on the R2-style URL path served by the fake mirror, so no GitHub credentials or live network are needed.

## Issues Encountered

None. The plan's code sketches compiled on first write; all 5 integration tests passed on first run. The only surprise was the stray root-level build artifact (handled by Rule 3 — .gitignore update).

## Known Stubs

None. `grep -ri 'TODO|FIXME|placeholder|coming soon|not available' cmd/pure-simdjson-bootstrap/` returns no results.

## User Setup Required

None. The CLI is usable immediately with `go run ./cmd/pure-simdjson-bootstrap <verb>` or `go install ./cmd/pure-simdjson-bootstrap`. The documented env vars (`PURE_SIMDJSON_BINARY_MIRROR`, `PURE_SIMDJSON_DISABLE_GH_FALLBACK`, `PURE_SIMDJSON_CACHE_DIR`, `PURE_SIMDJSON_LIB_PATH`) from Plans 02/04 are respected via the shared internal/bootstrap pipeline.

## Next Phase Readiness

- **Plan 06 (tests + CI matrix):** The CLI is the user-facing surface for the offline-bundle scenario in the Fault Injection Matrix. Plan 06 can drive `pure-simdjson-bootstrap fetch --all-platforms --dest X` + `verify --all-platforms --dest X` as a single end-to-end smoke test per CI runner.
- **Phase 6 (CI release pipeline):** Once `CI-05` populates `internal/bootstrap/checksums.go` with real SHA-256 digests, the same `verify --all-platforms` invocation doubles as a post-release integrity gate: `fetch --all-platforms --dest ./release-verify` then `verify --all-platforms --dest ./release-verify` either all PASS or the release is blocked.
- **Air-gapped consumers:** can now run `pure-simdjson-bootstrap fetch --all-platforms --dest /mnt/usb/vendor-libs` on a networked machine, transport the directory, and either set `PURE_SIMDJSON_LIB_PATH=/mnt/usb/vendor-libs/...` (bypass) or copy the directory into the OS cache (automatic load). Both paths are covered by the resolver chain from Plan 04.

## Self-Check: PASSED

All created files exist and all commits are present on the branch:

- FOUND: `cmd/pure-simdjson-bootstrap/main.go`
- FOUND: `cmd/pure-simdjson-bootstrap/fetch.go`
- FOUND: `cmd/pure-simdjson-bootstrap/verify.go`
- FOUND: `cmd/pure-simdjson-bootstrap/platforms.go`
- FOUND: `cmd/pure-simdjson-bootstrap/version.go`
- FOUND: `cmd/pure-simdjson-bootstrap/fetch_test.go`
- FOUND: `cmd/pure-simdjson-bootstrap/verify_test.go`
- FOUND: `.gitignore` (updated)
- FOUND: commit `3ddfa00` (Task 1)
- FOUND: commit `a5dd36e` (Task 2)

Plan-level verification:

- `go build ./cmd/pure-simdjson-bootstrap/...` exit 0
- `go build ./...` exit 0
- `go vet ./cmd/pure-simdjson-bootstrap/...` exit 0
- `go test ./cmd/pure-simdjson-bootstrap/... -run "TestFetchCmd|TestVerifyAllPlatforms|TestVerifyCurrentPlatformDefault" -count=1 -timeout 60s -v` — PASS (5/5 tests, 0.73s)
- `go test ./... -count=1 -timeout 180s` — PASS (root purejson 6.0s, cmd/pure-simdjson-bootstrap 1.6s, internal/bootstrap 9.6s, internal/ffi 1.1s)
- Every grep acceptance criterion from the plan matches:
  - `grep "SilenceUsage.*true" cmd/pure-simdjson-bootstrap/main.go` — present (line 16)
  - `grep "SilenceErrors.*true" cmd/pure-simdjson-bootstrap/main.go` — present (line 17)
  - `grep "all-platforms" cmd/pure-simdjson-bootstrap/fetch.go` — present (D-24)
  - `grep "all-platforms" cmd/pure-simdjson-bootstrap/verify.go` — present (M4)
  - `grep -- "--dest" cmd/pure-simdjson-bootstrap/verify.go` — present (M4)
  - `grep 'fetching %s/%s' cmd/pure-simdjson-bootstrap/fetch.go` — present (L4)
  - `grep "ReadBuildInfo" cmd/pure-simdjson-bootstrap/version.go` — present (D-27)
  - `grep "SupportedPlatforms" cmd/pure-simdjson-bootstrap/platforms.go` — present
  - `grep -r "sigstore\|cosign" cmd/ --include="*.go"` — 0 matches (D-30)
  - `grep -r "sigstore" . --include="*.go"` — 0 matches (DIST-10 repo-wide)
  - `grep "TestFetchCmd" cmd/pure-simdjson-bootstrap/fetch_test.go` — present
  - `grep "TestVerifyAllPlatforms" cmd/pure-simdjson-bootstrap/verify_test.go` — present

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
