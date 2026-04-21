# Bootstrap and Distribution

`pure-simdjson` ships pre-built shared libraries for five platforms. On the
first call to `NewParser()`, the library automatically downloads, verifies
(SHA-256), caches, and loads the correct binary for your platform. No cgo and
no `go generate` step is required at consumer build time.

## Release Operators

Tag publication, published URL layout, `SHA256SUMS`, cosign verification, and
the CI-only publish sequence now live in [`docs/releases.md`](./releases.md).
Use that runbook for the tag-driven release sequence instead of reconstructing
the process from workflow YAML.

## How It Works

Resolution order on `NewParser()`:

1. **Env override.** If `PURE_SIMDJSON_LIB_PATH` is set, load from that path
   verbatim. No download, no cache touch, no network.
2. **Cache hit.** Look for the artifact in the OS user cache (override base
   with `PURE_SIMDJSON_CACHE_DIR`). If present, load it. No SHA-256 re-verify
   on cache hit — verification happened at install time.
3. **Bootstrap.** Download from CloudFlare R2 (primary). Resolve the expected
   SHA-256 from published `SHA256SUMS` metadata for the requested tag, then
   atomically install into the cache with `0700` permissions on unix. An
   exclusive flock guards the install so concurrent callers collapse into a
   single download.
4. **GitHub Releases fallback.** If R2 is unreachable or returns a non-success
   status, fall back to `github.com/amikos-tech/pure-simdjson/releases`. Set
   `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` to suppress this for hermetic
   deployments.
5. **Load and verify ABI.** Open the artifact via `purego`, bind symbols,
   verify the ABI version reported by the native shim matches the version
   compiled into the Go wrapper.

Failures from any stage are wrapped with a hint pointing back at
`PURE_SIMDJSON_LIB_PATH` so end users learn the bypass mechanism without
reading the source.

## Environment Variables

| Variable                            | Description                                                                                                                                                       | Default                                                                                                                                                                  |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `PURE_SIMDJSON_LIB_PATH`            | Full path to a pre-built library. If set, bypasses all download and cache logic. Use for air-gapped environments and CI runners that vendor the library directly. | unset                                                                                                                                                                    |
| `PURE_SIMDJSON_BINARY_MIRROR`       | Override the R2 base URL. Must be HTTPS for non-loopback hosts. GitHub Releases fallback still fires on mirror failure unless `DISABLE_GH_FALLBACK` is also set.   | `https://releases.amikos.tech/pure-simdjson`                                                                                                                             |
| `PURE_SIMDJSON_DISABLE_GH_FALLBACK` | Set to `1` to disable the GitHub Releases fallback. Use in hermetic deployments where GitHub egress is blocked.                                                   | unset                                                                                                                                                                    |
| `PURE_SIMDJSON_CACHE_DIR`           | Override the base directory used to cache downloaded artifacts. Useful for CI/CD runners with ephemeral or read-only `$HOME`.                                     | `$XDG_CACHE_HOME/pure-simdjson` (linux) / `~/Library/Caches/pure-simdjson` (darwin) / `%LocalAppData%\pure-simdjson` (windows). Falls back to `$TMPDIR/pure-simdjson-<uid>` if `os.UserCacheDir` fails. |

## Air-Gapped Deployment

Pre-fetch artifacts on a connected machine, transport the resulting directory
to the air-gapped host, and point `PURE_SIMDJSON_LIB_PATH` at the artifact for
the target platform.

```bash
# On the connected machine:
go install github.com/amikos-tech/pure-simdjson/cmd/pure-simdjson-bootstrap@latest
pure-simdjson-bootstrap fetch --all-platforms --dest ./vendor-libs

# (transport ./vendor-libs to the air-gapped host)

# On the air-gapped host:
export PURE_SIMDJSON_LIB_PATH=/path/to/vendor-libs/v0.1.0/linux-amd64/libpure_simdjson.so
```

With `PURE_SIMDJSON_LIB_PATH` set, no network calls are made — ever.

To verify an offline bundle end-to-end before transporting it:

