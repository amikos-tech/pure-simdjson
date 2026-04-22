---
phase: 6
slug: ci-release-matrix-platform-coverage
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-20
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from `06-RESEARCH.md` §Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | mixed shell scripts, Go `testing`, Rust `cargo test`, and GitHub Actions workflow jobs |
| **Config file** | none — workflow YAML + repo scripts |
| **Quick run command** | `cargo test --release && go test ./... -count=1 -timeout 120s` |
| **Full suite command** | `cargo test --release && go test ./... -race -count=1 -timeout 180s && bash scripts/release/check_readiness.sh --strict` |
| **Estimated runtime** | ~60–180 seconds locally; CI matrix longer |

---

## Sampling Rate

- **After every task commit:** Run `cargo test --release && go test ./... -count=1 -timeout 120s`
- **After every plan wave:** Run `cargo test --release && go test ./... -race -count=1 -timeout 180s`
- **Before `$gsd-verify-work`:** Full suite plus readiness gate must be green
- **Max feedback latency:** 180 seconds locally; workflow syntax / grep checks on every workflow edit

---

## Per-Requirement Verification Map

| Req | Behavior | Test Type | Automated Command | File Exists | Status |
|-----|----------|-----------|-------------------|-------------|--------|
| PLAT-01 | linux/amd64 built under manylinux with glibc `<= 2.17` proof | workflow + shell | `rg 'manylinux2014_x86_64|objdump -T|GLIBC_2\\.17' .github/workflows/release*.yml scripts/release/verify_glibc_floor.sh` | ❌ Wave 0 | ⬜ pending |
| PLAT-02 | linux/arm64 built under manylinux with glibc `<= 2.17` proof on a 4K-page runner | workflow + shell | `rg 'manylinux2014_aarch64@sha256:|ubuntu-24\\.04-arm|getconf PAGE_SIZE|pagesize|4096|objdump -T|GLIBC_2\\.17' .github/workflows/release*.yml scripts/release/verify_glibc_floor.sh` | ❌ Wave 0 | ⬜ pending |
| PLAT-03 | darwin/amd64 ad-hoc codesigned and verified | workflow | `rg 'macos-15-intel|codesign -s - --force --timestamp=none|codesign --verify' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| PLAT-04 | darwin/arm64 ad-hoc codesigned and verified | workflow | `rg 'macos-15[^-]|codesign -s - --force --timestamp=none|codesign --verify|file ' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| PLAT-05 | windows/amd64 MSVC build, long-path enable, export audit | workflow | `rg 'windows-latest|msvc-dev-cmd|core.longpaths|dumpbin /EXPORTS|pure_simdjson-windows-amd64-msvc\\.dll' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| PLAT-06 | Alpine smoke validates `PURE_SIMDJSON_LIB_PATH` escape hatch in a pinned Alpine image | workflow + shell | `rg 'ALPINE_IMAGE_REF=alpine:.*@sha256:|--image-ref|PURE_SIMDJSON_LIB_PATH|run_alpine_smoke' .github/workflows/release*.yml scripts/release/run_alpine_smoke.sh` | ❌ Wave 0 | ⬜ pending |
| CI-01 | tag push builds all five target artifacts | workflow | `rg 'push:\\n    tags:|linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-amd64' .github/workflows/release.yml` | ❌ Wave 0 | ⬜ pending |
| CI-02 | macOS jobs codesign `.dylib` | workflow | `rg 'codesign -s - --force --timestamp=none' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| CI-03 | Linux jobs use manylinux base | workflow | `rg 'manylinux2014' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| CI-04 | native ABI-load smoke plus Go packaged-artifact smoke gate release | script + Go | `rg 'ffi_export_surface.c|minimal_parse.c|go_bootstrap_smoke|PURE_SIMDJSON_BINARY_MIRROR|PURE_SIMDJSON_DISABLE_GH_FALLBACK|run_go_packaged_smoke' scripts/release tests/smoke .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| CI-05 | prep workflow rewrites `version.go` + `checksums.go`; tag workflow checks digest match | script + workflow | `python3 scripts/release/update_bootstrap_release_state.py --help && rg 'internal/bootstrap/(version|checksums)\\.go|digest match' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |
| CI-06 | version prep, `CHANGELOG.md`, release notes, and runbook-backed readiness gate | workflow + docs | `test -f docs/releases.md && test -f scripts/release/check_readiness.sh && test -f CHANGELOG.md && rg 'CHANGELOG\\.md|release-prepare|generate_release_notes|check_readiness' .github/workflows/release*.yml docs/releases.md CHANGELOG.md` | ❌ Wave 0 | ⬜ pending |
| CI-07 | Alpine smoke runs in workflow via a pinned Alpine image ref | workflow | `rg 'ALPINE_IMAGE_REF=alpine:.*@sha256:|run_alpine_smoke\\.sh --image-ref' .github/workflows/release*.yml` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Fault Injection Test Matrix

| Fault | Test Pattern | Expected Behavior | Requirement |
|-------|--------------|-------------------|-------------|
| prep-workflow checksum map does not match tag-build artifact digests | tag workflow recomputes manifest and compares to committed `checksums.go` | publish aborts before signing/upload | CI-05 |
| linux artifact accidentally links newer glibc symbols | run `scripts/release/verify_glibc_floor.sh` on the staged `.so` | linux job fails on `GLIBC_2.18+` | PLAT-01 / PLAT-02 |
| windows artifact is packaged under the wrong public filename | packaging script produces a non-`pure_simdjson-windows-amd64-msvc.dll` asset | GitHub asset naming gate fails | PLAT-05 |
| loopback mirror bootstrap path broken | serve staged files, clear cache, run Go smoke | release aborts before publish | CI-04 |
| Alpine escape hatch regressed | alpine job builds local `.so`, sets `PURE_SIMDJSON_LIB_PATH`, runs Go smoke | release aborts before publish | PLAT-06 / CI-07 |
| R2 prefix already exists for tag | immutable-prefix check queries bucket before upload | publish aborts to prevent overwrite | CI-01 / CI-05 |

---

## Wave 0 Requirements

- [ ] `.github/workflows/release-prepare.yml`
- [ ] `.github/workflows/release.yml`
- [ ] `.github/actions/setup-rust/action.yml`
- [ ] `.github/actions/build-shared-library/action.yml`
- [ ] `.github/actions/package-shared-artifact/action.yml`
- [ ] `.github/actions/verify-shared-artifact/action.yml`
- [ ] `scripts/release/update_bootstrap_release_state.py`
- [ ] `scripts/release/check_readiness.sh`
- [ ] `scripts/release/verify_glibc_floor.sh`
- [ ] `scripts/release/run_native_smoke.sh`
- [ ] `scripts/release/run_go_packaged_smoke.sh`
- [ ] `scripts/release/run_alpine_smoke.sh`
- [ ] `tests/smoke/go_bootstrap_smoke.go`
- [ ] `docs/releases.md`
- [ ] `.agents/skills/pure-simdjson-release/SKILL.md`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS artifact opens after the documented quarantine workaround | Success Criterion 5 | requires a real downloaded artifact and Gatekeeper behavior | download the published `.dylib`, run `xattr -d com.apple.quarantine <path-to-dylib>`, and load it from a fresh host |
| final fresh-machine live-artifact bootstrap against public R2 / GitHub Releases | promoted Phase `06.1` | depends on already-published artifacts and empty-cache runners | defer to Phase `06.1` |
| release notes wording is acceptable for public release | CI-06 | content quality requires human review | inspect the generated GitHub release draft or notes file before tagging |

---

## Security Domain

### Applicable ASVS L1 Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V1 Architecture | yes | pre-tag / tag-publish split plus digest-match gate |
| V4 Access Control | yes | immutable R2 prefix check and least-privilege R2 token |
| V5 Input Validation | yes | strict version / target / filename validation in release scripts |
| V6 Cryptography | yes | SHA-256 manifest in source + cosign signatures on publish |
| V14 Build / Deploy | yes | CI is the only artifact publication path |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation | Test |
|---------|--------|---------------------|------|
| artifact drift between prep and tag publish | Tampering | recompute digests on tag and compare to committed `checksums.go` | CI-05 gate |
| overwrite of an existing release prefix in R2 | Tampering | immutable-prefix check before upload | release workflow shell gate |
| unsigned or incorrectly signed macOS artifact | Tampering | `codesign` + verify before packaging | PLAT-03 / PLAT-04 |
| bootstrap path regression hidden by env override | Tampering / DoS | loopback mirror Go smoke uses real bootstrap path | CI-04 |
| Alpine regression hidden by glibc runners | DoS | explicit pinned Alpine image job with env override | PLAT-06 / CI-07 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without an automated gate
- [x] Wave 0 covers all missing workflow / script infrastructure
- [x] No watch-mode flags
- [x] Feedback latency < 180s for local quick/full verification
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-21
