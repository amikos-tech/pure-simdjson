# Release Runbook

`docs/releases.md` is the authoritative Phase 6 release runbook for this
repository. `release-prepare.yml`, `release.yml`, and
`scripts/release/check_readiness.sh` are the only supported publish path. Do
not hand-upload artifacts, rewrite checksum state by hand, or bypass CI for
publication.

## Required GitHub Configuration

`release.yml` publishes to R2 through `scripts/release/publish_r2.sh` and then
creates the GitHub Release. Configure the repository before attempting a
release:

- GitHub Actions secret: `AWS_ACCESS_KEY_ID`
- GitHub Actions secret: `AWS_SECRET_ACCESS_KEY`
- GitHub Actions secret, if your R2 credentials are session-based:
  `AWS_SESSION_TOKEN`
- GitHub Actions variable or secret: `R2_BUCKET`
- GitHub Actions variable or secret: `R2_ENDPOINT_URL`
- Optional GitHub Actions variable: `AWS_DEFAULT_REGION=auto`

No cosign private key is required. `release.yml` uses GitHub OIDC keyless
signing, so the workflow must keep `id-token: write` and `contents: write`.

## Pre-Tag Readiness Gate

Before a human or agent recommends pushing a tag, run:

```bash
VERSION=0.1.0
bash scripts/release/check_readiness.sh --strict --version "${VERSION}"
```

`--strict` is the release gate. It fails if:

- `python3 scripts/release/assert_prepared_state.py --check-source --version <semver-without-v>` fails
- `git fetch origin main --depth=1 && git merge-base --is-ancestor HEAD origin/main` fails
- `.github/workflows/release-prepare.yml` is missing
- `.github/workflows/release.yml` is missing
- `docs/releases.md` is missing

## Supported Release Sequence

Only one sequencing is supported:

1. Dispatch `release-prepare.yml` with `version=<semver-without-v>`.
   Example: `gh workflow run release-prepare.yml -f version=0.1.0`
2. Inspect the uploaded prep artifacts before merging anything:
   `release-prepare-staging` must contain `combined-manifest.json` and the
   staged tree under `release-prepare-staged-root/v<version>/...`.
   The per-platform `release-prepare-<platform>` artifacts must each contain
   the packaged raw library plus the single-row manifest used to build the
   combined manifest.
3. Open a PR from `release-prep/v<version>` into `main`.
4. Merge that PR into `main`.
5. Create an annotated tag `v<version>` on the merged `main` commit only, then
   push that tag.

Example:

```bash
VERSION=0.1.0
git fetch origin main --tags
git checkout origin/main
bash scripts/release/check_readiness.sh --strict --version "${VERSION}"
git tag -a "v${VERSION}" -m "v${VERSION}"
git push origin "v${VERSION}"
```

Tagging `release-prep/v<version>` directly is unsupported. The release source
of truth is `release-prep/v<version> -> main -> tag on merged main commit`.

## What `release.yml` Enforces

The tag workflow rejects off-main releases before any publish step begins:

```bash
git fetch --no-tags origin main:refs/remotes/origin/main
git merge-base --is-ancestor "$GITHUB_SHA" "origin/main"
```

It also re-runs the prepared-state contract check:

```bash
python3 scripts/release/assert_prepared_state.py \
  --manifest "${GITHUB_WORKSPACE}/release-staging/combined-manifest.json" \
  --version "${VERSION}"
```

That means `release.yml` only publishes a tag that is:

- anchored on `origin/main`
- coherent with committed `internal/bootstrap/version.go`
- coherent with committed `internal/bootstrap/checksums.go`
- rebuilt, smoke-tested, and signed inside CI

## Published Artifact Layout

Raw R2 artifacts and checksum material live under the immutable tree rooted at:

```text
https://releases.amikos.tech/pure-simdjson/v<version>/
```

Raw library URLs:

