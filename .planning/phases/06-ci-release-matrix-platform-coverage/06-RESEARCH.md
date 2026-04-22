# Phase 6: CI Release Matrix + Platform Coverage - Research

**Researched:** 2026-04-20
**Domain:** GitHub Actions release automation for raw shared-library artifacts, pre-tag bootstrap metadata preparation, cosign signing, R2 + GitHub Releases publishing, and hard release gating
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** `v0.1` does not ship a musl runtime artifact. Alpine remains a smoke-test-only path using `PURE_SIMDJSON_LIB_PATH`.
- **D-02:** The Alpine smoke job is a hard release gate, not an advisory signal.
- **D-03:** Release metadata is prepared in a normal source commit before tagging. The publish tag must point at that exact prepared commit.
- **D-04:** The tag must not rely on a post-tag follow-up PR to make bootstrap metadata coherent. Tagged source must already contain the release-ready `internal/bootstrap/version.go`, `internal/bootstrap/checksums.go`, and release-facing docs required by the workflow.
- **D-05:** Linux release artifacts use manylinux-style container builds as the default path.
- **D-06:** Linux release jobs must prove a glibc baseline of `<= 2.17` via `objdump -T` checks on the produced `.so`.
- **D-07:** zig/cross remains an escape hatch if the default Linux path proves insufficient, but it is not the primary strategy.
- **D-08:** Release publication is blocked on native artifact verification and Go-side bootstrap / consumer verification using the packaged artifacts before publish.
- **D-09:** Artifact verification includes exported symbols, parse smoke, signature/codesign checks, checksum generation, and packaging correctness.
- **D-10:** Phase 6 includes an in-repo release runbook that humans and agents both follow.
- **D-11:** The runbook is backed by a scriptable readiness gate so the process is documented and enforceable.
- **D-12:** Release guidance is also delivered as a repo-local agent skill backed by the same runbook.
- **D-13:** Promoted follow-up Phase `06.1` is a hard gate before the final `v0.1` release closeout work in Phase 7.
- **D-14:** For Phase `06.1`, "fresh machine" can be satisfied by a clean CI runner / empty-cache validation path rather than a manually managed workstation.

### Claude's Discretion

- Exact split between composite actions, shell scripts, and workflow YAML, as long as the release path stays auditable and deterministic.
- Exact shape of the Go-side packaged-artifact smoke gate, as long as it exercises the real bootstrap / consumer path before publish.
- Exact runbook / skill file layout, as long as both point at the same release source of truth.

### Deferred Ideas (OUT OF SCOPE)

- Shipping a musl runtime artifact in `v0.1`
- A global reusable release skill beyond this repository
- Final fresh-machine live-artifact bootstrap validation after publish (Phase `06.1`)

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLAT-01 | linux/amd64 release artifact with glibc `<= 2.17` | manylinux2014 x86_64 container + `objdump -T` floor check + `nm -D` export audit |
| PLAT-02 | linux/arm64 release artifact with glibc `<= 2.17` | manylinux2014 aarch64 container on a 4K-page arm runner + same `objdump -T` proof |
| PLAT-03 | darwin/amd64 ad-hoc-signed `.dylib` | native `macos-15-intel` job + `codesign -s - --force --timestamp=none` + `codesign --verify` |
| PLAT-04 | darwin/arm64 ad-hoc-signed `.dylib` | native `macos-15` job + same codesign proof + thin-binary `file` check |
| PLAT-05 | windows/amd64 MSVC `.dll` named `pure_simdjson-msvc.dll` | `windows-latest` + `ilammy/msvc-dev-cmd@v1` + long-path enable + `dumpbin /EXPORTS` |
| PLAT-06 | Alpine smoke-test CI job loading via `PURE_SIMDJSON_LIB_PATH` | dedicated pinned Alpine-image job that locally builds a musl `.so`, sets the env var, and runs the Go smoke program |
| CI-01 | GitHub Actions workflow builds all 5 target artifacts on tag push | tag workflow reuses a single matrix definition shared with pre-tag prep |
| CI-02 | macOS jobs ad-hoc codesign the `.dylib` | native macOS jobs call `codesign` before smoke / packaging |
| CI-03 | Linux jobs use manylinux2014 base (or equivalent) | manylinux container script becomes the default Linux builder |
| CI-04 | Per-platform FFI smoke test verifies all exported symbols load and one parse round-trips | native C smoke plus Go packaged-artifact bootstrap smoke gate before publish |
| CI-05 | Release pipeline computes SHA-256 for each artifact and commits `checksums.go` in the tagged commit path | pre-tag release-prep workflow writes `version.go` + `checksums.go`; tag workflow rebuilds and requires digest match before publish |
| CI-06 | Version bump, `CHANGELOG.md`, and release-notes in the release path | pre-tag workflow prepares versioned source plus `CHANGELOG.md`; tag workflow uses generated release notes and the checked-in runbook |
| CI-07 | Alpine smoke-test job runs in a pinned Alpine image | explicit workflow job and `scripts/release/run_alpine_smoke.sh --image-ref` |

