---
phase: 05-bootstrap-distribution
plan: 01
subsystem: infra
tags: [bootstrap, distribution, go, cobra, flock, error-sentinels, purego]

# Dependency graph
requires:
  - phase: 03-go-public-api-purego-happy-path
    provides: root purejson error sentinels to extend (errors.go var block)
  - phase: 04-full-typed-accessor-surface
    provides: stable v0.1 public API surface; platform library filenames consumed by platformLibraryName()
provides:
  - internal/bootstrap package skeleton (compile-time Version + Checksums map placeholder)
  - URL construction utilities (r2ArtifactURL, githubArtifactURL) with platform-tagged GH asset names (H1 fix)
  - ChecksumKey exported helper for the CLI verify subcommand (Plan 05)
  - validateBaseURL HTTPS-only gate (loopback exception for tests) — T-05-05 mitigation
  - Canonical bootstrap error sentinels (ErrChecksumMismatch, ErrAllSourcesFailed, ErrNoChecksum) with pointer-identity aliasing from root purejson
  - Platform flock primitives (unix.Flock + windows.LockFileEx) in build-tag-paired files
  - cobra v1.10.2 available for Plan 05 CLI tooling
affects: [05-02-download-http-pipeline, 05-03-bootstrap-sync-cache, 05-04-loader-integration, 05-05-cli-bootstrap, 05-06-tests-ci-matrix, 06-ci-release-pipeline]

# Tech tracking
tech-stack:
  added: [github.com/spf13/cobra v1.10.2, github.com/spf13/pflag v1.0.9, github.com/inconshreveable/mousetrap v1.1.0]
  patterns:
    - "Canonical error sentinels in internal/bootstrap aliased by root purejson via pointer identity (errors.Is matches both paths)"
    - "GitHub release asset names are platform-tagged (libpure_simdjson-<goos>-<goarch>.ext) to avoid flat-namespace collision; cache filename stays platform-independent"
    - "Build-tag paired platform files (_unix.go / _windows.go) with identical exported symbol signatures (lockFile, unlockFile, isLockWouldBlock)"
    - "Exported ChecksumKey helper crosses the cmd/ → internal/bootstrap boundary without leaking map layout"

key-files:
  created:
    - internal/bootstrap/version.go
    - internal/bootstrap/checksums.go
    - internal/bootstrap/url.go
    - internal/bootstrap/errors.go
    - internal/bootstrap/bootstrap_lock_unix.go
    - internal/bootstrap/bootstrap_lock_windows.go
  modified:
    - errors.go
    - go.mod
    - go.sum

key-decisions:
  - "Canonical error sentinels live in internal/bootstrap/errors.go only; root errors.go aliases via pointer identity (H2 resolution)"
  - "GitHub release asset names are platform-tagged via githubAssetName(); R2 URLs keep platform-independent filename under <os>-<arch>/ directory (H1 resolution)"
  - "ChecksumKey exported to support the Plan 05 CLI verify subcommand without exposing the Checksums map layout"
  - "validateBaseURL rejects http:// for non-loopback hosts to mitigate T-05-05 information disclosure"

patterns-established:
  - "Platform-tagged GH asset names: libpure_simdjson-<goos>-<goarch>.(so|dylib), pure_simdjson-<goos>-<goarch>-msvc.dll"
  - "Error sentinel single source of truth: canonical in internal/bootstrap, re-exported via `var ErrX = bootstrap.ErrX` in root"
  - "Build-tag pairing: //go:build !windows vs //go:build windows for os-specific flock wrappers"

requirements-completed: [DIST-01, DIST-02, DIST-03, DIST-06, DIST-09, DIST-10]

# Metrics
duration: 3min
completed: 2026-04-20
---

# Phase 5 Plan 1: Bootstrap Package Scaffold Summary

**internal/bootstrap package now exposes compile-time Version, checksums map placeholder, URL construction with platform-tagged GH asset names, canonical error sentinels aliased from root purejson, and platform flock primitives — all three HIGH-severity contracts (H1 GH asset naming, H2 error ownership, D-10 platform filenames) locked before Wave 2.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-04-20T11:23:57Z
- **Completed:** 2026-04-20T11:26:10Z
- **Tasks:** 2
- **Files created:** 6
- **Files modified:** 3

## Accomplishments

- `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go` carry the compile-time Version constant and the SHA-256 map placeholder CI-05 will populate at release time
- `internal/bootstrap/url.go` contains the URL layer Wave 2 needs: `SupportedPlatforms`, `platformLibraryName`, `githubAssetName` (H1 fix), `r2ArtifactURL`, `githubArtifactURL`, `ChecksumKey`, and `validateBaseURL`
- `internal/bootstrap/errors.go` defines the three canonical error sentinels; root `errors.go` re-exports them via pointer alias (`errors.Is` matches both `purejson.ErrChecksumMismatch` and `bootstrap.ErrChecksumMismatch`)
- `internal/bootstrap/bootstrap_lock_unix.go` (unix.Flock LOCK_EX|LOCK_NB) and `internal/bootstrap/bootstrap_lock_windows.go` (windows.LockFileEx with LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY) provide the cross-platform flock primitives for Plan 03
- `github.com/spf13/cobra v1.10.2` added to `go.mod` for the Plan 05 CLI

