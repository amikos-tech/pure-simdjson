---
phase: 06-ci-release-matrix-platform-coverage
verified: 2026-04-21T08:11:45Z
status: human_needed
score: 7/8 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Download a published macOS dylib and clear quarantine"
    expected: "After `xattr -d com.apple.quarantine <path-to-dylib>`, the downloaded dylib loads successfully on a fresh macOS host"
    why_human: "Requires a real downloaded release artifact plus Gatekeeper behavior; the repo's own 06-VALIDATION.md marks this manual-only"
  - test: "Review the generated GitHub release notes for a real tag"
    expected: "The published notes are acceptable for a public release and align with the prepared CHANGELOG entry"
    why_human: "The workflow can generate notes, but wording/communication quality is a human judgment"
deferred:
  - truth: "Fresh-machine live-artifact bootstrap against public R2 and GitHub Releases"
    addressed_in: "Phase 06.1"
    evidence: "ROADMAP Phase 06.1 goal explicitly promotes fresh-machine public validation after Phase 6 publish"
---

# Phase 6: CI Release Matrix + Platform Coverage Verification Report

**Phase Goal:** A tag push produces signed, verified shared libraries for all five targets (plus an Alpine smoke-test signal) uploaded to R2 and GitHub Releases with a generated checksum manifest. CI is the only path to a release.
**Verified:** 2026-04-21T08:11:45Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | A tag push rebuilds five artifacts, generates `SHA256SUMS`, cosign-signs them, verifies the signatures, uploads to immutable R2, and publishes GitHub Release assets | ✓ VERIFIED | [`release.yml`](../../../.github/workflows/release.yml) stages `combined-manifest.json`, generates `SHA256SUMS`, runs `cosign sign-blob` and `cosign verify-blob`, calls `publish_r2.sh`, and then `softprops/action-gh-release` ([release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:545), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:568), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:589), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:613), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:682), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:689)). |
| 2 | The release cannot publish until all five target artifacts pass the native CI-04 smoke gate | ✓ VERIFIED | Each linux/darwin/windows build job runs `run_native_smoke.sh` before uploading its staged artifact bundle, and the publish-staging job depends on all build jobs plus Alpine ([release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:158), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:284), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:414), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:446)). `run_native_smoke.sh` performs audit -> `ffi_export_surface.c` -> `minimal_parse.c` in order ([run_native_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_native_smoke.sh:41), [run_native_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_native_smoke.sh:44), [run_native_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_native_smoke.sh:80), [run_native_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_native_smoke.sh:85)). |
| 3 | Linux GNU artifacts are built from pinned manylinux images, arm64 is blocked without a 4K page-size proof, and linux publish is blocked on the glibc floor | ✓ VERIFIED | Both workflows use pinned manylinux2014 image digests and `ubuntu-24.04-arm`; the arm64 job records `linux-arm64-pagesize.txt` before packaging; `verify_glibc_floor.sh` blocks on `GLIBC_2.17` and header-derived exports ([release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:35), [release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:75), [build_linux_manylinux.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/build_linux_manylinux.sh:51), [build_linux_manylinux.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/build_linux_manylinux.sh:145), [verify_glibc_floor.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/verify_glibc_floor.sh:41), [verify_glibc_floor.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/verify_glibc_floor.sh:76)). |
| 4 | Darwin and Windows artifacts follow the platform contract: ad-hoc-signed thin dylibs on native macOS runners and MSVC-named DLLs with long-path handling on Windows | ✓ VERIFIED | Darwin jobs build on `macos-15-intel` and `macos-15`, run `codesign -s - --force --timestamp=none`, verify codesign, and reject fat/universal output; Windows enables `core.longpaths`, uses `ilammy/msvc-dev-cmd`, audits exports/dependents, and asserts `pure_simdjson-msvc.dll` / `pure_simdjson-windows-amd64-msvc.dll` naming ([release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:180), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:238), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:244), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:298), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:319), [release.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release.yml:379)). |
| 5 | The Go packaged-artifact smoke exercises the real bootstrap mirror path, while Alpine stays limited to the documented `PURE_SIMDJSON_LIB_PATH` escape hatch | ✓ VERIFIED | `run_go_packaged_smoke.sh` explicitly unsets `PURE_SIMDJSON_LIB_PATH`, serves the staged tree over loopback, sets `PURE_SIMDJSON_BINARY_MIRROR`, sets `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1`, and runs `go_bootstrap_smoke.go`; `run_alpine_smoke.sh` enforces one pinned `alpine:latest@sha256:...` ref and uses only `PURE_SIMDJSON_LIB_PATH` ([run_go_packaged_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_go_packaged_smoke.sh:52), [run_go_packaged_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_go_packaged_smoke.sh:100), [run_go_packaged_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_go_packaged_smoke.sh:106), [run_alpine_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_alpine_smoke.sh:4), [run_alpine_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_alpine_smoke.sh:38), [run_alpine_smoke.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/run_alpine_smoke.sh:61)). |
| 6 | Release preparation rewrites `version.go`, `checksums.go`, and `CHANGELOG.md` on a normal `release-prep/v<version>` branch, and the strict readiness gate rejects unprepared source state before tagging | ✓ VERIFIED | `release-prepare.yml` combines manifest rows, runs `update_bootstrap_release_state.py`, updates `CHANGELOG.md`, validates prepared source state, commits to `release-prep/v<version>`, and prints merge-then-tag instructions; `check_readiness.sh --strict` shells out to `assert_prepared_state.py --check-source` and checks `origin/main` ancestry ([release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:452), [release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:510), [release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:517), [release-prepare.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/release-prepare.yml:589), [check_readiness.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/check_readiness.sh:44), [check_readiness.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/release/check_readiness.sh:71)). |
| 7 | The runbook, bootstrap docs, and repo-local skill all point to the same prep -> main -> tag CI-only release path and document the macOS quarantine workaround | ✓ VERIFIED | `docs/releases.md` declares the CI-only publish path, documents `release-prep/v<version> -> main -> tag`, and includes `xattr -d com.apple.quarantine`; `docs/bootstrap.md` mirrors the same workaround and points to the release runbook; the repo-local skill requires both the runbook and strict readiness gate ([docs/releases.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/releases.md:3), [docs/releases.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/releases.md:45), [docs/releases.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/releases.md:179), [docs/bootstrap.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/bootstrap.md:144), [.agents/skills/pure-simdjson-release/SKILL.md](/Users/tazarov/experiments/amikos/pure-simdjson/.agents/skills/pure-simdjson-release/SKILL.md:1)). |
| 8 | A downloaded macOS release dylib opens after the documented `xattr -d com.apple.quarantine` workaround | ? UNCERTAIN | The codebase and docs prepare for this (`codesign`, runbook, bootstrap docs), but the repo's own validation file marks it manual-only because it depends on a real downloaded artifact and Gatekeeper behavior ([docs/releases.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/releases.md:173), [docs/bootstrap.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/bootstrap.md:144), [06-VALIDATION.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/06-ci-release-matrix-platform-coverage/06-VALIDATION.md:92)). |

