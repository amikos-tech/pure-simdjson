# Phase 6: CI Release Matrix + Platform Coverage — Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 15 (planned new/modified)
**Analogs found:** 13 / 15

---

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------------|------|-----------|----------------|---------------|
| `.github/workflows/release-prepare.yml` | workflow | CI orchestration | `.github/workflows/phase3-go-wrapper-smoke.yml` + sibling `pure-tokenizers` release workflow | near-exact adapt |
| `.github/workflows/release.yml` | workflow | CI orchestration | sibling `pure-tokenizers` release workflow + `pure-onnx` runbook | near-exact adapt |
| `.github/actions/setup-rust/action.yml` | composite action | environment setup | sibling `pure-tokenizers` shared release actions | role-match |
| `.github/actions/build-shared-library/action.yml` | composite action | native build | sibling `pure-tokenizers` `build-rust-library` action | near-exact adapt |
| `.github/actions/package-shared-artifact/action.yml` | composite action | package / rename | `internal/bootstrap/url.go` naming contract + sibling packaging steps | role-match |
| `.github/actions/verify-shared-artifact/action.yml` | composite action | verification | `.github/workflows/phase2-rust-shim-smoke.yml` native smoke/export steps | near-exact adapt |
| `scripts/release/update_bootstrap_release_state.py` | transform | manifest -> source files | `internal/bootstrap/version.go` + `internal/bootstrap/checksums.go` contract | no in-repo analog |
| `scripts/release/package_shared_artifact.sh` | packaging | build output -> staged artifact | sibling workflow shell packaging steps | near-exact adapt |
| `scripts/release/verify_glibc_floor.sh` | verification | artifact -> proof | `build.rs` glibc-only policy + Phase 2 export verification pattern | role-match |
| `scripts/release/run_native_smoke.sh` | smoke | staged artifact -> C harness | `Makefile` `phase2-smoke-*` targets + `tests/smoke/minimal_parse.c` | exact adapt |
| `scripts/release/run_go_packaged_smoke.sh` | smoke | staged artifact -> bootstrap -> Go consumer | `scripts/phase3-go-wrapper-smoke.sh` + bootstrap mirror contract | role-match |
| `scripts/release/run_alpine_smoke.sh` | smoke | Alpine local build -> env override -> Go consumer | no in-repo analog; derived from Phase 6 context | no analog |
| `tests/smoke/go_bootstrap_smoke.go` | smoke binary | Go consumer | root package parse tests + bootstrap contract | role-match |
| `docs/releases.md` | docs | operator guidance | fetched `pure-onnx` `docs/releases.md` | docs-match |
| `.agents/skills/pure-simdjson-release/SKILL.md` | skill | agent guidance | global skill structure; no repo-local analog | no analog |

---

## Pattern Assignments

### `.github/workflows/release-prepare.yml` — CREATE

**Analogs**

- `.github/workflows/phase3-go-wrapper-smoke.yml` — current repo matrix / runner conventions
- `pure-tokenizers/.github/workflows/rust-release.yml` — build -> stage -> sign/publish split

**What to keep**

- `actions/checkout@v4`
- `dtolnay/rust-toolchain@stable`
- `actions/setup-go@v5`
- existing runner labels already used by this repo: `ubuntu-latest`, `ubuntu-24.04-arm`, `macos-15-intel`, `macos-15`, `windows-latest`

**What to adapt**

- trigger is `workflow_dispatch`, not tag push
- output is a prep commit / PR update, not publication
- workflow must upload manifest artifacts that later rewrite `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go`

### `.github/workflows/release.yml` — CREATE

**Analogs**

- fetched `pure-tokenizers` release workflow — artifact download, cosign signing, R2/GitHub publish
- `docs/bootstrap.md` — public naming contract for R2 vs GitHub assets

**What to keep**

- split between per-target build jobs and one publish job
- cosign signing after all target builds succeed
- R2 immutability check before upload

**What to adapt**

- publish raw `.so` / `.dylib` / `.dll`, not tarballs
- verify computed digests match committed `internal/bootstrap/checksums.go` before signing/upload
- add hard native / Go / Alpine smoke gates before publish

### `.github/actions/build-shared-library/action.yml` — CREATE

**Analog**

- fetched `pure-tokenizers/.github/actions/build-rust-library/action.yml`

**Pattern**

- input: target triple, runner kind, optional manylinux container path
- output: absolute path to built shared library
- naming resolution must handle:
  - `libpure_simdjson.so`
  - `libpure_simdjson.dylib`
  - `pure_simdjson.dll`

**Repo-specific divergence**

- local cargo output on Windows remains `pure_simdjson.dll`, but downstream packaging renames it to bootstrap contract names

### `.github/actions/verify-shared-artifact/action.yml` — CREATE

**Analogs**

- `.github/workflows/phase2-rust-shim-smoke.yml`
- `Makefile` `phase2-smoke-linux`, `phase2-smoke-windows`, `phase2-verify-exports`

**Pattern**

- inputs: artifact path, goos, goarch, target triple
- linux: run `nm -D --defined-only`, `objdump -T`, then a native ABI-load harness that resolves every public symbol from `include/pure_simdjson.h` and invokes each once, then the existing `tests/smoke/minimal_parse.c` harness
- macOS: run `codesign --verify --verbose`, `file`, then the same ABI-load harness and `tests/smoke/minimal_parse.c`
- windows: run `dumpbin /EXPORTS`, then the same ABI-load harness via `LoadLibraryW` / `GetProcAddress`, then `tests/smoke/minimal_parse.c`

