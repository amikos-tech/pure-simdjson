# Org Release Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align `pure-simdjson` with the tag-driven release process used by `pure-onnx` and `pure-tokenizers` while preserving this repo's matrix build and smoke coverage.

**Architecture:** Remove the prep-branch release contract from the operator path. Make the tag release workflow build, sign, verify, and publish the release in one pass, and move production checksum authority from committed source state to GitHub Release asset digests, with local/test checksum overrides retained for deterministic tests.

**Tech Stack:** GitHub Actions, Bash, Python, Go, GitHub Releases API, Cloudflare R2, cosign

---

### Task 1: Switch bootstrap checksum authority

**Files:**
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/bootstrap/download.go`
- Modify: `internal/bootstrap/url.go`
- Modify: `internal/bootstrap/checksums.go`
- Modify: `internal/bootstrap/export_test.go`
- Test: `internal/bootstrap/bootstrap_test.go`

- [x] Add release-metadata URL support and checksum resolution helpers.
- [x] Keep `Checksums` as an optional override map for tests and controlled local scenarios.
- [x] Resolve expected checksums from published `SHA256SUMS` metadata when no override is present.
- [x] Preserve `ErrNoChecksum` semantics for missing platform/version digests.
- [x] Add or update tests covering metadata resolution, override precedence, and missing metadata handling.

### Task 2: Standardize the tag release workflow

**Files:**
- Modify: `.github/workflows/release.yml`
- Modify: `scripts/release/publish_r2.sh`
- Create: `scripts/release/build_releases_index.sh`
- Modify: `scripts/release/check_readiness.sh`
- Modify: `scripts/release/run_native_smoke.sh`
- Delete: `.github/workflows/release-prepare.yml`

- [x] Remove the prep-branch source-state validation and release-prepare dependency.
- [x] Keep the build matrix, native smoke gates, packaged-artifact smoke, signing, and GitHub Release publishing.
- [x] Add org-standard R2 metadata publishing: `latest.json`, optional signed `releases.json`, and cache purges.
- [x] Accept org-standard R2 secret/variable naming with backward-compatible fallbacks where practical.
- [x] Fix the Windows `dumpbin` invocation under Git Bash/MSYS in the native smoke script.

### Task 3: Update repo documentation and verification guidance

**Files:**
- Modify: `docs/releases.md`
- Modify: `docs/bootstrap.md`
- Modify: `.agents/skills/pure-simdjson-release/SKILL.md`
- Modify: `.planning/phases/06-ci-release-matrix-platform-coverage/06-HUMAN-UAT.md`

- [x] Rewrite the runbook to the tag-driven flow used across the org.
- [x] Document that runtime checksum verification comes from published `SHA256SUMS`, not a generated source rewrite.
- [x] Update the repo-local release skill so it matches the new supported path.
- [x] Refresh the phase verification note so blocked or pending release state is described against the new operator contract.

### Task 4: Verify the migration locally

**Files:**
- Test: `internal/bootstrap/bootstrap_test.go`
- Test: `scripts/release/test_update_bootstrap_release_state.py`

- [x] Run focused Go tests for bootstrap behavior.
- [x] Run shell/YAML validation for the changed release scripts and workflows.
- [x] Remove or replace obsolete prepared-state tests if the prep-branch contract is gone.
- [ ] Summarize what still requires live GitHub Actions or a real tag to validate.