```bash
pure-simdjson-bootstrap verify --all-platforms --dest ./vendor-libs
```

## Corporate Firewall / Custom Mirror

If your environment blocks `releases.amikos.tech` but allows internal artifact
mirrors:

```bash
# Layout under <MIRROR>/v<version>/<os>-<arch>/<libname>
export PURE_SIMDJSON_BINARY_MIRROR=https://artifacts.corp.internal/pure-simdjson
```

If the mirror is also unreachable but `objects.githubusercontent.com` is
allowed, the GitHub Releases fallback fires automatically. To suppress that
fallback (e.g. corporate policy forbids GitHub egress):

```bash
export PURE_SIMDJSON_DISABLE_GH_FALLBACK=1
```

`BootstrapSync` then returns `bootstrap.ErrAllSourcesFailed` if every mirror
URL fails, and the error message includes the `PURE_SIMDJSON_LIB_PATH` hint.

## Offline Pre-Fetch with the CLI

The `pure-simdjson-bootstrap` CLI pre-downloads artifacts for CI caching,
offline bundle building, or corporate mirrors:

```bash
# Install the CLI
go install github.com/amikos-tech/pure-simdjson/cmd/pure-simdjson-bootstrap@latest

# Download for all platforms to a custom directory (per-platform progress to stderr)
pure-simdjson-bootstrap fetch --all-platforms --dest ./libs

# Download for a specific platform
pure-simdjson-bootstrap fetch --target linux/amd64 --dest ./libs

# Verify cached artifacts (current platform; OS user cache)
pure-simdjson-bootstrap verify

# Verify an offline bundle end-to-end
pure-simdjson-bootstrap verify --all-platforms --dest ./libs

# List platforms and per-platform cache status
pure-simdjson-bootstrap platforms

# Show version information
pure-simdjson-bootstrap version
```

`fetch --all-platforms` emits one `fetching <os>/<arch>...` line to stderr
before each platform's download and one `  ok <os>/<arch>` line after each
success, so the CLI never appears silently hung during multi-platform
downloads.

## GitHub Release Asset Naming

Release artifacts are published to both CloudFlare R2 and GitHub Releases. R2
uses a directory layout per platform, so all files can share the same base
filename. GitHub Release assets live in a flat namespace, so the filenames are
platform-tagged to avoid collision:

| Source        | linux/amd64                       | linux/arm64                       | darwin/amd64                          | darwin/arm64                          | windows/amd64                            |
| ------------- | --------------------------------- | --------------------------------- | ------------------------------------- | ------------------------------------- | ---------------------------------------- |
| R2 (dir)      | `libpure_simdjson.so`             | `libpure_simdjson.so`             | `libpure_simdjson.dylib`              | `libpure_simdjson.dylib`              | `pure_simdjson-msvc.dll`                 |
| GitHub (flat) | `libpure_simdjson-linux-amd64.so` | `libpure_simdjson-linux-arm64.so` | `libpure_simdjson-darwin-amd64.dylib` | `libpure_simdjson-darwin-arm64.dylib` | `pure_simdjson-windows-amd64-msvc.dll`   |

The bootstrap library handles this transparently; you only see the R2-style
name on disk after the library caches the download.

## Downloaded macOS Dylibs

The published macOS `.dylib` artifacts are ad-hoc signed in CI. If Gatekeeper
blocks a downloaded dylib on first load, remove the quarantine attribute and
retry:

```bash
xattr -d com.apple.quarantine <path-to-dylib>
```

This is the same operator guidance captured in [`docs/releases.md`](./releases.md).
For a repo-local approximation of the downloaded-artifact load path, run
`bash scripts/release/check_macos_downloaded_dylib.sh --build-local` on a macOS
host.

## Verifying Artifact Integrity (Cosign)

Release artifacts are signed with cosign keyless OIDC signing. Verification is
optional but recommended for sensitive deployments. The mandatory integrity
layer is the SHA-256 check baked into the bootstrap library; cosign adds a
provenance layer on top.