**Score:** 7/8 truths verified

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases.

| # | Item | Addressed In | Evidence |
| --- | --- | --- | --- |
| 1 | Fresh-machine live-artifact bootstrap against public R2 / GitHub Releases | Phase 06.1 | Phase `06.1` in `ROADMAP.md` exists specifically to validate the already-published artifacts on empty-cache fresh machines. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `.github/workflows/release-prepare.yml` | Pre-tag prep/build/smoke/source-rewrite workflow | ✓ VERIFIED | 623 lines; contains five-target prep matrix, bootstrap-state rewrite, packaged-artifact smoke, branch push, and handoff summary. |
| `.github/workflows/release.yml` | Tag-publish build/sign/publish workflow | ✓ VERIFIED | 693 lines; contains main-anchored tag guard, five-target rebuilds, staged-manifest assertion, signing, R2 upload, and GitHub Release publish. |
| `.github/actions/setup-rust/action.yml` | Repo-pinned Rust installation from `rust-toolchain.toml` | ✓ VERIFIED | Resolves `[toolchain].channel` from the file, installs the channel, and installs requested targets. |
| `.github/actions/build-shared-library/action.yml` | Deterministic shared-library build action | ✓ VERIFIED | Applies `SOURCE_DATE_EPOCH`, deterministic `RUSTFLAGS`, manylinux handoff, and optional reproducibility double-build hashing. |
| `scripts/release/package_shared_artifact.sh` | Single source of truth for R2 key, GitHub asset name, and per-entry SHA-256 manifest | ✓ VERIFIED | Copies staged bytes into both naming schemes and hashes the exact packaged R2 bytes before writing the manifest entry. |
| `scripts/release/update_bootstrap_release_state.py` | Manifest-driven rewrite of `internal/bootstrap/version.go` and `checksums.go` | ✓ VERIFIED | Enforces exact manifest shape, tuple order, checksum-key naming, and 64-char lowercase digests before rewriting. |
| `scripts/release/verify_glibc_floor.sh` | Linux GLIBC/export-surface gate | ✓ VERIFIED | Uses `objdump -T`, `nm -D`, and header-derived symbol expectations; prints highest observed GLIBC version. |
| `tests/smoke/ffi_export_surface.c` | CI-04 ABI-load harness that resolves and invokes every public symbol once | ✓ VERIFIED | 711 lines; resolves 24 public exports and requires every one to be called before passing. |
| `tests/smoke/go_bootstrap_smoke.go` | Minimal Go consumer smoke program | ✓ VERIFIED | Calls `purejson.NewParser()`, parses `42`, and asserts the result. |
| `scripts/release/run_go_packaged_smoke.sh` | Loopback bootstrap smoke for staged artifacts | ✓ VERIFIED | Serves staged artifacts over local HTTP, forces mirror-only bootstrap, and runs the Go smoke binary. |
| `scripts/release/publish_r2.sh` | Immutable-prefix R2 uploader | ✓ VERIFIED | Checks the destination prefix with `aws s3api list-objects-v2` before recursive upload. |
| `docs/releases.md` | Human release runbook | ✓ VERIFIED | Documents prep branch -> main -> tag sequencing, required repo config, artifact layout, cosign verification, and the Phase 06.1 boundary. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `.github/workflows/release-prepare.yml` | `.github/actions/build-shared-library/action.yml` | Shared deterministic build action with `verify_reproducible: true` | WIRED | Linux, darwin, and windows prep jobs all use the shared action before packaging. |
| `scripts/release/package_shared_artifact.sh` | `internal/bootstrap/url.go` | Shared R2/cache naming contract | WIRED | Both use `libpure_simdjson.so` / `.dylib` / `pure_simdjson-msvc.dll` and the flat GitHub asset naming split. |
| `.github/workflows/release-prepare.yml` | `scripts/release/update_bootstrap_release_state.py` | Combined manifest rewrites bootstrap source state | WIRED | Prep staging builds `combined-manifest.json` and immediately rewrites `version.go` / `checksums.go` from it. |
| `.github/workflows/release-prepare.yml` | `scripts/release/run_go_packaged_smoke.sh` | Loopback staged-artifact bootstrap smoke | WIRED | The smoke runs after prepared state exists and after the staged release tree is assembled. |
| `.github/workflows/release.yml` | `scripts/release/assert_prepared_state.py` | Rebuilt manifest must match committed prepared state | WIRED | Publish staging blocks on `assert_prepared_state.py --manifest ... --version ...`. |
| `.github/workflows/release.yml` | `scripts/release/publish_r2.sh` | Immutable R2 publication | WIRED | R2 publish happens only after smoke, manifest assertion, `SHA256SUMS`, and cosign sign/verify steps. |
| `.github/workflows/release.yml` | `softprops/action-gh-release` | Flat GitHub Release asset publication | WIRED | GitHub Release assets are prepared from the already-signed staged raw blobs and published with generated notes. |
| `scripts/release/check_readiness.sh` | `scripts/release/assert_prepared_state.py` | Strict readiness gate | WIRED | Readiness reuses the same source-state contract instead of reimplementing it in shell. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `scripts/release/package_shared_artifact.sh` | `sha256` / manifest entry fields | Exact bytes copied to the staged R2/cache path | Yes | ✓ FLOWING |
| `.github/workflows/release-prepare.yml` | `combined-manifest.json` | Five uploaded per-platform manifest rows | Yes | ✓ FLOWING |
| `scripts/release/update_bootstrap_release_state.py` | `Version` and `Checksums` rewrites | `combined-manifest.json` entries keyed by `r2_key` and `sha256` | Yes | ✓ FLOWING |
| `.github/workflows/release.yml` | `SHA256SUMS`, `.sig`, `.pem`, flat GitHub assets | Rebuilt combined manifest plus staged raw artifacts | Yes | ✓ FLOWING |
| `scripts/release/run_go_packaged_smoke.sh` | `PURE_SIMDJSON_BINARY_MIRROR` / `PURE_SIMDJSON_DISABLE_GH_FALLBACK` | Local HTTP server over the staged release tree | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Release helper scripts parse cleanly | `bash -n scripts/release/*.sh` subset covering package/build/verify/assemble/smoke/publish/readiness | Exit 0 | ✓ PASS |
| Manifest/state rewrite contract is unit-tested | `python3 -m unittest scripts/release/test_update_bootstrap_release_state.py` | `Ran 4 tests ... OK` | ✓ PASS |
| Native CI-04 smoke gate works on a real staged library on the current host | `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` | `ffi export surface smoke passed` and `phase2 smoke passed` | ✓ PASS |
| Minimal Go consumer smoke runs against a real local dylib | `PURE_SIMDJSON_LIB_PATH=... go run ./tests/smoke/go_bootstrap_smoke.go` | Exit 0 | ✓ PASS |
| Packaging -> staged-tree assembly -> bootstrap-state rewrite roundtrip works together | Synthetic temp-workspace roundtrip using `package_shared_artifact.sh`, `assemble_staged_release_tree.sh`, copied `update_bootstrap_release_state.py`, and copied `assert_prepared_state.py` | Rewritten temp source passed `--check-source` and emitted all five expected staged paths | ✓ PASS |
| Strict readiness gate rejects unprepared source state before tagging | `python3 scripts/release/assert_prepared_state.py --check-source --version 0.1.0` | Fails with the five missing checksum keys | ✓ PASS (negative test) |
| Basic readiness gate recognizes the repo-level workflow/runbook prerequisites | `bash scripts/release/check_readiness.sh` | `basic release readiness checks passed` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `PLAT-01` | `06-02-PLAN.md` | linux/amd64 glibc ≥ 2.17 baseline | ✓ SATISFIED | Pinned manylinux2014 x86_64 image plus `verify_glibc_floor.sh` gate in both workflows. |
| `PLAT-02` | `06-02-PLAN.md` | linux/arm64 glibc ≥ 2.17 with 4K-page runner proof | ✓ SATISFIED | Pinned manylinux2014 aarch64 image, explicit `PAGE_SIZE=4096` proof, and glibc-floor gate. |
| `PLAT-03` | `06-03-PLAN.md` | darwin/amd64 ad-hoc-signed dylib | ✓ SATISFIED | `macos-15-intel` job runs ad-hoc `codesign`, verifies codesign, and rejects fat output. |
| `PLAT-04` | `06-03-PLAN.md` | darwin/arm64 ad-hoc-signed dylib | ✓ SATISFIED | `macos-15` job runs the same thin/codesign gates for arm64. |
| `PLAT-05` | `06-03-PLAN.md` | windows/amd64 MSVC DLL named `pure_simdjson-msvc.dll` | ✓ SATISFIED | Windows job uses MSVC, enables long paths, packages the DLL under the bootstrap/R2 and GitHub names, and records dependents. |
| `PLAT-06` | `06-04-PLAN.md` | Alpine smoke via `PURE_SIMDJSON_LIB_PATH` using a user-built `.so` | ✓ SATISFIED | Dedicated `alpine-smoke` job runs `run_alpine_smoke.sh` with one pinned image ref and only the escape-hatch env var. |
| `CI-01` | `06-01/02/05` | Tag-push release workflow builds all five targets | ✓ SATISFIED | `release.yml` has linux, darwin, and windows build matrices for the five supported targets. |
| `CI-02` | `06-03-PLAN.md` | macOS jobs ad-hoc codesign the dylib | ✓ SATISFIED | Both darwin jobs run `codesign -s - --force --timestamp=none` and `codesign --verify`. |
| `CI-03` | `06-02-PLAN.md` | Linux jobs use manylinux2014/equivalent baseline | ✓ SATISFIED | Both linux jobs pin manylinux2014 image digests and route builds through `build_linux_manylinux.sh`. |
| `CI-04` | `06-04-PLAN.md` | Per-platform FFI smoke verifies export load + parse round-trip | ✓ SATISFIED | `run_native_smoke.sh` and `ffi_export_surface.c` are wired before artifact upload on all five targets; local `darwin-arm64` spot check passed. |
| `CI-05` | `06-01/05` | Release pipeline computes SHA-256 and updates bootstrap checksum state | ✓ SATISFIED | Prep rewrites `version.go` / `checksums.go` from manifest entries; publish recomputes manifest, asserts coherence, generates `SHA256SUMS`, and signs/uploads. |
| `CI-06` | `06-05/06` | Version bump, changelog, and release notes live in the release path | ✓ SATISFIED | Prep updates `CHANGELOG.md` and prepared source state; publish uses `generate_release_notes: true`; runbook/readiness document the same flow. |
| `CI-07` | `06-04-PLAN.md` | Alpine smoke job runs on every release | ✓ SATISFIED | `alpine-smoke` is a first-class job in both prep and publish workflows, and publish staging depends on it. |

