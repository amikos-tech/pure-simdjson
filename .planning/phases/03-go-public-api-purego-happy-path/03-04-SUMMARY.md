---
phase: 03-go-public-api-purego-happy-path
plan: "04"
subsystem: ci
tags: [go, ci, github-actions, race]
requires:
  - phase: 03-01
    provides: deterministic local loader and purego binding foundation
  - phase: 03-02
    provides: parser/doc happy path and typed error surface
  - phase: 03-03
    provides: parser pool and finalizer semantics
provides:
  - Local verification targets for the Go wrapper
  - Narrow five-target wrapper-smoke GitHub Actions workflow
  - Non-interactive branch-scoped observer with explicit run-id and job validation
affects: [phase-04, phase-05, ci]
tech-stack:
  added: []
  patterns: [branch-scoped workflow observation, explicit job-name validation, non-interactive gh polling]
key-files:
  created: [.github/workflows/phase3-go-wrapper-smoke.yml, scripts/phase3-go-wrapper-smoke.sh]
  modified: [Makefile]
key-decisions:
  - "Kept the workflow narrow: build the local shim and run `go test ./... -race`, with no bootstrap, release, signing, or artifact-upload behavior."
  - "Switched the observer from branch-local `workflow_dispatch` to branch-scoped push observation after GitHub rejected dispatch of a workflow file not present on the default branch."
  - "Replaced `gh run watch` with direct `gh run view --jq .status` polling so the helper stays deterministic and non-interactive."
patterns-established:
  - "Remote wrapper proof is tied to a specific branch run id and asserts the five required job names explicitly."
  - "Local `make` targets mirror the workflow commands so the repo has a consistent local-vs-remote proof path."
requirements-completed: [API-03, API-09, API-10]
duration: 24min
completed: 2026-04-16
---

# Phase 03: Go Public API + purego Happy Path Summary

**Local wrapper verification targets plus a five-platform Go-race smoke workflow with a robust branch-scoped observer**

## Performance

- **Duration:** 24 min
- **Started:** 2026-04-16T08:07:55Z
- **Completed:** 2026-04-16T08:31:36Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added `phase3-go-test`, `phase3-go-race`, and `phase3-go-wrapper-remote` targets so the repo can prove the wrapper locally before remote CI.
- Added `.github/workflows/phase3-go-wrapper-smoke.yml` with the exact five Phase 3 wrapper-smoke jobs: Linux amd64, Linux arm64, Darwin amd64, Darwin arm64, and Windows amd64.
- Built `scripts/phase3-go-wrapper-smoke.sh` into a non-interactive branch-scoped observer that pushes the current branch, resolves the matching run id, waits for completion, and verifies that all required job names conclude `success`.
- Observed the final scripted proof on GitHub Actions run `24500326284` for head `9e158a1c7b39812948bca23e84fcaf8b798b46a3`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add local verification targets and the narrow wrapper-smoke workflow** - `1e0ad4e` (build)
2. **Task 2: Add a robust branch-scoped dispatch-and-watch helper, then run it with human approval** - `ad92ea1` (build), `0f03cdd` (build), `dda5931` (build), `c376703` (build), `9e158a1` (build)

## Files Created/Modified
- `Makefile` - Added local Phase 3 verification targets and the remote wrapper helper entrypoint.
- `.github/workflows/phase3-go-wrapper-smoke.yml` - Added the narrow five-target wrapper-smoke workflow.
- `scripts/phase3-go-wrapper-smoke.sh` - Added the branch-scoped run observer and hardened it through multiple GitHub-specific edge cases.

## Decisions Made
- Kept the remote proof tightly scoped to `cargo build --release` plus `go test ./... -race` so Phase 3 does not drift into Phase 5/6 release automation.
- Treated GitHub's default-branch workflow registration rule as a platform constraint and adapted the helper to observe push-triggered branch runs instead of relying on impossible branch-local dispatch behavior.

## Deviations from Plan

- The plan text expected a branch-local `workflow_dispatch` observer. In practice, GitHub returns `404` when dispatching a workflow file that exists only on the feature branch, so the final helper uses a branch-specific `push` trigger plus explicit run-id polling. The behavioral goal stayed the same and the final proof is stronger because it exercises the exact pushed branch state.

## Issues Encountered

- `gh workflow run phase3-go-wrapper-smoke.yml --ref <branch>` failed because GitHub only exposes workflow files from the default branch to that dispatch path.
- The first observer revisions used heredoc-fed `python3 -` filters in pipelines, which caused `SIGPIPE`/exit `141`; switching those filters to `python3 -c` fixed the stdin wiring.

## User Setup Required

- Human approval was required before the first remote push/Actions execution, and `gh auth status -h github.com` had to be valid.

## Next Phase Readiness
- The Go wrapper now has both local and remote proof paths, which clears the Phase 3 verification gate and reduces risk for the broader accessor work in Phase 4 and the distribution/bootstrap work in Phase 5.
- The helper script is reusable for future reruns of the same branch-scoped wrapper-smoke proof without relying on manual Actions UI steps.

---
*Phase: 03-go-public-api-purego-happy-path*
*Completed: 2026-04-16*
