# Phase 5: Bootstrap + Distribution - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `05-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-04-20
**Phase:** 05-bootstrap-distribution
**Areas discussed:** Loader precedence & auto-bootstrap, Cache layout & version pinning, Network behavior, Bootstrap CLI scope + cosign verification UX
**Mode:** advisor (full_maturity calibration)

---

## Loader precedence & auto-bootstrap triggering

| Option | Description | Selected |
|--------|-------------|----------|
| A. pure-tokenizers mirror | env → cache → download (drop local target/). Strict sibling parity; breaks Phase-3 dev loop. | |
| B. Layered + dev-gated local target/ | env → local target/ (Cargo.toml-sibling heuristic) → cache → download. Preserves dev loop. | |
| C. Explicit preflight only | BootstrapSync(ctx) required before NewParser. Violates DIST-05. | |
| D. Hybrid | NewParser auto-downloads with internal 30s timeout; BootstrapSync(ctx) preflight. No re-hash on cache hit. | ✓ |
| E. Full compat + first-run log line | B + stderr log + PURE_SIMDJSON_QUIET env var. | |

**User's choice:** D. Hybrid
**Notes:** Aligns with ami-gin's preflight preference (ami-gin calls BootstrapSync in its build pipeline → auto-path never fires for it). Satisfies both DIST-04 and DIST-05 verbatim. Drops the Phase-3 local target/ branch — maintainers set PURE_SIMDJSON_LIB_PATH for local test runs, matching pure-tokenizers convention.

---

## Cache layout & library version pinning

| Option | Description | Selected |
|--------|-------------|----------|
| A. Const + per-version subdirs | const Version + cache path v<version>/<os>-<arch>/. Matches pure-onnx + DIST-01 URL layout. | ✓ |
| B. Const + single-slot overwrite | One slot per platform overwritten on upgrade. Smallest disk, no rollback. | |
| C. Content-addressed | OCI-style blobs/sha256/ + refs. Strong dedup; Windows symlink friction. | |
| D. ldflags -X injected version | Structurally broken for a library (ldflags don't run in consumer's go build). | |
| E. go:embed VERSION file | Adds init-time parse error path for a value that's trivially a const. | |

**User's choice:** A. Const + per-version subdirs
**Notes:** CI-05 already commits checksums.go at release time → sibling version.go in same package costs zero. Per-version layout enables rollback and parallel-version coexistence. Purge(keepLast) helper deferred to v0.2.

---

## Network behavior: retry, fallback, timeouts

| Option | Description | Selected |
|--------|-------------|----------|
| A. Lift pure-onnx verbatim | stdlib net/http, linear attempt*1s sleep, 3 retries, flock + atomic rename. | |
| B. Lift + exponential+jitter + ctx-aware | A plus Full-Jitter backoff (500ms base, 8s cap), ctx-aware sleep, DISABLE_GH_FALLBACK env. | ✓ |
| C. hashicorp/go-retryablehttp | Mature retry wrapper; breaks stdlib-first convention. | |
| D. cenkalti/backoff/v5 | MPL-2.0 dep; solves only the easy 15 lines. | |
| E. Parallel R2 + GH racing | Doubles egress; burns GH rate limit; opaque error attribution. | |

**User's choice:** B. Lift pure-onnx + jitter + ctx-aware
**Notes:** Cloudflare's own R2 guidance recommends Full-Jitter backoff. Two-line ctx-aware sleep fix is principled and documentable. PURE_SIMDJSON_BINARY_MIRROR overrides R2 URL; GH fallback still fires by default; PURE_SIMDJSON_DISABLE_GH_FALLBACK=1 opt-out for hermetic setups.

---

## Bootstrap CLI scope (cmd/pure-simdjson-bootstrap)

| Option | Description | Selected |
|--------|-------------|----------|
| A. Minimal zero-flag | Single-shot current-platform fetch. Fails Success Criterion #5. | |
| B. Stdlib flag pkg | --all-platforms, --target, --dest, --version, --mirror. Stdlib-only. | |
| C. Cobra subcommands | fetch / verify / platforms / version. Rich UX; heavier dep tree. | ✓ |
| D. B + --json output | B plus structured output. Small extra surface. | |
| E. CLI also generates checksums.go | Conflates release-author concern with consumer concern. | |

**User's choice:** C. Cobra subcommands
**Notes:** Deliberate deviation from the pure-* family's stdlib-CLI convention, justified by the four-verb v0.1 scope and low-cost growth path for v0.2 additions. Neither sibling (pure-onnx, pure-tokenizers) ships a CLI at all.

### Follow-up — CLI verbs for v0.1

| Verb | Description | Selected |
|------|-------------|----------|
| fetch | Download current platform / --all / --target=os/arch to cache or --dest | ✓ |
| verify | Re-verify SHA-256 of cached artifacts against checksums.go. Not cosign. | ✓ |
| platforms | List supported targets with cache status | ✓ |
| version | Print library version, Go runtime, build info | ✓ |

**User's choice:** All four verbs.

---

## Cosign verification UX

| Option | Description | Selected |
|--------|-------------|----------|
| A. Documented-only | docs/bootstrap.md cosign verify-blob recipe. Matches pure-onnx exactly. | ✓ |
| B. Shell-out to cosign CLI if on PATH | No Go deps; os/exec friction; PATH fiddly on Windows. | |
| C. sigstore-go lib, env-opt-in | Massive transitive deps (TUF, Rekor, in-toto, protobuf). | |
| D. CLI-only verify subcommand via sigstore-go | Contains dep weight to CLI binary. | |
| E. Library auto-verifies when .sig + .pem present | Violates DIST-10 "optional"; breaks offline install. | |

**User's choice:** A. Documented-only
**Notes:** Matches pure-onnx's docs/releases.md approach. SHA-256 (DIST-03) remains always-on integrity gate. sigstore-go's ~93-line go.mod is disproportionate for a JSON parsing library. Escape hatch if demand materializes in v0.2: option D (CLI-only verify subcommand).

---

## Claude's Discretion

- Exact Go file layout under `internal/bootstrap/` and `cmd/pure-simdjson-bootstrap/`.
- Exact `BootstrapOption` functional-options surface (e.g., `WithDest`, `WithTarget`, `WithMirror`, `WithVersion`).
- Exact progress-reporting shape in the CLI (progress bar vs counter vs percent) — stderr only.
- Exact typed error surface for permanent-vs-retryable distinction.
- Exact cache-lock file name and acquisition timeout budget (pure-onnx uses 2min).
- Whether to add `PURE_SIMDJSON_CACHE_DIR` override (not required by REQUIREMENTS.md).

## Deferred Ideas

- `Purge(ctx, keepLast int)` cache-cleanup helper — v0.2
- In-process cosign verification via sigstore-go — v0.2 escape hatch
- `--json` output mode for the bootstrap CLI — only if requested
- `PURE_SIMDJSON_CACHE_DIR` override — planning's discretion
- Cold-start benchmark with download latency — Phase 7 benchmarks
- HTTP Range resume for partial downloads — v0.2 reconsideration
- `PURE_SIMDJSON_QUIET` + first-run download log line — not shipping; siblings silent-on-success
- Shell-out to cosign CLI if on PATH — v0.2 revisit if users push back on docs-only