</phase_requirements>

---

## Summary

Phase 6 should ship as a **two-workflow release model** sharing one build/publish core:

1. **`release-prepare.yml`** runs before tagging. It builds and packages all five target artifacts, runs the native and Go packaged-artifact smoke gates, computes the real SHA-256 digests, rewrites `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go`, updates any release-facing docs or notes, and leaves those changes in a normal source commit for review / merge.
2. **`release.yml`** runs on tag push (`v*`). It rebuilds the same five artifacts from the prepared commit, verifies the newly computed digests exactly match the committed `checksums.go`, reruns the hard release gates, signs each artifact plus `SHA256SUMS` with cosign keyless OIDC, and publishes the raw artifacts plus `SHA256SUMS` and their `.sig` / `.pem` sidecars to both R2 and GitHub Releases.

This is the only model that satisfies the otherwise conflicting constraints:

- CI-05 allows the checksum manifest to live in the tagged commit.
- D-03 / D-04 forbid a post-tag "fixup PR" for `checksums.go`.
- The tag workflow still remains the only publication path.

The key enabling trick for D-08 is to validate the **real bootstrap path against staged artifacts before publish**. `internal/bootstrap/url.go` already permits `http://127.0.0.1` mirror URLs, so the release workflow can:

1. stage the packaged artifacts in the exact R2 directory layout,
2. serve them over a loopback HTTP server,
3. clear `PURE_SIMDJSON_CACHE_DIR`,
4. set `PURE_SIMDJSON_BINARY_MIRROR=http://127.0.0.1:<port>/pure-simdjson`,
5. set `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1`,
6. run a small Go consumer smoke program that calls `NewParser()`, parses `42`, and exits.

That validates the same bootstrap path Phase 5 implemented instead of falling back to `PURE_SIMDJSON_LIB_PATH`, which would only prove loader compatibility and would miss checksum / URL / cache regressions.

For Alpine, the documented contract is narrower: the release workflow should not try to prove the glibc artifact works on Alpine. It should prove the **escape hatch** works by building a musl-native `.so` inside a pinned Alpine image job, setting `PURE_SIMDJSON_LIB_PATH` to that locally built file, and running the same Go smoke binary there. That matches D-01 and D-02 exactly.

---

## Recommended Release Model

### 1. Pre-tag release preparation

**Recommended files:**

- `.github/workflows/release-prepare.yml`
- `scripts/release/update_bootstrap_release_state.py`
- `scripts/release/package_shared_artifact.sh`
- `scripts/release/check_readiness.sh`

**Recommended behavior:**

- `workflow_dispatch` with required `version` input like `0.1.0`
- build matrix covers:
  - `linux/amd64` via `quay.io/pypa/manylinux2014_x86_64`
  - `linux/arm64` via `quay.io/pypa/manylinux2014_aarch64`
  - `darwin/amd64` via `macos-15-intel`
  - `darwin/arm64` via `macos-15`
  - `windows/amd64` via `windows-latest`
- package raw library files, not tarballs, because bootstrap expects flat binary URLs
- compute a manifest that includes:
  - `version`
  - `goos`
  - `goarch`
  - `rust_target`
  - `r2_key`
  - `github_asset_name`
  - `sha256`
- rewrite `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go` from that manifest
- emit a release notes stub consumed by the tag workflow

