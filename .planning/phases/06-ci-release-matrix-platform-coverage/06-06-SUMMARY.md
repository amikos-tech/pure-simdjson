---
phase: 06-ci-release-matrix-platform-coverage
plan: 06
subsystem: infra
tags: [release, docs, github-actions, cosign, bootstrap, ci-06]

# Dependency graph
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: prep and publish workflows, prepared-state validation, and immutable R2 publication from 06-05
provides:
  - authoritative release runbook for prep branch to main to tag publication
  - strict release readiness gate backed by the prepared-state contract and main ancestry check
  - repo-local release skill that points agents at the same runbook and gate
affects: [release-operations, bootstrap-docs, agent-guidance, phase-06.1]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Release operators and agents share one checked-in runbook instead of parallel prose or workflow archaeology."
    - "The shell readiness gate delegates source-state truth to scripts/release/assert_prepared_state.py and only adds ancestry/file-presence checks."

key-files:
  created:
    - docs/releases.md
    - scripts/release/check_readiness.sh
    - .agents/skills/pure-simdjson-release/SKILL.md
  modified:
    - docs/bootstrap.md

key-decisions:
  - "docs/releases.md is the single human-readable source of truth for the release-prep -> main -> tag sequence, required repo configuration, artifact layout, and cosign verification commands."
  - "scripts/release/check_readiness.sh --strict reuses assert_prepared_state.py --check-source and adds origin/main ancestry checks instead of re-implementing release-state validation in shell."
  - "docs/bootstrap.md now points operators at the release runbook and mirrors the exact xattr Gatekeeper workaround, while Phase 06.1 owns the fresh-runner public validation boundary."

patterns-established:
  - "Run bash scripts/release/check_readiness.sh --strict --version <semver> before recommending a tag push."
  - "Repo-local agent skills must read docs/releases.md first and refuse hand-upload or non-CI publication paths."

requirements-completed: [CI-06]

# Metrics
duration: 7min
completed: 2026-04-21
---

# Phase 6 Plan 6: Release Runbook and Readiness Surface Summary

**Checked-in release operator guidance with a strict prep-state readiness gate, shared artifact verification commands, and a repo-local agent skill that enforces the same CI-only publish path**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-21T07:48:00Z
- **Completed:** 2026-04-21T07:54:34Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added `docs/releases.md` as the authoritative Phase 6 runbook covering `release-prepare.yml`, `release.yml`, merge-then-tag sequencing, R2/GitHub artifact layout, cosign verification, required repo configuration, and the macOS quarantine workaround.
- Added `scripts/release/check_readiness.sh` with a strict mode that shells out to `assert_prepared_state.py --check-source`, checks `origin/main` ancestry, and refuses missing workflow/runbook prerequisites.
- Added the repo-local `pure-simdjson-release` skill and updated `docs/bootstrap.md` to point operators at the runbook while repeating the exact `xattr -d com.apple.quarantine <path-to-dylib>` guidance.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the release runbook and the readiness gate** - `852f4b8` (feat)
2. **Task 2: Add the repo-local release skill backed by the runbook and readiness gate** - `5df0295` (feat)

## Files Created/Modified

- `docs/releases.md` - authoritative human-facing runbook for the prep branch, merge, tag, publish, and verification flow.
- `scripts/release/check_readiness.sh` - scriptable release readiness gate that reuses the prepared-state contract and checks main ancestry.
- `.agents/skills/pure-simdjson-release/SKILL.md` - narrow repo-local skill that forces agents through the runbook and strict gate.
- `docs/bootstrap.md` - bootstrap operator docs now link to the runbook and repeat the macOS quarantine workaround for downloaded dylibs.

## Decisions Made

- Kept one release source of truth: the runbook documents the exact CI path and the repo-local skill defers to it instead of inventing parallel instructions.
- Made the shell gate intentionally thin: it adds file-presence and `origin/main` ancestry checks but delegates checksum/version coherence to `assert_prepared_state.py`.
- Moved fresh-runner public validation out of the Phase 6 publish instructions and into the explicit Phase `06.1` boundary so the release runbook stays scoped to prep and publish.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no new local setup beyond the release credentials and R2 variables already required by the existing workflows.

## Next Phase Readiness

- Phase `06.1` can now use `docs/releases.md` plus `scripts/release/check_readiness.sh --strict` as the operator surface for fresh-runner/public artifact validation.
- Humans and agents now share the same prep-then-tag release path, reducing drift before Phase 7 release closeout work.

## Self-Check: PASSED

- Verified `.planning/phases/06-ci-release-matrix-platform-coverage/06-06-SUMMARY.md` exists on disk.
- Verified task commits `852f4b8` and `5df0295` exist in `git log --oneline --all`.
