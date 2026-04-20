# Phase 5: Bootstrap + Distribution - Context

**Gathered:** 2026-04-20
**Status:** Ready for planning

<domain>
## Phase Boundary

On a fresh machine, `NewParser()` must download the correct prebuilt shared library from CloudFlare R2 (with GitHub Releases as fallback), verify its SHA-256, cache it to the OS user-cache directory, and load it — while honoring `PURE_SIMDJSON_LIB_PATH` for air-gapped deployments and `PURE_SIMDJSON_BINARY_MIRROR` for corporate firewall / custom mirror setups. A companion CLI `cmd/pure-simdjson-bootstrap` supports offline pre-fetch of one or all platform artifacts.

This phase delivers the ten `DIST-01..10` requirements plus `DOC-05`. It does NOT change the Phase-3 `purego.RegisterFunc` binding shape, does NOT add On-Demand semantics, does NOT introduce zero-copy string views, and does NOT implement cosign verification in Go code — cosign is docs-only per DIST-10.

</domain>

<decisions>
## Implementation Decisions

### Loader precedence & auto-bootstrap triggering

- **D-01:** Loader chain for Phase 5 is `PURE_SIMDJSON_LIB_PATH` (full-path override, from Phase 3) → cache-hit → download-then-cache → fail. The Phase-3 local `target/release|debug|<triple>/...` branch is dropped. Maintainers working inside this repo set `PURE_SIMDJSON_LIB_PATH` to their `target/release/libpure_simdjson.<ext>` for local test runs, matching pure-tokenizers' convention.
- **D-02:** `NewParser()` auto-downloads on cache miss using an internal `http.Client{Timeout: 2 * time.Minute}` + 30s dial/TLS sub-timeouts. `NewParser()` signature stays ctx-less (no Phase-3 API break).
- **D-03:** `BootstrapSync(ctx context.Context, opts ...BootstrapOption) error` is the caller-owned preflight. It populates the same cache `NewParser()` would. After a successful `BootstrapSync`, `NewParser()` finds the artifact on the env→cache pass and never reaches the download branch.
- **D-04:** Cache-hit path does NOT re-verify SHA-256 on every `NewParser()` call. SHA-256 is verified once at download time; post-dlopen the ABI-version handshake from Phase 1 (`get_abi_version()`) is the runtime tamper-evidence guard. Cache-dir permissions (`0700` on unix per DIST-05) defend against cache poisoning.
- **D-05:** Concurrent first-run bootstrap across multiple processes is coordinated via a `.lock` file in the cache directory with `flock` (lifted from pure-onnx `bootstrap_lock_unix.go` + `bootstrap_lock_windows.go`). Atomic rename from `os.CreateTemp` prevents torn files.

### Cache layout & library version pinning

- **D-06:** Library version is pinned via a Go compile-time constant: `const Version = "0.1.0"` in `internal/bootstrap/version.go`. `ldflags -X` is explicitly rejected because consumer `go build` does not run our build flags — the const is the only mechanism that survives module distribution.
- **D-07:** Cache layout uses per-version subdirectories: `<userCacheDir>/pure-simdjson/v<Version>/<os>-<arch>/lib<name>.<ext>`. This mirrors the R2 URL layout from DIST-01 one-to-one and gives clean rollback + parallel-version coexistence on a shared cache.
- **D-08:** `internal/bootstrap/checksums.go` is a Go `map[string]string` keyed by the path fragment `"v<Version>/<os>-<arch>/lib<name>.<ext>"`. `version.go` + `checksums.go` are siblings in the same package — CI-05 generates both at release time in a single follow-up commit or in the tagged commit.
- **D-09:** On module upgrade (consumer pulls v0.1.1), `NewParser()` routes to a fresh subdirectory on first call and downloads the new artifact automatically. The previous `v0.1.0` tree is preserved on disk for rollback. A `Purge(ctx context.Context, keepLast int) error` cache-cleanup helper is **deferred to v0.2** and noted in Deferred Ideas below.
- **D-10:** Filenames follow the platform convention locked in REQUIREMENTS.md: `libpure_simdjson.so` (linux), `libpure_simdjson.dylib` (darwin), `pure_simdjson-msvc.dll` (windows). Toolchain suffix lives in the filename, not the path segment.
- **D-11:** `get_abi_version()` (Phase 1) remains a runtime check **post-dlopen** — orthogonal to version pinning. It guards against the rare case where SHA-256 passed but the loaded library's ABI is incompatible (corruption plus collision, or manual override via `PURE_SIMDJSON_LIB_PATH` pointing at a mismatched build). Mismatch returns `ErrABIVersionMismatch` from `NewParser()`.

