---
status: complete
phase: 05-bootstrap-distribution
source: [05-01-SUMMARY.md, 05-02-SUMMARY.md, 05-03-SUMMARY.md, 05-04-SUMMARY.md, 05-05-SUMMARY.md, 05-06-SUMMARY.md]
started: 2026-04-20T13:46:36Z
updated: 2026-04-20T13:49:14Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: From a clean build state, `go build ./... && go build ./cmd/pure-simdjson-bootstrap` completes with no errors. Running the binary with no args prints root cobra usage listing four verbs (fetch, verify, platforms, version) and exits cleanly.
result: pass
evidence: `go clean -cache && go build ./...` (exit 0) + `go build -o /tmp/pure-simdjson-bootstrap ./cmd/pure-simdjson-bootstrap` (exit 0). Binary with no args printed root cobra usage listing all four verbs plus `completion` and `help` (cobra built-ins), exit 0.

### 2. `version` verb prints bootstrap version + Go runtime + build info
expected: `./pure-simdjson-bootstrap version` prints three lines including `library: 0.1.0` (bootstrap.Version), the Go runtime version, and module build info from debug.ReadBuildInfo.
result: pass
evidence: Output `library:  0.1.0` / `go:       go1.26.2` / `module:   v0.0.0-20260420132857-07f8058ed775+dirty` — exit 0.

### 3. `platforms` verb lists 5 target platforms with cache status
expected: `./pure-simdjson-bootstrap platforms` iterates the 5 SupportedPlatforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64). Each row shows the resolved cache path and a `missing` or `cached` indicator derived from os.Stat.
result: pass
evidence: All 5 rows present with `missing` indicator (cache is empty on this machine, expected since Checksums map is empty pre-CI-05). Exit 0.

### 4. `fetch --help` wires all 5 expected flags
expected: `./pure-simdjson-bootstrap fetch --help` lists these flags: `--all-platforms`, `--target`, `--dest`, `--version`, `--mirror`.
result: pass
evidence: All 5 flags present — `--all-platforms`, `--target stringArray` (repeatable), `--dest string`, `--version string`, `--mirror string`. Help text references PURE_SIMDJSON_BINARY_MIRROR env var equivalence.

### 5. `verify --help` wires expected flags
expected: `./pure-simdjson-bootstrap verify --help` lists at minimum `--all-platforms` and `--dest`.
result: pass
evidence: Both flags present — `--all-platforms` and `--dest string` with offline-bundle use case documented inline.

### 6. Full test suite passes with `-race`
expected: `go test ./... -count=1 -race -timeout 120s` completes with all packages reporting `ok`. No FAIL, no DATA RACE output.
result: pass
evidence: 4 packages green — `github.com/amikos-tech/pure-simdjson` (9.087s), `.../cmd/pure-simdjson-bootstrap` (2.108s), `.../internal/bootstrap` (17.800s), `.../internal/ffi` (2.346s). Exit 0, no race warnings.

### 7. `docs/bootstrap.md` covers env vars + all 4 flows
expected: File contains table of 4 env vars, sections for Air-Gapped Deployment, Corporate Firewall / Custom Mirror, Verifying Artifact Integrity (cosign), Retry / Error Behavior, and L5 Phase-6 honesty note.
result: pass
evidence: 16 occurrences of the 4 env var names. All required sections present at lines 33 (Environment Variables), 42 (Air-Gapped Deployment), 67 (Corporate Firewall / Custom Mirror), 136 (Verifying Artifact Integrity / Cosign), 166 (Retry and Error Behavior), 204 (Testing and Release Scope — L5 honesty note).

## Summary

total: 7
passed: 7
issues: 0
pending: 0
skipped: 0

## Gaps

[none]
