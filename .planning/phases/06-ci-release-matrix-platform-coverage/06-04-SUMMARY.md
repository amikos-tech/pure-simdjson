---
phase: 06-ci-release-matrix-platform-coverage
plan: 04
subsystem: infra
tags: [github-actions, release, smoke-tests, bootstrap, alpine, abi]

# Dependency graph
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: linux manylinux release jobs, darwin/windows staged bundles, and shared release packaging from Plans 06-01 through 06-03
provides:
  - reusable staged-tree assembly plus native, packaged-bootstrap, and Alpine smoke entrypoints
  - workflow-native CI-04 gating through ffi-export coverage and minimal_parse against staged artifacts
  - loopback bootstrap smoke and pinned Alpine escape-hatch validation in both release workflows
affects: [release-prepare, release-publish, staged-release-tree, bootstrap-validation, alpine-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Release smoke execution lives in repo-local scripts, while workflow YAML just supplies platform tuples, staged artifact paths, and pinned action plumbing."
    - "Per-platform package manifests are promoted to unique manifest-row files before artifact merge, then reassembled into one ordered manifest for bootstrap checksum rewrites and staged-tree smoke."
    - "Packaged bootstrap smoke runs only after release-state generation has materialized real checksum values in the checked-out source used by go run."

key-files:
  created:
    - scripts/release/assemble_staged_release_tree.sh
    - scripts/release/run_native_smoke.sh
    - scripts/release/run_go_packaged_smoke.sh
    - scripts/release/run_alpine_smoke.sh
    - tests/smoke/ffi_export_surface.c
    - tests/smoke/go_bootstrap_smoke.go
  modified:
    - .github/workflows/release-prepare.yml
    - .github/workflows/release.yml

key-decisions:
  - "CI-04 now executes through scripts/release/run_native_smoke.sh so every platform runs the same audit -> ffi_export_surface.c -> minimal_parse.c sequence instead of embedding per-platform shell fragments in workflow YAML."
  - "The staged bootstrap smoke consumes one exact v<version>/<os>-<arch>/<libname> tree assembled from package manifests and staged artifacts, not ad hoc workflow paths."
  - "Both staging jobs rewrite bootstrap release state from the combined manifest before go run, which keeps the packaged-artifact smoke aligned with the real checksum contract from Phase 5."

patterns-established:
  - "Use manifest-<platform>.json row files inside uploaded bundles to avoid merge collisions when actions/download-artifact flattens per-platform release outputs."
  - "Drive Alpine validation through one pinned ALPINE_IMAGE_REF constant shared by workflow env and the smoke script, with PURE_SIMDJSON_LIB_PATH limited to that dedicated path."
  - "Upload the combined manifest and assembled staged release tree as the workflow-inspectable smoke inputs, not the unstructured merged bundle."

requirements-completed: [PLAT-06, CI-04, CI-07]

# Metrics
duration: 44min
completed: 2026-04-21
---

# Phase 6 Plan 4: Release Smoke Gates Summary

**Native ABI-load coverage, loopback bootstrap smoke, and pinned Alpine escape-hatch validation wired into both release workflows**

## Performance

- **Duration:** 44 min
- **Completed:** 2026-04-21
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Added reusable release verification entrypoints:
  - `scripts/release/assemble_staged_release_tree.sh` builds the exact `v<version>/<os>-<arch>/<libname>` tree from per-platform package bundles and manifest rows.
  - `scripts/release/run_native_smoke.sh` performs platform-native audits, runs the new `ffi_export_surface.c` ABI harness, then proves `tests/smoke/minimal_parse.c` against the same staged artifact.
  - `scripts/release/run_go_packaged_smoke.sh` serves the staged tree over loopback HTTP and forces the real bootstrap path via `PURE_SIMDJSON_BINARY_MIRROR` plus `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1`.
  - `scripts/release/run_alpine_smoke.sh` enforces one exact pinned `alpine:latest@sha256:...` reference and validates only the documented `PURE_SIMDJSON_LIB_PATH` escape hatch.
- Added smoke harnesses:
  - `tests/smoke/ffi_export_surface.c` dynamically loads the staged library with `dlopen`/`dlsym` or `LoadLibraryW`/`GetProcAddress`, resolves every public v0.1 export from `include/pure_simdjson.h`, and invokes each one at least once.
  - `tests/smoke/go_bootstrap_smoke.go` proves `purejson.NewParser()` bootstrap plus a literal `42` parse from a real Go consumer.
- Rewired `release-prepare.yml` and `release.yml` so all platform jobs run the repo-local native smoke script, both workflows assemble a real staged release tree, both staging jobs run the loopback bootstrap smoke after checksum generation, and both workflows block on the pinned Alpine validation job.

## Task Commits

1. **Task 1: Add reusable native, bootstrap-path, and Alpine smoke scripts** - `99bf47f` (feat)
2. **Task 2: Wire native, Go packaged-artifact, and Alpine smoke gates into both workflows** - `e4e7574` (feat)

## Files Created/Modified

- `.github/workflows/release-prepare.yml` - now promotes per-platform manifest rows, runs `run_native_smoke.sh`, includes the pinned Alpine job, assembles a combined manifest plus staged release tree, rewrites bootstrap state in-workflow, and runs the loopback packaged-artifact smoke.
- `.github/workflows/release.yml` - mirrors the same native/bootstrap/Alpine smoke structure for the publish workflow skeleton.
- `scripts/release/assemble_staged_release_tree.sh` - validates exactly five manifest rows and copies packaged artifacts into the exact bootstrap/R2 layout.
- `scripts/release/run_native_smoke.sh` - shared CI-04 gate for linux, darwin, and windows staged artifacts.
- `scripts/release/run_go_packaged_smoke.sh` - shared loopback bootstrap smoke using only the mirror/fallback env contract from `internal/bootstrap/bootstrap.go`.
- `scripts/release/run_alpine_smoke.sh` - pinned-container Alpine escape-hatch validation.
- `tests/smoke/ffi_export_surface.c` - exhaustive ABI-load harness for every exported v0.1 symbol.
- `tests/smoke/go_bootstrap_smoke.go` - minimal Go consumer smoke binary for staged-artifact bootstrap validation.

## Decisions Made

- Kept all hard smoke logic in repo-local scripts so the release workflows remain declarative and later plans can add prep/publish policy without re-encoding platform commands.
- Used unique `manifest-<platform>.json` rows in uploaded bundles because the old flat `manifest.json` naming would be overwritten during merged artifact downloads and made staged-tree assembly impossible.
- Reused `scripts/release/update_bootstrap_release_state.py` inside the staging jobs before running the Go smoke so the packaged-artifact bootstrap path sees real checksum data instead of the dev-time empty `Checksums` map.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Aligned the ABI smoke with the runtime offset contract**
- **Found during:** Task 1 verification
- **Issue:** The first cut of `ffi_export_surface.c` incorrectly treated `pure_simdjson_parser_get_last_error_offset()` returning `UINT64_MAX` as a failure after invalid JSON. The native contract and Rust smoke tests explicitly use that sentinel for "unknown offset."
- **Fix:** Updated the harness to assert the documented sentinel instead of inventing a non-zero requirement.
- **Files modified:** `tests/smoke/ffi_export_surface.c`
- **Verification:** `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64`
- **Committed in:** `99bf47f`

**2. [Rule 3 - Blocking] Prevented manifest-row collisions in merged staging artifacts**
- **Found during:** Task 2 implementation
- **Issue:** Every platform bundle carried `manifest.json`, so `actions/download-artifact --merge-multiple` would overwrite earlier rows and make exact five-row staged-tree assembly impossible.
- **Fix:** Promoted each row to `manifest-<platform>.json` before upload and had the staging jobs rebuild one ordered combined manifest from those files.
- **Files modified:** `.github/workflows/release-prepare.yml`, `.github/workflows/release.yml`
- **Verification:** `ruby -e 'require "yaml"; [".github/workflows/release-prepare.yml", ".github/workflows/release.yml"].each { |p| YAML.load_file(p) }; puts "yaml ok"'` and the Task 2 grep gate
- **Committed in:** `e4e7574`

**3. [Rule 2 - Missing critical functionality] Added a pinned Go toolchain to the staging jobs**
- **Found during:** Task 2 implementation
- **Issue:** The new staging jobs invoke `go run ./tests/smoke/go_bootstrap_smoke.go`, but the workflow skeleton did not guarantee Go was installed on those jobs.
- **Fix:** Added pinned `actions/setup-go` in both staging jobs before the packaged-artifact smoke step.
- **Files modified:** `.github/workflows/release-prepare.yml`, `.github/workflows/release.yml`
- **Verification:** `ruby -e 'require "yaml"; [".github/workflows/release-prepare.yml", ".github/workflows/release.yml"].each { |p| YAML.load_file(p) }; puts "yaml ok"'` and the Task 2 grep gate
- **Committed in:** `e4e7574`

---

**Total deviations:** 3 auto-fixed (1 bug, 1 blocking, 1 missing critical prerequisite)
**Impact on plan:** All three fixes were required to make the smoke-gate path executable and faithful to the Phase 5 bootstrap contract.

## Issues Encountered

None remaining.

## User Setup Required

None - no new secrets or local setup required for this plan.

## Next Phase Readiness

- Plan `06-05` can build directly on the combined-manifest and staged-root flow to add prepared-state commits, digest-coherence assertions, signing, and upload without reworking smoke execution.
- Plan `06-06` can document one actual release path instead of a planned one: native CI-04 gating, loopback bootstrap smoke, and pinned Alpine validation are now real workflow steps.

## Self-Check: PASSED

- Verified `.planning/phases/06-ci-release-matrix-platform-coverage/06-04-SUMMARY.md` exists on disk.
- Verified task commits `99bf47f` and `e4e7574` exist in `git log --oneline --all`.
