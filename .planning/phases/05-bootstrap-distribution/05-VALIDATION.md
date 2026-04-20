---
phase: 5
slug: bootstrap-distribution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-20
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from `05-RESEARCH.md` §Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), `net/http/httptest` for HTTP mock |
| **Config file** | none — `go test` is stdlib-native |
| **Quick run command** | `go test ./internal/bootstrap/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -race -timeout 120s` |
| **Estimated runtime** | ~30–60 seconds (quick: ~5s, full: ~30s + race overhead) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/bootstrap/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -race -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds (quick); 120 seconds (full)

---

## Per-Requirement Verification Map

> Task IDs filled in by planner during `/gsd-plan-phase`. Requirements mapped below; planner must ensure every requirement has at least one task with automated verification.

| Req | Behavior | Test Type | Automated Command | File Exists | Status |
|-----|----------|-----------|-------------------|-------------|--------|
| DIST-01 | R2 URL construction for all 5 platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) | unit | `go test ./internal/bootstrap/... -run TestURLConstruction` | ❌ W0 | ⬜ pending |
| DIST-02 | GH Releases URL construction + fallback triggered after R2 exhaustion | unit | `go test ./internal/bootstrap/... -run TestFallback` | ❌ W0 | ⬜ pending |
| DIST-03 | SHA-256 verify passes on correct hash, fails on corrupted bytes | unit | `go test ./internal/bootstrap/... -run TestChecksumVerify` | ❌ W0 | ⬜ pending |
| DIST-03 | Corrupted download rejected before dlopen | integration | `go test ./internal/bootstrap/... -run TestCorruptedDownloadRejected` | ❌ W0 | ⬜ pending |
| DIST-04 | `BootstrapSync(ctx)` downloads, verifies, caches | integration (httptest) | `go test ./internal/bootstrap/... -run TestBootstrapSync` | ❌ W0 | ⬜ pending |
| DIST-04 | `BootstrapSync(ctx)` cancellation propagates within 50ms | unit | `go test ./internal/bootstrap/... -run TestBootstrapSyncCancellation` | ❌ W0 | ⬜ pending |
| DIST-05 | Cache directory created with 0700 perms on unix | unit | `go test ./internal/bootstrap/... -run TestCacheDirPerms` | ❌ W0 | ⬜ pending |
| DIST-05 | Second `NewParser()` call (cache hit) makes no HTTP requests | integration (httptest) | `go test ./... -run TestNewParserCacheHit` | ❌ W0 | ⬜ pending |
| DIST-06 | `PURE_SIMDJSON_LIB_PATH` set → no HTTP call made | unit | `go test ./... -run TestLibPathEnvBypassesDownload` | ✅ (library_loading_test.go) | ⬜ pending |
| DIST-07 | `PURE_SIMDJSON_BINARY_MIRROR` overrides R2 base URL | integration (httptest) | `go test ./internal/bootstrap/... -run TestMirrorOverride` | ❌ W0 | ⬜ pending |
| DIST-08 | `fetch` verb downloads all 5 platform artifacts to `--dest` | integration (httptest) | `go test ./cmd/pure-simdjson-bootstrap/... -run TestFetchCmd` | ❌ W0 | ⬜ pending |
| DIST-09 | `resolveLibraryPath()` never returns relative/bare path | unit | `go test ./... -run TestResolveLibraryPathAbsolute` | ❌ W0 | ⬜ pending |
| DIST-10 | cosign docs-only: no Go code imports sigstore | lint/grep | `! grep -r 'sigstore' . --include='*.go'` | — | ⬜ pending |
| DOC-05 | `docs/bootstrap.md` exists and covers env vars | manual/CI diff | `test -f docs/bootstrap.md && grep -q 'PURE_SIMDJSON_LIB_PATH' docs/bootstrap.md` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Fault Injection Test Matrix

> Required coverage. Every fault below maps to at least one automated test.