### 2. Tag-push publish workflow

**Recommended files:**

- `.github/workflows/release.yml`
- `scripts/release/verify_glibc_floor.sh`
- `scripts/release/run_native_smoke.sh`
- `scripts/release/run_go_packaged_smoke.sh`
- `scripts/release/run_alpine_smoke.sh`

**Required gates before upload:**

- linux glibc floor proof from `objdump -T`
- `nm -D --defined-only` / `dumpbin /EXPORTS` / `codesign --verify` checks
- native C smoke using `tests/smoke/minimal_parse.c`
- Go packaged-artifact smoke via loopback mirror using the actual staged artifacts
- Alpine smoke via `PURE_SIMDJSON_LIB_PATH` and a user-built Alpine `.so`
- checksum digest match against committed `internal/bootstrap/checksums.go`

Only after all of those pass should the workflow:

- sign each raw artifact
- sign `SHA256SUMS`
- upload raw artifacts + `SHA256SUMS` + signatures + certificates to:
  - `s3://releases/pure-simdjson/v<version>/<os>-<arch>/...`
  - GitHub Release assets using `githubAssetName(goos, goarch)` plus `SHA256SUMS` and each blob's `.sig` / `.pem`
- create or update the GitHub Release with generated notes

### 3. Shared workflow building blocks

Use local composite actions for repeated logic. The patterns in `pure-tokenizers` show that the shared action boundary is worth it once the same build/package logic appears in both prep and publish workflows.

Recommended action set:

- `.github/actions/setup-rust/action.yml`
- `.github/actions/build-shared-library/action.yml`
- `.github/actions/package-shared-artifact/action.yml`
- `.github/actions/verify-shared-artifact/action.yml`

---

## Architecture Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Linux manylinux build | composite action + shell script | workflow matrix | lets prep and publish reuse the same builder |
| macOS / Windows native build | workflow job | composite action | native runners stay explicit in YAML; packaging stays shared |
| Artifact naming | `internal/bootstrap/url.go` + packaging script | workflow env | avoids a third naming scheme |
| Checksum manifest rewrite | `scripts/release/update_bootstrap_release_state.py` | pre-tag workflow | deterministic edits to `version.go` and `checksums.go` |
| Native smoke | `tests/smoke/minimal_parse.c` + shell script | workflow jobs | current repo already has the harness |
| Go packaged-artifact smoke | dedicated Go smoke binary + loopback HTTP mirror | workflow jobs | exercises bootstrap path honestly |
| Alpine escape-hatch validation | shell script in Alpine container | workflow job | matches documented contract without widening release scope |
| Signing | tag workflow | cosign CLI | publish-only concern |
| Runbook + readiness gate | `docs/releases.md` + `scripts/release/check_readiness.sh` | repo-local skill | one source of truth for humans and agents |

---

## Standard Stack

### Core

| Component | Version / Shape | Purpose | Why Standard |
|-----------|-----------------|---------|--------------|
| GitHub Actions workflows | repo-local YAML | prep + publish orchestration | current repo already uses Actions for smoke matrices |
| `actions/checkout` | `@v4` | source checkout | already used in current repo workflows |
| `actions/setup-go` | `@v5` | Go toolchain for wrapper / smoke tests | already used in current repo workflows |
| `dtolnay/rust-toolchain` | `@stable` | Rust toolchain | already used in current repo workflows |
| `ilammy/msvc-dev-cmd` | `@v1` | Windows MSVC environment | already used in current repo workflows |
| `sigstore/cosign-installer` | `@v3` | cosign CLI install | matches sibling release workflow pattern |
| `softprops/action-gh-release` | `@v2` | GitHub Release asset publish | matches sibling release workflow pattern |
| manylinux2014 containers | `x86_64` / `aarch64` | glibc floor guarantee | directly satisfies D-05 / D-06 |
| `aws` CLI | v2 | R2 upload | sibling projects already use S3-compatible uploads |

### Supporting