### Network behavior: retry, fallback, timeouts, error classification

- **D-12:** Transport uses the Go standard library `net/http` only. No third-party retry wrappers (`hashicorp/go-retryablehttp`, `cenkalti/backoff`). Matches pure-* family stdlib-first convention.
- **D-13:** Retry policy: capped exponential backoff with AWS/Cloudflare "Full Jitter" — `backoff = min(500ms * 2^attempt, 8s) + rand.Float64()*500ms`, 3–4 attempts, using `math/rand/v2` (stdlib, auto-seeded in Go 1.22+). The linear `time.Duration(attempt)*time.Second` pattern from pure-onnx is explicitly upgraded — documented deviation.
- **D-14:** Sleep between retries is ctx-aware: `select { case <-time.After(d): case <-ctx.Done(): return ctx.Err() }`. Cancellation via caller's ctx on `BootstrapSync(ctx)` propagates within milliseconds, not up to one backoff interval.
- **D-15:** R2 → GitHub Releases fallback fires **after all R2 attempts are exhausted**, not in parallel. Matches DIST-01 "R2 primary" semantics, preserves GitHub's 60/hr unauthenticated rate limit for users who wouldn't otherwise hit it, and gives clean error attribution.
- **D-16:** Retryable HTTP statuses: 408, 429, 500, 502, 503, 504 — plus GitHub-specific 403 body-sniff for `"rate limit"` text per the pure-onnx pattern. Honor `Retry-After` header when present.
- **D-17:** Permanent (non-retryable) failures: 404, non-ratelimit 403, SHA-256 mismatch, ABI mismatch, HTTPS→HTTP redirect (TLS-interception signal — rejected via `http.Client.CheckRedirect`). These fail fast without retry and do NOT trigger GH fallback when the cause is structural (404 against R2 does fall through to GH; checksum mismatch does not retry anywhere).
- **D-18:** Two timeout clocks operate simultaneously. Per-request: `http.Client{Timeout: 2*time.Minute}` + `Transport{DialContext: 30s, TLSHandshakeTimeout: 10s, ResponseHeaderTimeout: 30s}`. Total operation: caller's `ctx` deadline on `BootstrapSync(ctx)`. Auto-path (`NewParser()` trigger) uses the internal timeouts only since `NewParser()` takes no ctx.
- **D-19:** `PURE_SIMDJSON_BINARY_MIRROR` overrides the R2 base URL for corporate-firewall / self-hosted mirror setups. GitHub Releases fallback still fires on mirror failure by default — many corporate firewalls allow `objects.githubusercontent.com` even when custom mirrors fail.
- **D-20:** `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` is an opt-out env var for hermetic deployments where GitHub egress is explicitly blocked. When set with `PURE_SIMDJSON_BINARY_MIRROR`, only the mirror is tried.
- **D-21:** Error classification uses stdlib `errors.Is` / `errors.As` without custom typed errors where possible: `context.DeadlineExceeded`, `*net.DNSError`, `*tls.RecordHeaderError`, `syscall.ECONNREFUSED`, `syscall.ECONNRESET`. The final error returned from `BootstrapSync` wraps these with a human-readable hint referencing `PURE_SIMDJSON_LIB_PATH` as the air-gapped escape hatch (pitfall #20 — actionable egress-block diagnosis).

### Bootstrap CLI scope (cmd/pure-simdjson-bootstrap)

- **D-22:** CLI framework is `spf13/cobra`. This is a deliberate deviation from the pure-* family's stdlib-CLI convention, justified by the v0.1 verb count (four subcommands) and low-cost growth path for v0.2 additions.
- **D-23:** v0.1 verbs: `fetch`, `verify`, `platforms`, `version`.
- **D-24:** `fetch` — downloads artifacts to the cache (or `--dest`). Flags: `--all-platforms`, `--target=os/arch` (repeatable), `--dest=<path>` (default: OS user cache), `--version=<semver>` (default: `bootstrap.Version`), `--mirror=<url>` (same semantics as `PURE_SIMDJSON_BINARY_MIRROR`). Maps 1:1 to the library's `BootstrapOption` setters.
- **D-25:** `verify` — re-verifies SHA-256 of locally cached artifacts against `internal/bootstrap/checksums.go`. Catches disk corruption, silent tampering, or cache mismatch after manual file operations. Does NOT run cosign — cosign is documented-only per D-28.
- **D-26:** `platforms` — lists the five supported OS/arch targets and indicates which are present in the local cache (✓ cached vs ✗ missing). Useful for air-gapped pre-flight inventory and bundle-building.
- **D-27:** `version` — prints library version (`bootstrap.Version`), Go runtime version, and build info (`runtime/debug.ReadBuildInfo()`). Useful for bug reports.
- **D-28:** Output shape: human-friendly progress to stderr, silent-on-success to stdout, non-zero exit on failure. No `--json` mode in v0.1 — add behind a flag only if ami-gin or other CI consumers request it.

### Cosign verification UX

- **D-29:** Cosign verification is **documented-only** in v0.1. `docs/bootstrap.md` includes a recipe using the official `cosign verify-blob --certificate-identity ... --certificate-oidc-issuer https://token.actions.githubusercontent.com` command, mirroring pure-onnx's `docs/releases.md` pattern exactly.
- **D-30:** No Go code imports `sigstore/sigstore-go` in v0.1. The library and CLI remain dep-lean. Neither sibling in the pure-* family (pure-onnx, pure-tokenizers) has in-process cosign verification; pure-simdjson follows that convention.
- **D-31:** SHA-256 integrity (DIST-03, pitfall #17) remains always-on and verified **before** `dlopen`. This is structurally separate from cosign and not optional. Cosign adds a provenance/tamper-evidence layer above SHA-256 for users who want to verify the release pipeline wasn't compromised.
- **D-32:** If user feedback during v0.1 shows demand for in-process verification, the v0.2 escape hatch is to add a `verify` subcommand variant or a separate `cmd/pure-simdjson-cosign-verify` binary that uses `sigstore-go` — containing the dep weight to an opt-in `go install` binary without touching library consumers.

### Environment variable surface (consolidated)

- `PURE_SIMDJSON_LIB_PATH` — full-path override; bypasses cache + download entirely (locked in Phase 3 D-06).
- `PURE_SIMDJSON_BINARY_MIRROR` — overrides R2 base URL; GH fallback still active.
- `PURE_SIMDJSON_DISABLE_GH_FALLBACK` — opt-out for hermetic mirror setups.
- (No `PURE_SIMDJSON_VERIFY_SIGNATURE` — cosign is docs-only.)
- (No `PURE_SIMDJSON_QUIET` — download is silent-on-success; no first-run log line.)

### The agent's Discretion

- Exact Go file layout under `internal/bootstrap/` and `cmd/pure-simdjson-bootstrap/` as long as the public `BootstrapSync(ctx, opts...)` signature and CLI verbs above remain stable.
- Exact `BootstrapOption` functional-options surface (e.g., `WithDest`, `WithTarget`, `WithMirror`, `WithVersion`) as long as flags map 1:1 to options.
- Exact progress-reporting shape in the CLI (progress bar, counter, percent) as long as it stays on stderr.
- Exact typed error surface for permanent-vs-retryable distinction — a thin internal sentinel type is fine; public API wraps via `errors.Is/As` per D-21.
- Exact cache-lock file name (`.lock` vs `.pure-simdjson.lock` etc.) and acquisition timeout budget (pure-onnx uses 2 min — sensible default).
- Whether to add `PURE_SIMDJSON_CACHE_DIR` override (not strictly required by REQUIREMENTS.md; `os.UserCacheDir()` + subdir is adequate for v0.1, but planner may add if ergonomics warrant).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements

- `.planning/ROADMAP.md` — Phase 5 goal, must-haves, nice-to-haves, and all five success criteria for Bootstrap + Distribution.
- `.planning/PROJECT.md` — project-level constraints: CloudFlare R2 as canonical distribution, SHA-256 table in Go source, ad-hoc macOS codesign (Phase 6), no-cgo consumer promise.
- `.planning/REQUIREMENTS.md` — `DIST-01` through `DIST-10` and `DOC-05`, which define the full distribution and documentation scope this phase delivers.

### Locked prior decisions

- `.planning/phases/03-go-public-api-purego-happy-path/03-CONTEXT.md` — Phase-3 loader decisions carried forward: `PURE_SIMDJSON_LIB_PATH` env override (D-06), full-path Windows `LoadLibrary` (D-09), `purego.RegisterFunc` only (D-10). Phase 5 layers cache + download into this chain and drops the local `target/` branch.
- `.planning/phases/01-ffi-contract-design/01-CONTEXT.md` — ABI version contract: `get_abi_version()` is a post-dlopen runtime check; `^0.1.x` semver constraint on the Go side; mismatch surfaces `ErrABIVersionMismatch`.

### Normative contract and code anchors

- `docs/ffi-contract.md` — ABI version handshake semantics (§ABI version) that the post-dlopen check in this phase must preserve unchanged.
- `include/pure_simdjson.h` — exported `get_abi_version()` signature (and other loader-touched symbols) that the Phase-3 loader already binds.
- `library_loading.go` — existing Phase-3 `resolveLibraryPath()`. Phase 5 rewrites this to add cache + download stages after the env-var check and before fail. Maintain the fully-resolved-path invariant for Windows (pitfall #29).
- `library_unix.go` / `library_windows.go` — existing platform `loadLibrary` / `lookupSymbol`. Already correct for full-absolute-path loading; no changes expected in Phase 5.
- `internal/ffi/bindings.go` — existing purego symbol binding. `get_abi_version` binding already present; Phase 5 does not extend the symbol table.

### Research and pattern guidance

- `.planning/research/SUMMARY.md` — identifies pure-onnx `bootstrap.go` as the canonical lift target for this phase; confirms stdlib-first network convention; locks "R2 primary + GH Releases fallback" at the project level.
- `.planning/research/ARCHITECTURE.md` — §"Bootstrap/download" layer: R2 primary + GH Releases fallback, SHA-256 table in Go source, OS-user-cache-dir storage, `PURE_SIMDJSON_LIB_PATH` override.
- `.planning/research/PITFALLS.md` — §#16 OS-cache + `0700` perms (non-world-writable), §#17 SHA-256 verification before `dlopen`, §#20 retry with exponential backoff + ctx cancellation + egress-block error clarity, §#29 Windows `LoadLibrary` full-path / no-bare-filename.

### Sibling reference implementations (study before writing, not in-repo)

- `github.com/amikos-tech/pure-onnx` — `ort/bootstrap.go`, `ort/bootstrap_lock_unix.go`, `ort/bootstrap_lock_windows.go`, `ort/environment.go`, `docs/releases.md`. Canonical lift target for retry/fallback/flock/atomic-rename code, per-version cache layout, and docs-only cosign pattern.
- `github.com/amikos-tech/pure-tokenizers` — `library_loading.go`, `download.go`. Comparison reference for env→cache→download chain (single-slot cache — pure-simdjson uses per-version instead).
- `github.com/amikos-tech/fast-distance` — purego loader pattern reference; less relevant to this phase (no bootstrap).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `library_loading.go` — existing Phase-3 `resolveLibraryPath()` is the extension point. `loadedLibrary` struct already caches `path`, `handle`, `implementationName`, `bindings` and is guarded by `libraryMu` — Phase 5 reuses this cache around the new resolver.
- `library_unix.go` / `library_windows.go` — `loadLibrary(path)` and `lookupSymbol(handle, name)` already honor the full-path invariant on Windows and use `purego.RTLD_NOW|RTLD_LOCAL` on unix. No changes expected.
- `errors.go` — existing `wrapLoadFailure(stage, err)` and typed error sentinels (`ErrABIVersionMismatch`, `ErrLoadFailure`, etc.) — Phase 5 adds bootstrap-specific sentinels (e.g., `ErrChecksumMismatch`, `ErrAllSourcesFailed`) but keeps the existing wrap-with-stage pattern.
- `internal/ffi/bindings.go` — `Bind(handle, lookup)` already binds `ImplementationName` and the full Phase-3 symbol set. Phase 5 does not extend the symbol table; bootstrap is orthogonal to the ABI surface.

### Established Patterns

- Single package-level `sync.Mutex` guards the cached loaded library. Bootstrap must not contend with this mutex during download — downloads happen before the mutex is acquired, and the mutex only protects the final handle-cache insertion.
- Typed error wrapping via `wrapLoadFailure(stage, err)` attaches a human-readable stage string. Phase 5 follows this pattern for bootstrap errors (e.g., "download v0.1.0/linux-amd64 from R2", "verify SHA-256 of <path>").
- Env var names use the `PURE_SIMDJSON_` prefix. Phase 5 adds `PURE_SIMDJSON_BINARY_MIRROR` and `PURE_SIMDJSON_DISABLE_GH_FALLBACK`.
- Filenames in the repo use kebab-case (`library_loading.go`, `library_unix.go`). Phase 5 code under `internal/bootstrap/` should follow the same convention.

### Integration Points

- **`library_loading.go::resolveLibraryPath()`** — rewrite to chain: env override → cache-hit → `BootstrapSync(internal ctx)` → cache-hit. Fail with wrapped error if all stages fail.
- **`internal/bootstrap/`** (new package) — `version.go` (const), `checksums.go` (map), `bootstrap.go` (`BootstrapSync` + `BootstrapOption`), `download.go` (http client + retry), `cache.go` (layout + flock + atomic rename), `url.go` (R2 + GH URL construction), platform `bootstrap_lock_unix.go` + `bootstrap_lock_windows.go`.
- **`cmd/pure-simdjson-bootstrap/`** (new package) — `main.go` + one file per subcommand (`fetch.go`, `verify.go`, `platforms.go`, `version.go`). Cobra root command wires them up.
- **`docs/bootstrap.md`** (new) — env var reference, mirror setup, air-gapped flow with `PURE_SIMDJSON_LIB_PATH`, corporate firewall workaround, cosign verify-blob recipe.
- **`internal/ffi/bindings.go`** — no changes in Phase 5; the ABI-version post-dlopen check already exists.

### Out-of-tree dependencies this phase introduces

- `github.com/spf13/cobra` — new dep for the CLI only (Option C locked per D-22).
- `golang.org/x/sys` — existing or new dep for `flock` (unix + windows). Check `go.mod` to confirm; already a common Go dependency.
- No other runtime deps. No sigstore, no go-retryablehttp, no backoff.

</code_context>

<specifics>
## Specific Ideas

- The single highest-leverage reuse in this phase is **lifting pure-onnx's `bootstrap.go` + `bootstrap_lock_*.go` almost verbatim**, then applying the two deliberate deviations: Full-Jitter exponential backoff with ctx-aware sleep (replacing linear `time.Sleep(attempt*1s)`) and per-version cache subdirectories (replacing single-slot overwrite if pure-onnx uses that).
- The environment variable surface must stay **tight** — three vars (`PURE_SIMDJSON_LIB_PATH`, `PURE_SIMDJSON_BINARY_MIRROR`, `PURE_SIMDJSON_DISABLE_GH_FALLBACK`). Resist additions unless they map to a locked requirement.
- SHA-256 verification is **pre-dlopen**. No exception. Pitfall #17 is the load-bearing rule — a corrupted `.so`/`.dll` that reaches `dlopen` is catastrophic.
- The `verify` CLI subcommand is about **SHA-256 re-verification against `checksums.go`**, not cosign. Cosign users run the cosign CLI per the docs recipe. Keeping the two verification layers distinct and documented separately is important for operator clarity.
- `NewParser()` signature **does not change** in Phase 5 — no ctx argument, no new error surface beyond existing load failures. This preserves Phase-3's public contract. All ctx-aware paths flow through `BootstrapSync(ctx)`.

</specifics>

<deferred>
## Deferred Ideas

- **`Purge(ctx context.Context, keepLast int) error`** — cache-cleanup helper that removes stale per-version subdirectories. Deferred to v0.2. Per-version layout makes accumulation visible (~2–5 MB per version), not problematic in v0.1.
- **In-process cosign verification via `sigstore-go`** — dep weight is disproportionate for v0.1 (TUF, Rekor, in-toto, protobuf transitive closure). Escape hatch if demand materializes: v0.2 `cosign-verify` subcommand or separate binary that isolates the dep from library consumers.
- **`--json` output mode for the bootstrap CLI** — not in v0.1. Add behind a flag only if ami-gin or other CI consumers ask.
- **`PURE_SIMDJSON_CACHE_DIR` override** — not strictly required; `os.UserCacheDir()` + subdir is adequate. The agent may add during planning if ergonomics warrant.
- **Cold-start benchmark characterizing first-parse latency including download** — nice-to-have per ROADMAP.md Phase 5 nice-to-haves list; Phase 7 benchmark work can fold this in.
- **HTTP Range resume for partial downloads** — always redownload from scratch is acceptable for <50MB artifacts in v0.1. Reconsider in v0.2 if artifacts grow or users report WAN issues.
- **`PURE_SIMDJSON_QUIET` + first-run download log line** — not shipping a log line in v0.1 (siblings are silent-on-success); env var not needed.
- **Shell-out to `cosign` CLI if on PATH** — considered and rejected for v0.1; revisit in v0.2 if users push back on docs-only.

</deferred>

---

*Phase: 05-bootstrap-distribution*
*Context gathered: 2026-04-20*