| Fault | Test Pattern | Expected Behavior | Requirement |
|-------|--------------|-------------------|-------------|
| Checksum corruption (body tampered) | `httptest.Server` returns file with flipped byte | `ErrChecksumMismatch`, no dlopen, no cache write | DIST-03 |
| HTTP 429 on first attempt, 200 on retry | `httptest.Server` returns 429 then 200 | retry succeeds, correct file written | DIST-04 (retry) |
| HTTP 503 on all R2 attempts, 200 on GH | `httptest.Server` mux with R2 returning 503 N times | falls back to GH, succeeds | DIST-02 |
| ctx cancel mid-download | Cancel ctx after first 1KB received | `context.Canceled`, temp file cleaned up | DIST-04 |
| ctx cancel during retry sleep | Cancel ctx during sleep | returns within 50ms, not full backoff interval | DIST-04 |
| Concurrent bootstrap (10 goroutines racing) | `sync.WaitGroup` with 10 goroutines | exactly one download, all find cached artifact | DIST-04 (lock) |
| HTTPS→HTTP redirect | `httptest.Server` returns 301 to `http://` | redirect rejected, permanent error | DIST-04 (TLS) |
| `.lock` file contention | Two processes: sleep in lock body | second process waits, acquires after first | DIST-04 (flock) |
| 404 on R2, 200 on GH | R2 mock returns 404, GH mock returns 200 | GH fallback fires, artifact cached | DIST-02 |
| `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` + R2 404 | env set, R2 returns 404 | fails with `ErrAllSourcesFailed`, no GH attempt | DIST-07 |
| GitHub 403 rate-limit body-sniff | 403 with `{"message": "API rate limit exceeded..."}` | classified as retryable | DIST-04 (error classification) |

---

## Wave 0 Requirements

Infrastructure that must exist before parallel tasks can run:

- [ ] `internal/bootstrap/` package created with `version.go` (const), `checksums.go` (map placeholder)
- [ ] `internal/bootstrap/bootstrap_test.go` — test file stubs for DIST-01..07
- [ ] `internal/bootstrap/bootstrap_lock_unix.go` + `bootstrap_lock_windows.go` — flock/LockFileEx
- [ ] `cmd/pure-simdjson-bootstrap/` directory created with `main.go` + verb stubs
- [ ] `cmd/pure-simdjson-bootstrap/fetch_test.go` — test file stub for DIST-08
- [ ] `github.com/spf13/cobra@v1.10.2` added to `go.mod`: `go get github.com/spf13/cobra@v1.10.2`
- [ ] `httptest` fixtures shared helper (if multiple test files need R2/GH mock)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `docs/bootstrap.md` is readable and covers all four flows (air-gapped, mirror, corporate firewall, cosign verify) | DOC-05 | Doc readability is a human judgment | Read doc; confirm each flow has a runnable example |
| Cold-start first-parse from fresh machine against live R2 | Success Criterion 1 | Requires fresh VM / cache clear and real network | `rm -rf ~/Library/Caches/pure-simdjson` on mac; `go run ./cmd/example-consumer` |
| Corporate firewall workaround actually works behind a real proxy | DIST-07 | Requires corporate network | Deferred to Phase 7 or user-reported validation |

*All core distribution behaviors have automated verification via `httptest`. The three manual items are environmental and cannot be automated meaningfully in CI.*

---

## Security Domain

> Cross-reference with PLAN.md `<threat_model>` blocks.

### Applicable ASVS L1 Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V4 Access Control | yes (cache dir) | `os.MkdirAll(dir, 0700)` on unix; Windows ACL default |
| V5 Input Validation | yes (URLs, mirror env var) | `url.Parse` + scheme check; reject HTTP for non-loopback |
| V6 Cryptography | yes (SHA-256 verify) | `crypto/sha256` stdlib; hex comparison is constant-time-safe (equality is fine — attacker gets no timing signal from map lookup) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation | Test |
|---------|--------|---------------------|------|
| MITM substitutes `.so` via corporate TLS proxy | Tampering | SHA-256 verify against embedded `checksums.go` before dlopen | `TestCorruptedDownloadRejected` |
| DLL hijacking via CWD on Windows | Elevation of privilege | Always absolute path to `windows.LoadLibrary` (pitfall #29) | `TestResolveLibraryPathAbsolute` |
| Cache directory world-writable → local priv esc | Tampering | `0700` perms on unix cache dir | `TestCacheDirPerms` |
| HTTPS→HTTP redirect downgrade | Tampering | `CheckRedirect` rejects downgrade | `TestRedirectDowngradeRejected` |
| HTTP mirror URL (no TLS) for non-loopback | Information disclosure | URL validation rejects HTTP for non-loopback hosts | `TestMirrorURLValidation` |
| GitHub mirror substitution if R2 compromised | Tampering | SHA-256 from `checksums.go` in Go source validates both sources | (same as DIST-03 tests) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags (tests are one-shot, CI-ready)
- [ ] Feedback latency < 30s (quick) / 120s (full)
- [ ] `nyquist_compliant: true` set in frontmatter (flip when planner finalizes task map)

**Approval:** pending
