---
phase: 7
slug: benchmarks-v0.1-release
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-22
---

# Phase 7 - Validation Strategy

> Per-phase validation contract for benchmark, documentation, and release-close work. Derived from `07-RESEARCH.md`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` for oracle + benchmarks, Rust `cargo test` for any new FFI stats exports, shell scripts for benchstat/release readiness |
| **Config file** | none - benchmark commands, testdata manifests, and release scripts are the source of truth |
| **Quick run command** | `go test ./... -count=1 -timeout 180s` |
| **Benchmark smoke command** | `go test ./... -run '^$' -bench 'Benchmark(Tier1|Tier2|Tier3|ColdStart|Warm)_' -benchtime=1x -count=1` |
| **Full suite command** | `cargo test --release && go test ./... -race -count=1 -timeout 240s && bash scripts/release/check_readiness.sh --strict --version 0.1.1` |
| **Estimated runtime** | ~2-5 minutes locally for tests; longer for real benchmark capture and release workflows |

---

## Sampling Rate

- **After every benchmark-code task:** `go test ./... -count=1 -timeout 180s`
- **After every FFI-stats task:** `cargo test --release && go test ./... -count=1 -timeout 180s`
- **After every benchmark-harness plan:** `go test ./... -run '^$' -bench 'Benchmark(Tier1|Tier2|Tier3|ColdStart|Warm)_' -benchtime=1x -count=1`
- **Before README/changelog claim updates:** capture fresh benchmark outputs and run `scripts/bench/run_benchstat.sh`
- **Before release-close:** `bash scripts/release/check_readiness.sh --strict --version 0.1.1`
- **Max feedback latency:** 240 seconds locally, excluding the tag-driven GitHub workflows

---

## Per-Requirement Verification Map

| Req | Behavior | Test Type | Automated Command | File Exists | Status |
|-----|----------|-----------|-------------------|-------------|--------|
| BENCH-01 | Three benchmark tiers exist and are runnable | benchmark | `go test ./... -run '^$' -bench 'BenchmarkTier(1|2|3)_' -benchtime=1x -count=1` | ❌ Wave 0 | ⬜ pending |
| BENCH-02 | Canonical corpus is vendored locally | file + grep | `test -f testdata/bench/twitter.json && test -f testdata/bench/citm_catalog.json && test -f testdata/bench/canada.json && test -f testdata/bench/mesh.json && test -f testdata/bench/numbers.json && rg 'twitter.json|citm_catalog.json|canada.json|mesh.json|numbers.json|sha256' testdata/bench/README.md` | ❌ Wave 0 | ⬜ pending |
| BENCH-03 | Comparator set includes stdlib any/struct, `simdjson-go`, `sonic`, and `go-json` | grep + benchmark | `rg 'github.com/minio/simdjson-go|github.com/bytedance/sonic|github.com/goccy/go-json|encoding/json' go.mod go.sum benchmark_*_test.go && go test ./... -run '^$' -bench 'BenchmarkTier(1|2|3)_' -benchtime=1x -count=1` | ❌ Wave 0 | ⬜ pending |
| BENCH-04 | Cold-start and warm are separate and benchstat-friendly | grep + shell | `rg 'BenchmarkColdStart_|BenchmarkWarm_|ResetTimer|ReportAllocs|SetBytes' benchmark_*_test.go && bash -n scripts/bench/run_benchstat.sh` | ❌ Wave 0 | ⬜ pending |
| BENCH-05 | Native allocator stats are exposed and reported beside Go alloc counts | Rust + Go + grep | `cargo test --release native_alloc && go test ./... -run '^$' -bench 'Benchmark(Tier2|Tier3)_' -benchtime=1x -count=1 && rg 'pure_simdjson_native_alloc_stats_reset|pure_simdjson_native_alloc_stats_snapshot|native-bytes/op|native-allocs/op|native-live-bytes' src/lib.rs include/pure_simdjson.h internal/ffi/bindings.go benchmark_native_alloc_test.go` | ❌ Wave 0 | ⬜ pending |
| BENCH-06 | Correctness oracle matches vendored expectations | test | `go test ./... -run TestJSONTestSuiteOracle -count=1` | ❌ Wave 0 | ⬜ pending |
| BENCH-07 | README claims are backed by benchmark snapshot + caveats | grep + docs | `rg '>=3x|within 2x|minio/simdjson-go|Benchmark Snapshot|Methodology' README.md docs/benchmarks.md CHANGELOG.md` | ❌ Wave 0 | ⬜ pending |
| DOC-01 | README contains installation, quick start, platform matrix, benchmark snapshot | grep | `test -f README.md && rg '^# pure-simdjson$|## Installation|## Quick Start|## Supported Platforms|## Benchmark Snapshot' README.md` | ❌ Wave 0 | ⬜ pending |
| DOC-06 | Changelog remains Keep-a-Changelog and captures Phase 7 work | grep | `test -f CHANGELOG.md && rg '^## \\[Unreleased\\]|^## \\[0\\.1\\.1\\]|Keep a Changelog|Benchmark' CHANGELOG.md` | ❌ Wave 0 | ⬜ pending |
| DOC-07 | MIT license and simdjson notice are committed | file + grep | `test -f LICENSE && test -f NOTICE && rg 'MIT License|Apache License|simdjson' LICENSE NOTICE third_party/simdjson/LICENSE third_party/simdjson/LICENSE-MIT` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠ flaky*

---

## Fault Injection Test Matrix

| Fault | Test Pattern | Expected Behavior | Requirement |
|-------|--------------|-------------------|-------------|
| benchmark fixture missing or renamed | remove one `testdata/bench/*.json` path and run benchmark loader | benchmark suite fails with a concrete missing-path error | BENCH-02 |
| comparator unsupported on the current target | run benchmark suite on an unsupported target/toolchain | comparator is omitted or skipped, not rendered as fake `N/A` | BENCH-03 |
| warm benchmark accidentally includes setup cost | inspect missing `b.ResetTimer()` or parser warm-up | validation fails grep gate before numbers are published | BENCH-04 |
| native allocator counters drift or never reset | run allocator test twice in one process | second run reports near-zero residual counters after explicit reset | BENCH-05 |
| correctness manifest and vendored files drift | add/remove a JSON file without touching `expectations.tsv` | oracle test fails on manifest/file mismatch | BENCH-06 |
| README claim updated without fresh benchmark evidence | edit README only | docs grep passes, but release-close plan blocks on fresh benchstat capture | BENCH-07 |
| release-close tries to reuse `v0.1.0` | run release task with `--version 0.1.0` after Phase 7 changes | readiness/tagging path is rejected; patch release remains required | DOC-06 / BENCH-07 |

---

## Wave 0 Requirements

- [ ] `testdata/bench/README.md`
- [ ] `testdata/bench/twitter.json`
- [ ] `testdata/bench/citm_catalog.json`
- [ ] `testdata/bench/canada.json`
- [ ] `testdata/bench/mesh.json`
- [ ] `testdata/bench/numbers.json`
- [ ] `testdata/jsontestsuite/README.md`
- [ ] `testdata/jsontestsuite/expectations.tsv`
- [ ] `benchmark_fixtures_test.go`
- [ ] `benchmark_oracle_test.go`
- [ ] `benchmark_schema_test.go`
- [ ] `benchmark_comparators_test.go`
- [ ] `benchmark_fullparse_test.go`
- [ ] `benchmark_coldstart_test.go`
- [ ] `benchmark_typed_test.go`
- [ ] `benchmark_selective_test.go`
- [ ] `benchmark_native_alloc_test.go`
- [ ] `scripts/bench/run_benchstat.sh`
- [ ] `README.md`
- [ ] `docs/benchmarks.md`
- [ ] `LICENSE`
- [ ] `NOTICE`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Benchmark headline is acceptable for public release notes | BENCH-07 | Requires human review of the measured numbers and caveats | inspect `docs/benchmarks.md`, the README snapshot, and the release notes before tagging |
| Patch release publish succeeds for the Phase 7 output | BENCH-07 / DOC-06 | depends on live GitHub Actions secrets, R2 publish, and annotated tag push | merge to `origin/main`, run `bash scripts/release/check_readiness.sh --strict --version 0.1.1`, push `v0.1.1`, and review `release.yml` |
| Fresh-runner public bootstrap validation passes for the Phase 7 release | BENCH-07 | depends on already-published artifacts and hosted GitHub runners | dispatch `.github/workflows/public-bootstrap-validation.yml` with `version=0.1.1` and review the matrix results |

---

## Security Domain

### Applicable ASVS L1 Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V1 Architecture | yes | benchmark tiers and comparator rules are documented explicitly so public claims match actual workloads |
| V5 Input Validation | yes | vendored corpora and expectation manifests are local files with explicit path checks |
| V6 Cryptography | yes | benchmark-source provenance is recorded with checksums; release-close still uses the existing signed CI path |
| V14 Build / Deploy | yes | patch release uses the same CI-only publish path already locked in Phase 6 |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation | Test |
|---------|--------|---------------------|------|
| benchmark claim overstates the actual workload | T | Tier labeling plus README/doc caveats; strict Tier 1 materialization parity | BENCH-01 / BENCH-07 |
| missing or swapped corpus file invalidates results | T | vendored local corpus plus README checksums | BENCH-02 |
| native allocations are hidden behind Go-only stats | I | explicit FFI allocator snapshot/reset exports and benchmark custom metrics | BENCH-05 |
| release-close tries to mutate existing published history | T | immutable `v0.1.0`; new public Phase 7 artifacts ship only through `v0.1.1` | DOC-06 / BENCH-07 |

---

## Validation Sign-Off

- [x] All tasks have an automated verify path or an explicit manual-release boundary
- [x] Sampling continuity: no long run of tasks without executable checks
- [x] Wave 0 covers all missing benchmark/doc/legal scaffolding
- [x] No watch-mode tooling required
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-22