| Component | Purpose | When to Use |
|-----------|---------|-------------|
| `objdump -T` | glibc symbol-floor proof | linux publish gating |
| `nm -D --defined-only` | exported-symbol audit | linux artifact verification |
| `dumpbin /EXPORTS` | exported-symbol audit | windows artifact verification |
| `codesign --verify --verbose` | ad-hoc codesign verification | macOS artifact verification |
| `python3 -m http.server` or a tiny Python server | loopback artifact mirror | Go packaged-artifact smoke gate |
| `python3` rewrite script | deterministic edits to `version.go` / `checksums.go` | pre-tag prep |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| manylinux containers | zig / cross as primary | conflicts with D-05; keep only as escape hatch |
| single tag workflow that also rewrites source | post-tag follow-up PR | violates D-03 / D-04 |
| `PURE_SIMDJSON_LIB_PATH` for all smoke | loopback mirror bootstrap smoke | easier, but fails D-08 because it skips checksum + URL + cache behavior |
| shipping musl artifact | Alpine smoke-only | violates D-01 |

---

## Open Questions (RESOLVED)

### 1. How can CI-05 and D-04 both be true?

**Answer:** the checksum-producing workflow must happen **before tagging**, and the tag workflow must require that its freshly built artifact digests exactly match the committed `checksums.go`. The repo ships the "tagged commit" branch of CI-05, not the "follow-up PR" branch.

### 2. How do we verify the bootstrap path before publish without public R2?

**Answer:** serve the staged artifact directory over loopback HTTP and point `PURE_SIMDJSON_BINARY_MIRROR` at it. `validateBaseURL()` already permits loopback HTTP, so no code change is required for the mirror scheme.

### 3. How should Windows naming work when cargo emits `pure_simdjson.dll` locally but bootstrap expects `pure_simdjson-msvc.dll` in release storage?

**Answer:** keep cargo output unchanged, then rename during packaging:

- local build output: `target/<triple>/release/pure_simdjson.dll`
- R2 / cache filename: `pure_simdjson-msvc.dll`
- GitHub asset name: `pure_simdjson-windows-amd64-msvc.dll`

This matches the existing `internal/bootstrap/url.go` contract and avoids touching local dev / test conventions.

### 4. How should Alpine be validated without widening the public matrix?

**Answer:** the Alpine job builds a local musl `.so`, exports `PURE_SIMDJSON_LIB_PATH=<that file>`, and runs the Go smoke program. No artifact from that job is uploaded.

---

## Environment Availability