### `scripts/release/update_bootstrap_release_state.py` — CREATE

**Analog**

- no direct code analog; contract comes from `internal/bootstrap/version.go` and `internal/bootstrap/checksums.go`

**Pattern**

- input manifest shape:
  - `version`
  - `entries[]` with `goos`, `goarch`, `rust_target`, `r2_key`, `github_asset_name`, `local_path`, `sha256`
- rewrite only:
  - `internal/bootstrap/version.go`
  - `internal/bootstrap/checksums.go`
- consume `entries[].sha256` directly; do not re-hash `local_path` during rewrite
- preserve comment blocks and deterministic ordering by `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`

### `scripts/release/package_shared_artifact.sh` — CREATE

**Analogs**

- fetched `pure-tokenizers` release packaging shell
- `internal/bootstrap/url.go` naming helpers

**Pattern**

- copy build output into a staging directory
- emit two filenames from one source:
  - R2 / cache filename via `PlatformLibraryName(goos)`
  - GitHub asset filename via `githubAssetName(goos, goarch)`
- compute `sha256` from the packaged R2/cache file
- produce manifest metadata consumed by both the Python rewrite step and prepared-state assertion:
  - `goos`, `goarch`, `rust_target`, `r2_key`, `github_asset_name`, `local_path`, `sha256`

### `scripts/release/run_native_smoke.sh` — CREATE

**Analogs**

- `Makefile` smoke targets
- `tests/smoke/README.md`
- `tests/smoke/minimal_parse.c`

**Pattern**

- linux/macOS: compile a dedicated ABI-load harness that resolves every public symbol from `include/pure_simdjson.h` with `dlopen` / `dlsym`, invoke each once, then compile and run `tests/smoke/minimal_parse.c`
- windows: compile the same ABI-load harness with `LoadLibraryW` / `GetProcAddress`, invoke each once, then compile `tests/smoke/minimal_parse.c` with `cl` and run it with the DLL on `PATH`

### `scripts/release/run_go_packaged_smoke.sh` — CREATE

**Analogs**

- `scripts/phase3-go-wrapper-smoke.sh`
- `docs/bootstrap.md` loopback mirror allowance from `validateBaseURL()`

**Pattern**

- stage artifacts under the exact R2 directory layout
- serve them from loopback HTTP
- clear `PURE_SIMDJSON_CACHE_DIR`
- set `PURE_SIMDJSON_BINARY_MIRROR=http://127.0.0.1:<port>/pure-simdjson`
- set `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1`
- run the dedicated Go smoke binary

### `scripts/release/run_alpine_smoke.sh` — CREATE

**Analog**

- none in repo; direct translation of Phase 6 context D-01 / D-02

**Pattern**

- use one exact image reference string everywhere: `alpine:latest@sha256:<resolved-digest>`
- install Rust, Go, C++ build prerequisites
- run `cargo build --release`
- export `PURE_SIMDJSON_LIB_PATH=<repo>/target/release/libpure_simdjson.so`
- run the Go smoke binary

### `docs/releases.md` — CREATE

**Analog**

- fetched `pure-onnx/docs/releases.md`

**Pattern**

- explain prep-then-tag model
- list required secrets / vars
- document published URLs
- include cosign verification examples for raw artifacts and `SHA256SUMS`

### `.agents/skills/pure-simdjson-release/SKILL.md` — CREATE

**Analog**

- no repo-local analog

**Pattern**

- instruct agents to read `docs/releases.md` first
- require `scripts/release/check_readiness.sh --strict` before suggesting a tag
- point Phase `06.1` work to fresh-runner validation, not manual workstation assumptions

---

## Existing Code Anchors To Reuse

| Existing File | Why It Matters | Pattern To Reuse |
|---------------|----------------|------------------|
| `internal/bootstrap/url.go` | one source of truth for on-disk and GitHub asset names | no release script should invent names independently |
| `internal/bootstrap/version.go` | release-prep must update compile-time version | keep file shape stable |
| `internal/bootstrap/checksums.go` | release-prep must write real SHA-256 values | keep map key format stable |
| `cmd/pure-simdjson-bootstrap/verify.go` | packaged-artifact verification contract | smoke flow should stay aligned with checksum key format |
| `tests/smoke/minimal_parse.c` | existing native smoke harness | reuse, do not write a second C smoke harness |
| `.github/workflows/phase2-rust-shim-smoke.yml` | current export / smoke sequencing | verify first, then package / publish |
| `.github/workflows/phase3-go-wrapper-smoke.yml` | current five-target runner selection | keep runner labels consistent |

---

## Planner Guidance

- Keep the build logic in reusable actions or scripts; keep the publish policy in workflow YAML.
- Use `internal/bootstrap/url.go` as the canonical naming source in every task action and acceptance criterion.
- Treat `release-prepare.yml` and `release.yml` as one system. Any file naming, matrix, or smoke change must land in both via shared helpers.
- Do not plan a musl publish artifact. Alpine is validation only.
- Do not plan a second Go-side verifier. Use the existing bootstrap path with staged artifacts and a loopback mirror.
