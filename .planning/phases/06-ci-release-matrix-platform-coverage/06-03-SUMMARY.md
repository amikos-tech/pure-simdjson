---
phase: 06-ci-release-matrix-platform-coverage
plan: 03
subsystem: infra
tags: [github-actions, release, darwin, windows, codesign, msvc, bootstrap, ci]

# Dependency graph
requires:
  - phase: 06-ci-release-matrix-platform-coverage
    provides: shared release composite actions and the linux release workflow skeleton from Plans 06-01 and 06-02
provides:
  - darwin amd64 and arm64 prep/publish jobs with ad-hoc codesign, thin-binary enforcement, and staged artifact bundles
  - windows amd64 prep/publish jobs with MSVC, git long-path enable, import-library preservation, and recorded dependency audits
  - forward-slash normalized shared-action artifact paths so the windows bash/pwsh release flow can reuse the Plan 06-01 helpers
affects: [release-prepare, release-publish, darwin-artifacts, windows-artifacts, staged-release-tree]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Workflow matrix rows carry the expected public filenames, and each job asserts the shared packager outputs exactly match the bootstrap naming contract."
    - "Darwin release gating stays native-runner-first: build via the shared action, ad-hoc codesign the dylib, fail on fat/universal output, then run the shared staged-artifact verification action."
    - "Windows release gating bridges bash and PowerShell inside the shared verify action inputs: dumpbin/export comparison runs in bash, while the staged minimal_parse smoke uses MSVC through a generated PowerShell script."

key-files:
  created: []
  modified:
    - .github/workflows/release-prepare.yml
    - .github/workflows/release.yml
    - .github/actions/build-shared-library/action.yml
    - .github/actions/package-shared-artifact/action.yml
    - scripts/release/package_shared_artifact.sh

key-decisions:
  - "The darwin jobs make the expected GitHub asset names explicit in the matrix and assert them after packaging, so the workflow file itself documents the public bootstrap contract."
  - "The windows jobs preserve pure_simdjson.dll.lib alongside the staged DLL and dependency report, keeping the local cargo naming unchanged while making the release bundle usable for later smoke assembly and documentation."
  - "Shared release helpers now emit forward-slash absolute paths and Python-created temp directories so the same bash-based composite actions work on windows runners without diverging into a second packaging path."

patterns-established:
  - "Use the shared verify-shared-artifact action as the final native gate after packaging, with platform-specific export comparison and minimal_parse commands injected by the workflow."
  - "Persist windows runtime dependency evidence directly into the staged artifact bundle so later plans can consume or publish it without rebuilding."
  - "Promote the merged staging artifact names from linux-only to release-wide artifacts as soon as a second platform joins the matrix."

requirements-completed: [PLAT-03, PLAT-04, PLAT-05, CI-02]

# Metrics
duration: 15min
completed: 2026-04-21
---

# Phase 6 Plan 3: Darwin and Windows Release Jobs Summary

**Native darwin and windows release jobs with ad-hoc-signed thin dylibs, MSVC-named DLL packaging, and staged export/smoke verification wired into both release workflows**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-21T06:47:42Z
- **Completed:** 2026-04-21T07:02:33Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added darwin/amd64 and darwin/arm64 prep/build jobs on `macos-15-intel` and `macos-15`, routed through the shared build/package/verify actions with ad-hoc codesign and explicit thin-binary enforcement.
- Added a windows/amd64 prep/build job on `windows-latest` with pinned MSVC setup, `core.longpaths` enablement, `dumpbin /EXPORTS`, `dumpbin /DEPENDENTS`, staged import-library preservation, and the exact `pure_simdjson-msvc.dll` / `pure_simdjson-windows-amd64-msvc.dll` naming split.
- Expanded the merged staging artifacts from Linux-only to cross-platform release trees so later Phase 6 plans can assemble staged smoke inputs without rebuilding darwin or windows outputs.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add darwin release jobs with ad-hoc codesign and thin-binary verification** - `f8299ad` (feat)
2. **Task 2: Add windows release job with MSVC, long-path enable, and contract naming** - `ba48cf2` (feat)

## Files Created/Modified

- `.github/workflows/release-prepare.yml` - adds darwin and windows pre-tag build/package/verify jobs plus merged cross-platform prep staging.
- `.github/workflows/release.yml` - adds darwin and windows tag/workflow-dispatch build/package/verify jobs plus merged cross-platform release staging.
- `.github/actions/build-shared-library/action.yml` - normalizes built artifact paths and reproducible temp directories so the shared action can drive windows jobs safely.
- `.github/actions/package-shared-artifact/action.yml` - normalizes packaged asset-path outputs for downstream windows workflow consumption.
- `scripts/release/package_shared_artifact.sh` - emits forward-slash absolute artifact paths so the staged windows verification flow can reuse the packaging helper unchanged.

## Decisions Made

- Kept the darwin and windows jobs on the same shared build/package/verify surface as Linux rather than creating platform-specific release scripts, so the release workflows stay one system instead of three branches.
- Made the workflow matrix carry the expected public filenames and asserted them in-job, which turns the bootstrap naming contract into an executable workflow invariant instead of an implicit helper detail.
- Preserved the MSVC import library and dependency report inside the staged windows bundle, because later release assembly and documentation work need that evidence without rerunning the build.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Normalized shared helper paths for windows bash/pwsh interoperability**
- **Found during:** Task 2 (Add windows release job with MSVC, long-path enable, and contract naming)
- **Issue:** The shared build/package helpers emitted native windows backslash paths and `mktemp` templates rooted in `RUNNER_TEMP`, but the new windows release jobs are required to flow through bash-based composite actions and staged-artifact verification. On clean runners that mismatch would have broken reproducible builds, packaging, and later smoke commands.
- **Fix:** Updated the shared build and packaging helpers to emit forward-slash absolute paths, switched reproducible temp-dir creation to Python, and kept the windows workflow staging paths relative where the bash steps write additional files.
- **Files modified:** `.github/actions/build-shared-library/action.yml`, `.github/actions/package-shared-artifact/action.yml`, `scripts/release/package_shared_artifact.sh`, `.github/workflows/release-prepare.yml`, `.github/workflows/release.yml`
- **Verification:** `ruby -e 'require "yaml"; %w[.github/workflows/release-prepare.yml .github/workflows/release.yml .github/actions/build-shared-library/action.yml .github/actions/package-shared-artifact/action.yml].each { |p| YAML.load_file(p) }; puts "yaml ok"'`, `bash -n scripts/release/package_shared_artifact.sh`, and the Task 2 workflow grep gate
- **Committed in:** `ba48cf2`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The deviation was required for the shared release helpers to work on windows runners. No scope creep beyond the planned darwin/windows release path.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `release-prepare.yml` and `release.yml` now stage Linux, darwin, and windows artifacts under merged cross-platform artifacts, so Plan 06-04 can assemble loopback smoke inputs without rebuilding those platforms.
- The darwin artifacts are already signed, thin-checked, and smoke-gated; the windows bundle now carries both the renamed public DLL and the preserved import-library/dependency evidence needed for later publish and documentation work.
- Alpine validation, checksum rewriting, signing, and publish logic remain for the later Phase 6 plans.

## Self-Check: PASSED

- Verified `.planning/phases/06-ci-release-matrix-platform-coverage/06-03-SUMMARY.md` exists on disk.
- Verified task commits `f8299ad` and `ba48cf2` are present in `git log --oneline --all`.