| Dependency | Required By | Available in repo / referenced pattern | Fallback |
|------------|------------|-----------------------------------------|----------|
| Go toolchain | Go smoke / wrapper verification | yes — current workflows already use `actions/setup-go@v5` | — |
| Rust toolchain | native library build | yes — current workflows already use `dtolnay/rust-toolchain@stable` | — |
| MSVC developer shell | windows build and export check | yes — current workflows already use `ilammy/msvc-dev-cmd@v1` | — |
| manylinux images | linux glibc floor guarantee | pattern selected in context; not yet wired in repo | zig / cross only if container path blocks |
| cosign | tag publish signing | sibling release workflows use `sigstore/cosign-installer@v3` | none within scope |
| Python 3 | state rewrite + tiny loopback server | already assumed by repo (`python3` used in Makefile tests) | — |

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | mixed shell + Go `testing` + Rust `cargo test` + GitHub Actions matrix |
| Config file | none — workflow YAML + repo scripts |
| Quick run command | `cargo test --release && go test ./... -count=1 -timeout 120s` |
| Full suite command | `cargo test --release && go test ./... -race -count=1 -timeout 180s && bash scripts/release/check_readiness.sh --strict` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLAT-01 | linux/amd64 built in manylinux with glibc `<= 2.17` | workflow + shell | `rg 'manylinux2014_x86_64|objdump -T|GLIBC_2\\.17' .github/workflows/release*.yml scripts/release/verify_glibc_floor.sh` | ❌ Wave 0 |
| PLAT-02 | linux/arm64 built in manylinux with glibc `<= 2.17` on a 4K-page runner | workflow + shell | `rg 'manylinux2014_aarch64@sha256:|ubuntu-24\\.04-arm|getconf PAGE_SIZE|pagesize|4096|objdump -T|GLIBC_2\\.17' .github/workflows/release*.yml scripts/release/verify_glibc_floor.sh` | ❌ Wave 0 |
| PLAT-03 | darwin/amd64 ad-hoc codesign | workflow | `rg 'macos-15-intel|codesign -s - --force --timestamp=none|codesign --verify' .github/workflows/release*.yml` | ❌ Wave 0 |
| PLAT-04 | darwin/arm64 ad-hoc codesign | workflow | `rg 'macos-15[^-]|codesign -s - --force --timestamp=none|file ' .github/workflows/release*.yml` | ❌ Wave 0 |
| PLAT-05 | windows/amd64 MSVC build, long-paths, export audit | workflow | `rg 'windows-latest|msvc-dev-cmd|core.longpaths|dumpbin /EXPORTS|pure_simdjson-windows-amd64-msvc\\.dll' .github/workflows/release*.yml` | ❌ Wave 0 |
| PLAT-06 | Alpine smoke via `PURE_SIMDJSON_LIB_PATH` in a pinned Alpine image | workflow + shell | `rg 'ALPINE_IMAGE_REF=alpine:.*@sha256:|--image-ref|PURE_SIMDJSON_LIB_PATH|run_alpine_smoke' .github/workflows/release*.yml scripts/release/run_alpine_smoke.sh` | ❌ Wave 0 |
| CI-01 | tag push builds all five target artifacts | workflow | `rg 'push:\\n    tags:|linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-amd64' .github/workflows/release.yml` | ❌ Wave 0 |
| CI-02 | macOS jobs codesign `.dylib` | workflow | `rg 'codesign -s - --force --timestamp=none' .github/workflows/release*.yml` | ❌ Wave 0 |
| CI-03 | linux jobs use manylinux base | workflow | `rg 'manylinux2014' .github/workflows/release*.yml` | ❌ Wave 0 |
| CI-04 | native and Go packaged-artifact smoke gates | shell + Go | `rg 'minimal_parse.c|go_bootstrap_smoke|PURE_SIMDJSON_BINARY_MIRROR|run_go_packaged_smoke' scripts/release tests/smoke .github/workflows/release*.yml` | ❌ Wave 0 |
| CI-05 | prep workflow rewrites `version.go` + `checksums.go`; tag workflow checks digest match | script + workflow | `python3 scripts/release/update_bootstrap_release_state.py --help && rg 'internal/bootstrap/(version|checksums)\\.go|digest match' .github/workflows/release*.yml` | ❌ Wave 0 |
| CI-06 | release path has version prep, `CHANGELOG.md`, release notes, and runbook-backed readiness gate | workflow + docs | `test -f docs/releases.md && test -f scripts/release/check_readiness.sh && test -f CHANGELOG.md && rg 'CHANGELOG\\.md|generate_release_notes|release-prepare|check_readiness' .github/workflows/release*.yml docs/releases.md CHANGELOG.md` | ❌ Wave 0 |
| CI-07 | Alpine smoke runs in workflow via a pinned Alpine image ref | workflow | `rg 'ALPINE_IMAGE_REF=alpine:.*@sha256:|run_alpine_smoke\\.sh --image-ref' .github/workflows/release*.yml` | ❌ Wave 0 |

### Fault Injection / Failure Gates

| Fault | Test Pattern | Expected Behavior |
|-------|-------------|-------------------|
| prepared commit checksum drift vs tag rebuild | tag workflow recomputes manifest and diffs against committed `checksums.go` | publish aborts before signing/upload |
| missing manylinux floor proof | helper script sees `GLIBC_2.18+` symbol | linux publish job fails |
| macOS artifact unsigned or universal | `codesign --verify` or `file` check fails | publish aborts |
| windows artifact renamed incorrectly | `rg` / packaging script mismatch | Go bootstrap mirror smoke and GH asset naming gate fail |
| staged artifacts pass native smoke but fail bootstrap path | loopback Go smoke exits non-zero | publish aborts |
| Alpine escape hatch broken | Alpine job fails to parse with env override | publish aborts |

### Sampling Rate