- `https://releases.amikos.tech/pure-simdjson/v<version>/linux-amd64/libpure_simdjson.so`
- `https://releases.amikos.tech/pure-simdjson/v<version>/linux-arm64/libpure_simdjson.so`
- `https://releases.amikos.tech/pure-simdjson/v<version>/darwin-amd64/libpure_simdjson.dylib`
- `https://releases.amikos.tech/pure-simdjson/v<version>/darwin-arm64/libpure_simdjson.dylib`
- `https://releases.amikos.tech/pure-simdjson/v<version>/windows-amd64/pure_simdjson-msvc.dll`
- `https://releases.amikos.tech/pure-simdjson/v<version>/SHA256SUMS`

Each raw object is published with `.sig` and `.pem` sidecars. `SHA256SUMS`
also gets `SHA256SUMS.sig` and `SHA256SUMS.pem`.

GitHub Releases publishes the same bytes under flat asset names:

- `libpure_simdjson-linux-amd64.so`
- `libpure_simdjson-linux-arm64.so`
- `libpure_simdjson-darwin-amd64.dylib`
- `libpure_simdjson-darwin-arm64.dylib`
- `pure_simdjson-windows-amd64-msvc.dll`
- `SHA256SUMS`

## Cosign Verification

`release.yml` uses keyless GitHub OIDC signing. Verification must use:

- certificate identity:
  `https://github.com/amikos-tech/pure-simdjson/.github/workflows/release.yml@refs/tags/v<version>`
- certificate issuer:
  `https://token.actions.githubusercontent.com`

Verify one raw artifact:

```bash
TAG=v0.1.0
BASE_URL="https://releases.amikos.tech/pure-simdjson/${TAG}"
LIB="libpure_simdjson.so"

curl -LO "${BASE_URL}/linux-amd64/${LIB}"
curl -LO "${BASE_URL}/linux-amd64/${LIB}.sig"
curl -LO "${BASE_URL}/linux-amd64/${LIB}.pem"

cosign verify-blob \
  --signature "${LIB}.sig" \
  --certificate "${LIB}.pem" \
  --certificate-identity "https://github.com/amikos-tech/pure-simdjson/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "${LIB}"
```

Verify `SHA256SUMS` itself:

```bash
TAG=v0.1.0
BASE_URL="https://releases.amikos.tech/pure-simdjson/${TAG}"

curl -LO "${BASE_URL}/SHA256SUMS"
curl -LO "${BASE_URL}/SHA256SUMS.sig"
curl -LO "${BASE_URL}/SHA256SUMS.pem"

cosign verify-blob \
  --signature SHA256SUMS.sig \
  --certificate SHA256SUMS.pem \
  --certificate-identity "https://github.com/amikos-tech/pure-simdjson/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  SHA256SUMS
```

## macOS Downloaded Dylibs

The macOS artifacts are ad-hoc signed in CI. A downloaded `.dylib` can still
arrive with a Gatekeeper quarantine attribute. If the loader is blocked after
download, remove the quarantine attribute and retry:

```bash
xattr -d com.apple.quarantine <path-to-dylib>
```

This exact operator step is the one referenced by Phase 6 Success Criterion 5
and `06-VALIDATION.md`.

For a repeatable local approximation of the downloaded-artifact load path on a
macOS workstation, run:

```bash
bash scripts/release/check_macos_downloaded_dylib.sh --build-local
```

That script builds or copies a local `.dylib`, re-signs a temp copy, applies a
synthetic `com.apple.quarantine` xattr, runs the native and Go smoke paths
before and after `xattr -d`, and records the current host's `spctl` result.
Treat it as a repo-local artifact revalidation probe, not a replacement for
the real published-artifact/fresh-machine UAT.

## Phase 06.1 Boundary

Phase `06.1` is where post-publish fresh-runner / fresh-machine public
validation happens. Phase 6 ends when the prep-then-tag CI release path, the
signatures, the checksum manifest, and this runbook are coherent. Do not turn
Phase 6 into manual workstation validation.
