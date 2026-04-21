---
phase: 06-ci-release-matrix-platform-coverage
plan: 02
subsystem: infra
tags: [github-actions, release, linux, manylinux, glibc, objdump, bootstrap, ci]

# Dependency graph
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: shared release composite actions, package helper, and bootstrap manifest/state tooling from Plan 06-01
provides:
  - pinned manylinux linux release jobs for linux/amd64 and linux/arm64 in release-prepare and release workflows
  - explicit linux/arm64 4K page-size proof with a persisted prep-workflow evidence file
  - reusable linux glibc-floor and export-surface verification before packaging/upload
affects: [release-prepare, release-publish, linux-artifacts, bootstrap-staging]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Linux release workflows stay thin: workflows declare runner/image tuples and staging, while the shared build action delegates manylinux execution to a repo-local script."
    - "Linux verification is repo-driven: objdump and nm gates run before packaging so staged artifacts are already proven against the glibc floor and header-defined ABI surface."

key-files:
  created:
    - .github/workflows/release-prepare.yml
    - .github/workflows/release.yml
    - scripts/release/build_linux_manylinux.sh
    - scripts/release/verify_glibc_floor.sh
  modified:
    - .github/actions/build-shared-library/action.yml

key-decisions:
  - "The shared build action now hands manylinux execution to scripts/release/build_linux_manylinux.sh so workflow YAML does not duplicate docker mount logic or arm64 page-size enforcement."
  - "linux/arm64 page-size proof runs as both an explicit workflow step and a builder-side guard; the prep workflow also uploads linux-arm64-pagesize.txt with the staged artifact bundle."
  - "verify_glibc_floor.sh derives the expected pure_simdjson export set from include/pure_simdjson.h instead of freezing a separate symbol list in CI."

patterns-established:
  - "Upload per-platform linux bundles from matrix jobs, then re-download and merge them into a staging artifact for later publish plans."
  - "Validate manylinux tuple correctness in the repo script itself, including exact rust target and pinned image digest checks."
  - "Use objdump GLIBC version extraction plus header-derived nm comparison as the reusable linux publish gate."

requirements-completed: [PLAT-01, PLAT-02, CI-01, CI-03]

# Metrics
duration: 11min
completed: 2026-04-21
---

# Phase 6 Plan 2: Linux Manylinux Release Summary

**Pinned manylinux linux release jobs with explicit arm64 4K page-size proof, staged bootstrap artifact bundles, and a reusable glibc/export verification gate**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-21T06:31:27Z
- **Completed:** 2026-04-21T06:42:55Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added `release-prepare.yml` and `release.yml` Linux jobs for `linux/amd64` and `linux/arm64`, both pinned to exact manylinux2014 image digests and routed through the shared build/package actions.
- Added `scripts/release/build_linux_manylinux.sh`, which validates the exact Linux tuple, runs the manylinux docker build, and enforces the arm64 `PAGE_SIZE=4096` proof in the versioned build path.
- Added `scripts/release/verify_glibc_floor.sh` and wired it before packaging/upload so Linux artifacts are blocked on both a `GLIBC_2.17` floor and the expected `pure_simdjson_` export surface.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add manylinux linux build jobs to release-prepare and release workflows** - `79c3724` (feat)
2. **Task 2: Add the glibc-floor verification gate and linux export audit** - `d9df9a8` (feat)

## Files Created/Modified

- `.github/workflows/release-prepare.yml` - Linux prep workflow with pinned manylinux matrix jobs, reproducibility enabled, arm64 page-size proof upload, and merged staging artifact output.
- `.github/workflows/release.yml` - Linux tag/workflow-dispatch build workflow with pinned manylinux matrix jobs and merged staging artifact output for later publish plans.
- `scripts/release/build_linux_manylinux.sh` - Repo-local manylinux build/proof helper that validates the exact Linux tuples, performs arm64 page-size proofing, and runs the dockerized cargo build.
- `scripts/release/verify_glibc_floor.sh` - Reusable Linux gate that inspects `objdump -T` and `nm -D --defined-only` before artifacts can be staged.
- `.github/actions/build-shared-library/action.yml` - Shared action now delegates Linux manylinux execution to the repo script instead of inlining docker mounts.

## Decisions Made

- Kept the workflows thin and reusable by pushing Linux-specific docker execution and tuple validation into `scripts/release/build_linux_manylinux.sh`, while still using the shared build action from Plan 06-01.
- Recorded the arm64 page-size proof twice on purpose: once as an explicit workflow step for operator visibility and once inside the build script so the constraint stays attached to the versioned build path.
- Used the committed header as the export-surface source of truth for the Linux gate, avoiding a second hardcoded symbol list that could drift from the ABI contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Mounted reproducibility target directories inside the manylinux build path**
- **Found during:** Task 1 (Add manylinux linux build jobs to release-prepare and release workflows)
- **Issue:** The shared manylinux build path from Plan 06-01 only mounted the repo workspace and cargo/rustup homes. Task 1 needed reproducibility builds in temporary target directories outside the workspace, which would have failed inside docker.
- **Fix:** Routed manylinux execution through `scripts/release/build_linux_manylinux.sh` and mounted the explicit target directory alongside cargo/rustup state.
- **Files modified:** `.github/actions/build-shared-library/action.yml`, `scripts/release/build_linux_manylinux.sh`
- **Verification:** `bash -n scripts/release/build_linux_manylinux.sh` plus the Task 1 workflow grep gate
- **Committed in:** `79c3724`

**2. [Rule 3 - Blocking] Removed the ripgrep runtime dependency from the glibc gate**
- **Found during:** Task 2 (Add the glibc-floor verification gate and linux export audit)
- **Issue:** The first cut of `verify_glibc_floor.sh` used `rg`, but the shared Linux setup guarantees `binutils`, not ripgrep. The gate would have failed on clean runners before performing the actual verification.
- **Fix:** Rewrote GLIBC and symbol extraction to use `grep -oE`, `awk`, `sed`, `sort`, and `comm` only.
- **Files modified:** `scripts/release/verify_glibc_floor.sh`
- **Verification:** `bash -n scripts/release/verify_glibc_floor.sh` plus the Task 2 grep gate
- **Committed in:** `d9df9a8`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were required for the Linux workflow path to execute on clean runners. No scope creep beyond the task contract.

## Issues Encountered

- The repo did not yet contain `release-prepare.yml` or `release.yml`, so this plan established the Linux portions of those workflows from scratch instead of patching an existing release pipeline.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Linux staging artifacts now already match the bootstrap cache/R2 naming contract and are available as merged workflow artifacts for later phase plans.
- Later Phase 6 plans can layer smoke binaries, macOS/Windows jobs, checksum rewrites, and publish/signing logic onto the new workflow skeleton without reintroducing Linux-specific docker logic.
- The glibc/export verification step is reusable for any future Linux publish job that consumes a staged `.so`.

## Self-Check: PASSED

- Verified all created/modified task files and `.planning/phases/06-ci-release-matrix-platform-coverage/06-02-SUMMARY.md` exist on disk.
- Verified task commits `79c3724` and `d9df9a8` are present in `git log --oneline --all`.