- **Per task commit:** `cargo test --release && go test ./... -count=1 -timeout 120s`
- **Per wave merge:** `cargo test --release && go test ./... -race -count=1 -timeout 180s`
- **Phase gate:** readiness script plus workflow YAML grep checks must pass before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `.github/workflows/release-prepare.yml`
- [ ] `.github/workflows/release.yml`
- [ ] `.github/actions/setup-rust/action.yml`
- [ ] `.github/actions/build-shared-library/action.yml`
- [ ] `.github/actions/package-shared-artifact/action.yml`
- [ ] `.github/actions/verify-shared-artifact/action.yml`
- [ ] `scripts/release/update_bootstrap_release_state.py`
- [ ] `scripts/release/check_readiness.sh`
- [ ] `scripts/release/verify_glibc_floor.sh`
- [ ] `scripts/release/run_go_packaged_smoke.sh`
- [ ] `scripts/release/run_alpine_smoke.sh`
- [ ] `tests/smoke/go_bootstrap_smoke.go`
- [ ] `docs/releases.md`
- [ ] `.agents/skills/pure-simdjson-release/SKILL.md`

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V1 Architecture | yes | hard pre-tag / tag-publish split with digest-match gate |
| V4 Access Control | yes | immutable R2 prefix check before upload |
| V5 Input Validation | yes | release scripts validate version strings, target names, and expected file names |
| V6 Cryptography | yes | SHA-256 manifest in source + cosign keyless signing on publish |
| V14 Build / Deploy | yes | CI is the only publish path; humans prepare source but do not hand-upload artifacts |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| malicious or accidental artifact drift between prep and tag publish | Tampering | tag workflow recomputes digests and requires exact match to committed `checksums.go` |
| R2 overwrite of an existing release prefix | Tampering | immutable-prefix check before upload |
| unsigned macOS artifact shipped | Tampering | codesign + codesign verify as hard gate |
| wrong DLL name breaks bootstrap fallback | Denial of service | package step renames to `pure_simdjson-msvc.dll` / `pure_simdjson-windows-amd64-msvc.dll` from one source of truth |
| bootstrap path regressions hidden by env override smoke | Tampering / quality gap | loopback mirror smoke exercises checksum + URL + cache path directly |

---

## Sources

### Primary (HIGH confidence)

- `.planning/phases/06-ci-release-matrix-platform-coverage/06-CONTEXT.md` — locked Phase 6 decisions [VERIFIED]
- `.planning/ROADMAP.md` — Phase 6 must-haves and success criteria [VERIFIED]
- `.planning/REQUIREMENTS.md` — `PLAT-01..06`, `CI-01..07` [VERIFIED]
- `.github/workflows/phase2-rust-shim-smoke.yml` — existing native smoke/export pattern [VERIFIED]
- `.github/workflows/phase3-go-wrapper-smoke.yml` — existing five-target Go runner pattern [VERIFIED]
- `tests/smoke/minimal_parse.c` — existing native smoke harness [VERIFIED]
- `internal/bootstrap/url.go`, `internal/bootstrap/checksums.go`, `internal/bootstrap/version.go`, `cmd/pure-simdjson-bootstrap/verify.go` — release naming and checksum contract [VERIFIED]
- `docs/bootstrap.md` — current operator-facing bootstrap contract [VERIFIED]
- `build.rs` — current glibc-only build contract and static libstdc++ / libgcc link args [VERIFIED]

### Secondary (HIGH confidence)

- `https://github.com/amikos-tech/pure-tokenizers/blob/main/.github/workflows/rust-release.yml` — current sibling release workflow pattern [FETCHED 2026-04-20]
- `https://github.com/amikos-tech/pure-tokenizers/blob/main/.github/actions/build-rust-library/action.yml` — composite action boundary and target handling [FETCHED 2026-04-20]
- `https://github.com/amikos-tech/pure-onnx/blob/main/docs/releases.md` — runbook + cosign + R2 documentation pattern [FETCHED 2026-04-20]

### Tertiary (MEDIUM confidence)

- current GitHub-hosted runner labels and manylinux image availability are treated as operationally stable but must still be verified during execution

---

## Metadata

**Confidence breakdown:**

- release workflow shape: HIGH
- checksum/tag coherence model: HIGH
- Linux manylinux / glibc-floor strategy: HIGH
- packaged-artifact bootstrap smoke approach: HIGH
- Alpine escape-hatch validation: HIGH

**Research date:** 2026-04-20
**Valid until:** 2026-05-20