## Task Commits

Each task was committed atomically:

1. **Task 1: Add cobra dep + create internal/bootstrap package skeleton (version.go, checksums.go, url.go)** — `080d382` (feat)
2. **Task 2: Create internal/bootstrap/errors.go (canonical sentinels) + root errors.go aliases + platform flock files** — `d169c6a` (feat)

## Files Created/Modified

**Created:**
- `internal/bootstrap/version.go` — compile-time `const Version = "0.1.0"`
- `internal/bootstrap/checksums.go` — empty `map[string]string` placeholder keyed by `v<version>/<os>-<arch>/<libname>`
- `internal/bootstrap/url.go` — URL construction + platform constants + exported `ChecksumKey` + `validateBaseURL`
- `internal/bootstrap/errors.go` — canonical `ErrChecksumMismatch`, `ErrAllSourcesFailed`, `ErrNoChecksum`
- `internal/bootstrap/bootstrap_lock_unix.go` — `unix.Flock` wrappers (`lockFile`, `unlockFile`, `isLockWouldBlock`)
- `internal/bootstrap/bootstrap_lock_windows.go` — `windows.LockFileEx`/`UnlockFileEx` wrappers

**Modified:**
- `errors.go` — added `internal/bootstrap` import and a new `var()` block aliasing the three sentinels (existing sentinel block untouched)
- `go.mod` — added `github.com/spf13/cobra v1.10.2`
- `go.sum` — pinned cobra + pflag + mousetrap checksums

## Decisions Made

- **Error sentinel ownership (H2):** canonical `errors.New` calls live only in `internal/bootstrap/errors.go`; root `errors.go` re-exports via pointer alias. `grep -r 'errors.New("checksum mismatch")' --include="*.go"` across the repo returns exactly 1, enforcing the no-duplication invariant.
- **GitHub asset naming (H1):** GH releases are a flat namespace, so the release asset filename is platform-tagged (`libpure_simdjson-<goos>-<goarch>.(so|dylib)`, `pure_simdjson-<goos>-<goarch>-msvc.dll`) and distinct from the cache filename returned by `platformLibraryName`. The R2 URL keeps a platform-independent file segment under the `<os>-<arch>/` directory because directories prevent collision.
- **`ChecksumKey` exported:** the Plan 05 CLI (separate package under `cmd/`) needs the same key format as the `Checksums` map; exporting the helper keeps the map layout encapsulated.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None. `go get github.com/spf13/cobra@v1.10.2` records the dependency as `// indirect` until a Go source file actually imports cobra (expected once Plan 05's CLI lands). This is benign for Plan 01 — the plan's acceptance criterion only requires the entry to exist in `go.mod`, which `grep "github.com/spf13/cobra" go.mod` confirms.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Wave 2 (Plan 02: HTTP download pipeline) can compile against `r2ArtifactURL`, `githubArtifactURL`, `githubAssetName`, `platformLibraryName`, `ChecksumKey`, `ErrAllSourcesFailed`, `validateBaseURL`
- Plan 03 (bootstrap sync + cache) can compile against the `lockFile`/`unlockFile`/`isLockWouldBlock` flock primitives and `ErrNoChecksum`
- Plan 05 (CLI) can import cobra (already in go.mod) and call `ChecksumKey`
- Phase 6 CI-05 is reminded that GH release upload MUST use `githubAssetName(goos, goarch)` verbatim; windows builds `pure_simdjson-msvc.dll` locally and renames at upload

## Self-Check: PASSED

All created files exist and all commits are present on the branch:

- FOUND: `internal/bootstrap/version.go`
- FOUND: `internal/bootstrap/checksums.go`
- FOUND: `internal/bootstrap/url.go`
- FOUND: `internal/bootstrap/errors.go`
- FOUND: `internal/bootstrap/bootstrap_lock_unix.go`
- FOUND: `internal/bootstrap/bootstrap_lock_windows.go`
- FOUND: commit `080d382` (Task 1)
- FOUND: commit `d169c6a` (Task 2)

Plan-level verification:
- `go build ./...` exit 0
- `go vet ./internal/bootstrap/...` exit 0
- `grep -c 'errors.New("checksum mismatch")' .` across repo = 1 (canonical only)
- `grep -r "sigstore" . --include="*.go"` returns no results (DIST-10)
- `grep "github.com/spf13/cobra" go.mod` present

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