```bash
TAG=v0.1.0
OS=linux
ARCH=amd64
BASE_URL="https://releases.amikos.tech/pure-simdjson/${TAG}"
LIB="libpure_simdjson.so"

curl -LO "${BASE_URL}/${OS}-${ARCH}/${LIB}"
curl -LO "${BASE_URL}/${OS}-${ARCH}/${LIB}.sig"
curl -LO "${BASE_URL}/${OS}-${ARCH}/${LIB}.pem"

cosign verify-blob \
  --signature "${LIB}.sig" \
  --certificate "${LIB}.pem" \
  --certificate-identity "https://github.com/amikos-tech/pure-simdjson/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "${LIB}"
```

The bootstrap library never invokes `cosign` and never imports any sigstore Go
package. SHA-256 verification is always performed by the library before
loading, regardless of whether the user runs the cosign step.

## Retry and Error Behavior

The bootstrap pipeline retries transient failures (HTTP 429, HTTP 5xx, GitHub
403 rate-limit responses) with exponential backoff plus additive jitter
(capped at 8 seconds). When a server returns a `Retry-After` header, the
bootstrap loop honors that hint (capped at 16 seconds) instead of the
computed backoff. The retry loop is context-aware: a cancelled context aborts
both in-flight downloads and the inter-attempt sleep within milliseconds.

URL-fatal errors (e.g. HTTP 404) skip the remaining retries for the current
URL and roll over to the next URL in the ladder, so a missing artifact on R2
still triggers the GitHub Releases fallback. Ladder-fatal errors (checksum
mismatch, missing checksum, HTTPS→HTTP redirect downgrade) abort the whole
ladder — no other URL can recover from them.

On blocked networks, bootstrap failures are memoized per-process for 30
seconds so repeated `NewParser()` calls fail fast instead of replaying the
retry ladder. A successful bootstrap clears the memoized failure.

If all download sources fail, the error message includes a hint pointing at
the bypass mechanism:

```
bootstrap failed (set PURE_SIMDJSON_LIB_PATH to bypass): all sources failed: ...
```

## Supported Platforms

| Platform      | Library Filename (on disk) |
| ------------- | -------------------------- |
| linux/amd64   | `libpure_simdjson.so`      |
| linux/arm64   | `libpure_simdjson.so`      |
| darwin/amd64  | `libpure_simdjson.dylib`   |
| darwin/arm64  | `libpure_simdjson.dylib`   |
| windows/amd64 | `pure_simdjson-msvc.dll`   |

linux/arm (32-bit) is intentionally not supported: `purego` v0.10.0 requires
`CGO_ENABLED=1` on that target, which would defeat the no-cgo promise this
library makes to consumers.

## Testing and Release Scope (v0.1)

The fresh-machine bootstrap flow is exercised in the `internal/bootstrap` test
suite via `net/http/httptest` with staged fake artifacts and synthetic SHA-256
values injected into `bootstrap.Checksums`. This validates the pipeline
end-to-end: URL construction, retry cascade, GitHub fallback, SHA-256
verification, atomic rename, flock concurrency, context cancellation, and
env-var overrides.

What now lives outside this bootstrap document:

- The Phase 6 release operator flow in [`docs/releases.md`](./releases.md):
  `release.yml`, publish-time verification, and `SHA256SUMS` / cosign
  commands.
- Fresh-runner public validation against the live
  `releases.amikos.tech` CDN and the
  `github.com/amikos-tech/pure-simdjson/releases` mirror. That follow-up is
  Phase `06.1`, not part of the publish runbook itself.

On ordinary development branches, `BootstrapSync` against an unpublished
version still returns `bootstrap.ErrNoChecksum` because no published
`SHA256SUMS` entry exists for that tag yet. Developers working inside this
repository set
`PURE_SIMDJSON_LIB_PATH` to their local `target/release/libpure_simdjson.<ext>`
build output to bypass the download pipeline. The repository's `TestMain` in
`testmain_test.go` does this automatically when the cargo artifact is present.
