---
phase: 06-ci-release-matrix-platform-coverage
plan: 01
subsystem: infra
tags: [github-actions, release, rust, bootstrap, checksums, manifest, packaging, ci-01, ci-05]

# Dependency graph
requires:
  - phase: 02-rust-shim-minimal-parse-path
    provides: release-library build baseline plus tests/smoke/minimal_parse.c for later release verification
  - phase: 05-bootstrap-distribution
    provides: bootstrap naming/version/checksum contract in internal/bootstrap/url.go, version.go, and checksums.go
provides:
  - reusable setup/build/package/verify composite actions for Phase 6 release workflows
  - repo-pinned Rust toolchain selection via rust-toolchain.toml
  - packaging helper that emits R2/cache and GitHub asset names plus a checksum-bearing manifest entry
  - deterministic bootstrap version/checksum rewrite script with unittest coverage
affects: [release-prepare, release-publish, artifact-verification, bootstrap-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Composite release action layering: setup-rust installs the repo-pinned toolchain once, build-shared-library adds deterministic flags/repro checks, package-shared-artifact stages release names, verify-shared-artifact hard-requires later smoke commands"
    - "Manifest-driven bootstrap state generation: release metadata rewrites version.go and checksums.go from entries[].sha256 without re-hashing local_path"
    - "Temp-workspace rewrite testing: copy the script and target Go files into a TemporaryDirectory so rewrite tests prove idempotence without mutating the real repo"

key-files:
  created:
    - rust-toolchain.toml
    - .github/actions/setup-rust/action.yml
    - .github/actions/build-shared-library/action.yml
    - .github/actions/package-shared-artifact/action.yml
    - .github/actions/verify-shared-artifact/action.yml
    - scripts/release/package_shared_artifact.sh
    - scripts/release/update_bootstrap_release_state.py
    - scripts/release/test_update_bootstrap_release_state.py
  modified:
    - internal/bootstrap/version.go
    - internal/bootstrap/checksums.go

key-decisions:
  - "Pinned Rust setup is handled by a local setup-rust composite action that reads rust-toolchain.toml directly, avoiding floating-toolchain behavior in later workflows."
  - "verify-shared-artifact fails when later plans do not provide native ABI and minimal_parse smoke commands, so export audits cannot accidentally become the only release gate."
  - "update_bootstrap_release_state.py resolves the repo root from its own path and rewrites copied temp workspaces in tests, which keeps the repository unchanged during verification."

patterns-established:
  - "Use one packaging script to derive both the R2/cache filename and the GitHub flat-namespace asset name, then compute sha256 from the exact staged R2 bytes."
  - "Preserve generated-file comment blocks by rewriting only the Version constant and Checksums map body rather than templating full Go files from scratch."
  - "Validate release manifests strictly: exact top-level keys, exact entry keys, exact five supported tuples, exact checksum-key naming, and 64-char lowercase sha256 values."

requirements-completed: [CI-01, CI-05]

# Metrics
duration: 5min
completed: 2026-04-21
---

# Phase 6 Plan 1: Release Scaffold Summary

**Reusable release composite actions, deterministic artifact packaging, and a manifest-driven bootstrap checksum/version rewriter for the Phase 6 CI release pipeline**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-21T06:23:37Z
- **Completed:** 2026-04-21T06:28:20Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added `rust-toolchain.toml` plus four local composite actions so later Phase 6 workflows can share pinned Rust setup, deterministic shared-library builds, artifact packaging, and verification entry points instead of duplicating YAML inline.
- Added `scripts/release/package_shared_artifact.sh`, which renames staged libraries into both the bootstrap cache/R2 name and the GitHub flat-namespace asset name, then emits a one-entry manifest carrying `goos`, `goarch`, `rust_target`, `r2_key`, `github_asset_name`, `local_path`, and `sha256`.
- Added `scripts/release/update_bootstrap_release_state.py` with unittest coverage that rewrites `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go` deterministically from manifest data while preserving the surrounding comment blocks.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add reusable release actions and the packaging helper** - `d3ebae6` (feat)
2. **Task 2: Add deterministic bootstrap release-state generation and tests** - `44656b5` (feat)

## Files Created/Modified

- `rust-toolchain.toml` - pins the repo to Rust `1.89.0` with a checked-in toolchain file for release builds.
- `.github/actions/setup-rust/action.yml` - installs the pinned Rust toolchain and optional targets/helper packages for downstream release steps.
- `.github/actions/build-shared-library/action.yml` - applies deterministic env flags, optional manylinux/docker execution, and optional reproducibility hashing before exposing the built library path.
- `.github/actions/package-shared-artifact/action.yml` - wraps the packaging helper and exports manifest/checksum outputs for later workflows.
- `.github/actions/verify-shared-artifact/action.yml` - performs supplemental platform audits and hard-requires future native ABI plus `minimal_parse.c` smoke commands.
- `scripts/release/package_shared_artifact.sh` - stages R2/cache and GitHub asset copies and writes a checksum-bearing manifest entry from the staged bytes.
- `scripts/release/update_bootstrap_release_state.py` - validates release manifests and rewrites `version.go` / `checksums.go` in exact platform order.
- `scripts/release/test_update_bootstrap_release_state.py` - covers temp-workspace rewrites, idempotence, and invalid/missing sha256 rejection.
- `internal/bootstrap/version.go` - documents that the release-state rewrite script owns this constant during CI release prep.
- `internal/bootstrap/checksums.go` - documents that the release-state rewrite script owns this generated map during CI release prep.

## Decisions Made

- Used a local `toolchain-file` contract in the composite actions so later workflows must consume `rust-toolchain.toml` instead of a floating `stable` alias.
- Kept `package_shared_artifact.sh` as the single naming/manifest source for both R2/cache and GitHub asset outputs; later workflows can aggregate manifests without reconstructing names in YAML.
- Made the verification action fail fast when the two release smoke commands are missing. That preserves CI-04’s intent that export audits are only supplemental evidence.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Later Phase 6 workflow plans can now call a shared setup/build/package/verify surface instead of re-encoding Rust toolchain, deterministic flags, or artifact names in workflow YAML.
- `update_bootstrap_release_state.py` is ready for `release-prepare.yml` to feed it aggregated manifest data and rewrite the bootstrap version/checksum source before tagging.
- `verify-shared-artifact` intentionally blocks until later plans wire the concrete native ABI-load smoke harness and `tests/smoke/minimal_parse.c` command, which keeps the release gate honest.

## Self-Check: PASSED

- Verified all 10 task files plus `.planning/phases/06-ci-release-matrix-platform-coverage/06-01-SUMMARY.md` exist on disk.
- Verified task commits `d3ebae6` and `44656b5` are present in `git log --oneline --all`.
