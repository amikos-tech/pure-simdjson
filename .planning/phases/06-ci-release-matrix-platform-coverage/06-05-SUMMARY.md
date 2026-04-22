---
phase: 06-ci-release-matrix-platform-coverage
plan: 05
subsystem: infra
tags: [github-actions, release, cosign, r2, changelog, bootstrap, ci-01, ci-05, ci-06]

# Dependency graph
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: release matrix builds, staged release-tree assembly, and hard native/bootstrap/Alpine smoke gates from Plans 06-01 through 06-04
provides:
  - release-prepare workflow that rewrites version/checksum source state, updates CHANGELOG.md, and pushes a mergeable release-prep branch
  - release workflow that blocks off-main tags, asserts rebuilt manifest digests against committed bootstrap state, signs blobs with cosign, and publishes immutable R2 plus GitHub Release assets
  - prepared-state validator plus R2 upload helper for later readiness/runbook automation
affects: [release-prep, release-publish, bootstrap-state, r2-distribution, github-releases]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Release prep happens on a normal source branch first: CI rewrites version.go, checksums.go, and CHANGELOG.md, then pushes release-prep/v<version> for PR+merge before any tag exists."
    - "Tag publish validates source coherence instead of rewriting it: a rebuilt manifest must match committed bootstrap checksums before SHA256SUMS generation, cosign signing, and upload."
    - "R2 and GitHub Releases publish from one signed staged root: raw cache-layout artifacts keep their immutable R2 paths while flat GitHub asset copies inherit the same bytes plus .sig/.pem sidecars."

key-files:
  created:
    - CHANGELOG.md
    - scripts/release/assert_prepared_state.py
    - scripts/release/publish_r2.sh
  modified:
    - .github/actions/build-shared-library/action.yml
    - .github/workflows/release-prepare.yml
    - .github/workflows/release.yml

key-decisions:
  - "release-prepare.yml now takes the target semver as workflow_dispatch input and writes release-facing source state before tagging, so the published tag points at already-prepared source."
  - "release.yml now gates publication with a dedicated verify-tag-source job that checks both origin/main ancestry and committed bootstrap source coherence before any build starts."
  - "The publish path signs and verifies the raw staged blobs in R2 layout first, then copies those exact bytes and sidecars into flat GitHub Release asset names so both destinations carry the same signed payload."

patterns-established:
  - "Use scripts/release/assert_prepared_state.py as the single digest/version coherence gate for both source-only readiness checks and rebuilt-manifest validation."
  - "Keep R2 immutability in a dedicated shell helper that refuses any non-empty prefix before aws s3 recursive upload."
  - "Emit human handoff instructions directly in GITHUB_STEP_SUMMARY from the prep workflow so the merge-then-tag sequence is explicit in CI."

requirements-completed: [CI-01, CI-05, CI-06]

# Metrics
duration: 15min
completed: 2026-04-21
---

# Phase 6 Plan 5: Two-Workflow Release Path Summary

**Release-prep source rewrites plus main-anchored tag publishing with digest-coherence checks, cosign signatures, immutable R2 upload, and GitHub Release assets**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-21T07:31:00Z
- **Completed:** 2026-04-21T07:45:52Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added `release-prepare.yml` inputs, concurrency, pinned action usage, release-prep branch creation, changelog updates, prepared-source validation, and explicit merge-then-tag handoff instructions.
- Added `assert_prepared_state.py` to validate committed `version.go` / `checksums.go` state both in `--check-source` mode and against rebuilt manifest digests with GitHub summary mismatch tables.
- Added the signed publish path in `release.yml` plus `publish_r2.sh`: off-main tags now fail early, the rebuilt manifest must match committed bootstrap state, `SHA256SUMS` and raw blobs are cosign-signed and verified, R2 refuses overwrites, and GitHub Releases publish the flat platform-tagged assets with `.sig` / `.pem` sidecars.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement the pre-tag release-preparation workflow and source rewrite path** - `cdf4326` (feat)
2. **Task 2: Implement tag-publish release workflow with digest coherence, signing, and immutable R2 upload** - `fd1b1a8` (feat)

## Files Created/Modified

- `CHANGELOG.md` - adds the checked-in Keep a Changelog artifact that the prep workflow now updates before tagging.
- `scripts/release/assert_prepared_state.py` - validates manifest/source coherence and exposes the `--check-source` gate for later readiness automation.
- `scripts/release/publish_r2.sh` - enforces immutable-prefix checks and performs the R2 recursive upload for the signed staged release tree.
- `.github/workflows/release-prepare.yml` - implements the release-prep branch flow, changelog/source rewrites, prepared-state validation, and operator handoff summary.
- `.github/workflows/release.yml` - implements main-anchored tag validation, rebuilt-manifest digest checks, cosign sign/verify, R2 publish, and GitHub Release publication.
- `.github/actions/build-shared-library/action.yml` - pins the nested Windows MSVC setup action so the release path does not retain an unpinned helper under the workflow layer.

## Decisions Made

- Kept the prep and publish workflows as one shared release system: both still reuse the staged manifest/tree structure from Waves 1-4, but only the prep workflow rewrites source while the tag workflow only verifies source coherence.
- Added a dedicated `verify-tag-source` job rather than sprinkling ancestry and source-state checks into every build job, which guarantees off-main tags fail before any artifact build starts.
- Generated GitHub Release assets from copies of the already-signed raw staged blobs instead of renaming the staged R2 files in place, which preserves the bootstrap/R2 naming contract and still gives GitHub the flat platform-tagged filenames.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Pinned the hidden Windows MSVC helper inside the shared build action**
- **Found during:** Task 1 (Implement the pre-tag release-preparation workflow and source rewrite path)
- **Issue:** The release workflows were updated to use pinned actions, but `.github/actions/build-shared-library/action.yml` still invoked `ilammy/msvc-dev-cmd@v1`. That left the actual release path with an unpinned external action under the workflow layer, violating the plan's pinned-action contract.
- **Fix:** Pinned the nested MSVC helper to the same full commit SHA already used directly by the workflows.
- **Files modified:** `.github/actions/build-shared-library/action.yml`
- **Verification:** `git diff -- .github/actions/build-shared-library/action.yml` plus the Task 1 workflow grep gate showing pinned release-path actions
- **Committed in:** `cdf4326`

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The auto-fix was necessary to make the real release path match the plan's supply-chain hardening requirement. No scope creep beyond the release workflow contract.

## Issues Encountered

None.

## User Setup Required

None - no new local setup beyond the existing release secrets/credentials already implied by the release workflows.

## Next Phase Readiness

- Plan `06-06` can now document and gate a real prep-then-tag process instead of a proposed one; the ready-made `--check-source` mode can back the readiness script directly.
- The release path now has explicit machine-enforced anchors for source coherence, off-main tag rejection, signing, and immutable publication, so the remaining work is operator/runbook surface rather than more release plumbing.

## Self-Check: PASSED

- Verified `.planning/phases/06-ci-release-matrix-platform-coverage/06-05-SUMMARY.md` exists on disk.
- Verified task commits `cdf4326` and `fd1b1a8` exist in `git log --oneline --all`.
