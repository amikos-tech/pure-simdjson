---
phase: 06-ci-release-matrix-platform-coverage
reviewed: 2026-04-21T08:07:32Z
depth: standard
files_reviewed: 27
files_reviewed_list:
  - .agents/skills/pure-simdjson-release/SKILL.md
  - .github/actions/build-shared-library/action.yml
  - .github/actions/package-shared-artifact/action.yml
  - .github/actions/setup-rust/action.yml
  - .github/actions/verify-shared-artifact/action.yml
  - .github/workflows/release-prepare.yml
  - .github/workflows/release.yml
  - CHANGELOG.md
  - docs/bootstrap.md
  - docs/releases.md
  - internal/bootstrap/checksums.go
  - internal/bootstrap/version.go
  - rust-toolchain.toml
  - scripts/release/assemble_staged_release_tree.sh
  - scripts/release/assert_prepared_state.py
  - scripts/release/build_linux_manylinux.sh
  - scripts/release/check_readiness.sh
  - scripts/release/package_shared_artifact.sh
  - scripts/release/publish_r2.sh
  - scripts/release/run_alpine_smoke.sh
  - scripts/release/run_go_packaged_smoke.sh
  - scripts/release/run_native_smoke.sh
  - scripts/release/test_update_bootstrap_release_state.py
  - scripts/release/update_bootstrap_release_state.py
  - scripts/release/verify_glibc_floor.sh
  - tests/smoke/ffi_export_surface.c
  - tests/smoke/go_bootstrap_smoke.go
findings:
  critical: 0
  warning: 4
  info: 1
  total: 5
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-04-21T08:07:32Z
**Depth:** standard
**Files Reviewed:** 27
**Status:** issues_found

## Summary

Reviewed the Phase 6 release-prep and release automation end to end: custom GitHub actions, workflow orchestration, release-state rewrite helpers, smoke gates, and the operator runbooks. The main problems are rerun safety and release-source enforcement rather than syntax or basic packaging logic.

I also ran `python3 -m unittest scripts/release/test_update_bootstrap_release_state.py` successfully and locally reproduced the artifact-collision failure mode in `assemble_staged_release_tree.sh` (`expected exactly one packaged artifact ... found 2`) by adding a prior staging tree beside the per-platform artifacts.

## Warnings

### WR-01: Wildcard Artifact Downloads Break Staging Job Reruns

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:445-450`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:611-618`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:480-485`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:672-680`, `/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/assemble_staged_release_tree.sh:143-151`
**Issue:** The staging jobs download artifacts with `pattern: release-prepare-*` and `pattern: release-*`, while the aggregate artifacts are named `release-prepare-staging` and `release-staging`. On a rerun after those aggregate artifacts already exist in the same workflow run, the wildcard also pulls the previously assembled staging bundle back into the input directory. `assemble_staged_release_tree.sh` then finds duplicate matches for a single `r2_key` and aborts. That makes failed staging jobs non-rerunnable even when the per-platform artifacts are valid.
**Fix:**
```yaml
# Use a prefix reserved for per-platform bundles only.
- uses: actions/upload-artifact@...
  with:
    name: release-platform-${{ matrix.platform_id }}

- uses: actions/download-artifact@...
  with:
    pattern: release-platform-*
```

### WR-02: Partial R2 Success Leaves `release.yml` Unrecoverable on Rerun

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:682-693`, `/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/publish_r2.sh:69-82`
**Issue:** The workflow publishes the immutable R2 tree before creating the GitHub Release. If the R2 upload succeeds and `softprops/action-gh-release` then fails, rerunning the job hits `publish_r2.sh`'s "Refusing to overwrite immutable release prefix" guard and stops before GitHub assets can be retried. That leaves the release half-published with no CI-only recovery path, which conflicts with the documented "CI is the only publish path" contract.
**Fix:** Make the R2 publish step idempotent when the existing prefix exactly matches the staged manifest, or split GitHub Release publication into a separate rerunnable job that can skip R2 once the immutable payload is already present.

### WR-03: `release-prepare` Does Not Enforce the Documented `main` Source

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:3-9`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:48-52`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:589-604`, `/Users/tazarov/experiments/amikos/pure-simdjson/docs/releases.md:45-72`
**Issue:** The runbook says the only supported source is `release-prep/v<version> -> main -> tag on merged main commit`, but `release-prepare.yml` is a plain `workflow_dispatch` with no guard that the selected ref is `main`. It will happily build and force-push `release-prep/v<version>` from any feature branch or stale ref chosen in the workflow UI. That undermines the release-source contract and makes it easy to stage release metadata from the wrong commit line.
**Fix:**
```bash
git fetch --no-tags origin main:refs/remotes/origin/main
test "$GITHUB_REF" = "refs/heads/main"
test "$GITHUB_SHA" = "$(git rev-parse origin/main)"
```
Run that as an early fail-fast step in `release-prepare.yml` before any build or branch push.

### WR-04: Repeated "Semver" Regex Is Incorrect

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:64-67`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:183-186`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:308-311`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:438-441`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:40-43`, `/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/check_readiness.sh:49-51`, `/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/publish_r2.sh:52-54`
**Issue:** The shared regex `^[0-9]+(\.[0-9]+){2}([-.][0-9A-Za-z.-]+)?$` is described as semver validation, but it rejects valid SemVer 2 build metadata (`1.2.3+build.4`, `v1.2.3+build.4`) and accepts invalid dotted suffixes (`1.2.3.rc1`, `v1.2.3.rc1`). That creates inconsistent acceptance rules across readiness, prep, publish, and tag verification.
**Fix:** Centralize version validation in one helper and use a SemVer-correct parser or pattern, for example:
```python
r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?"
```

## Info

### IN-01: `build-shared-library` Exposes a `toolchain-file` Input but Ignores It

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/actions/build-shared-library/action.yml:5-8`, `/Users/tazarov/experiments/amikos/pure-simdjson/.github/actions/build-shared-library/action.yml:45-49`
**Issue:** The action advertises a configurable `toolchain-file` input, but the implementation always forwards `rust-toolchain.toml` to `setup-rust`. That makes the action contract misleading and prevents callers from overriding the file path even though the interface says they can.
**Fix:** Pass `${{ inputs.toolchain-file }}` into `./.github/actions/setup-rust` instead of the hardcoded filename.

---

_Reviewed: 2026-04-21T08:07:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