No orphaned Phase 6 requirement IDs were found: the union of plan-frontmatter IDs matches the Phase 6 requirement set in `.planning/REQUIREMENTS.md`.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `.github/actions/verify-shared-artifact/action.yml` | 1 | Orphaned composite action (no non-doc references in the actual release path) | ℹ️ Info | The real workflows call `run_native_smoke.sh` directly, so future edits to this action will not affect the release path unless it is wired in later. |

### Human Verification Required

### 1. Downloaded macOS Dylib

**Test:** Publish a real tag, download the released macOS `.dylib`, run `xattr -d com.apple.quarantine <path-to-dylib>`, and load it on a fresh macOS host.
**Expected:** After removing the quarantine attribute, the downloaded dylib loads successfully.
**Why human:** Requires a real downloaded artifact and Gatekeeper behavior. The repo marks this as manual-only in `06-VALIDATION.md`.

### 2. Public Release Notes

**Test:** Inspect the generated GitHub Release draft/notes for a real published tag.
**Expected:** The notes are accurate, publicly acceptable, and aligned with the prepared `CHANGELOG.md` entry.
**Why human:** Release-note wording and public communication quality cannot be validated programmatically.

### Gaps Summary

No code gaps were found in the Phase 6 implementation. The release path is substantively present: prep rewrites source state, tag publish re-validates it, five-target smoke gates are wired before publish, Alpine validation is separate and explicit, signatures and checksum material are generated in CI, and publication fans out to both R2 and GitHub Releases. Remaining work is manual verification of real published-artifact behavior and release-note quality, which keeps the phase at `human_needed` rather than `passed`.

---

_Verified: 2026-04-21T08:11:45Z_  
_Verifier: Claude (gsd-verifier)_
