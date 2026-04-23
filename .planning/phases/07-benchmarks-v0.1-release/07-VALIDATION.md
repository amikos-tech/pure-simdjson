---
phase: 7
slug: benchmarks-v0.1-release
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-22
audited: 2026-04-23
---

# Phase 7 - Validation Strategy

> Per-phase validation contract for benchmark, documentation, and legal-artifact work. Derived from `07-RESEARCH.md`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` for validation contracts, oracle coverage, and benchmark smoke runs; Rust `cargo test` for FFI/native allocator exports; shell scripts for benchstat helpers |
| **Config file** | none - test files, benchmark commands, testdata manifests, and docs are the source of truth |
| **Quick run command** | `go test ./... -count=1 -timeout 180s` |
| **Benchmark smoke command** | `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder|ColdStart|Warm|Tier1Diagnostics)_' -benchtime=1x -count=1` |
| **Full suite command** | `cargo test --release -- --test-threads=1 && go test ./... -race -count=1 -timeout 240s` |
| **Estimated runtime** | ~2-5 minutes locally for tests; longer for real benchmark capture |

---

## Sampling Rate

- **After every benchmark-code task:** `go test ./... -count=1 -timeout 180s`
- **After every FFI-stats task:** `cargo test --release -- --test-threads=1 && go test ./... -count=1 -timeout 180s`
- **After every benchmark-harness plan:** `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder|ColdStart|Warm|Tier1Diagnostics)_' -benchtime=1x -count=1`
- **Before README/changelog claim updates:** run `go test ./... -run 'TestPhase7ReleaseArtifactContract$|^Example' -count=1`, then capture fresh benchmark outputs and run `scripts/bench/run_benchstat.sh`
- **Before Phase 7 closeout:** run `go test ./... -run 'TestPhase7ReleaseArtifactContract$' -count=1` to verify the evidence/docs/legal files and closeout routing lines
- **Max feedback latency:** 240 seconds locally

---

## Per-Requirement Verification Map

| Req | Behavior | Test Type | Automated Command | File Exists | Status |
|-----|----------|-----------|-------------------|-------------|--------|
| BENCH-01 | Three benchmark tiers exist and are runnable | benchmark smoke | `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchtime=1x -count=1` | ✅ `benchmark_fullparse_test.go`, `benchmark_typed_test.go`, `benchmark_selective_test.go` | ✅ green |
| BENCH-02 | Canonical corpus is vendored locally | Go contract test | `go test ./... -run 'TestPhase7BenchmarkFixtureContract$' -count=1` | ✅ `phase7_validation_contract_test.go::TestPhase7BenchmarkFixtureContract` | ✅ green |
| BENCH-03 | Comparator set includes stdlib any/struct, `simdjson-go`, `sonic`, and `go-json` | Go contract + benchmark smoke | `go test ./... -run 'TestPhase7BenchmarkComparatorContract$' -count=1 && go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchtime=1x -count=1` | ✅ `benchmark_comparators*_test.go`, `phase7_validation_contract_test.go::TestPhase7BenchmarkComparatorContract` | ✅ green |
| BENCH-04 | Cold-start and warm are separate and benchstat-friendly | benchmark smoke + shell syntax | `go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchtime=1x -count=1 && bash -n scripts/bench/run_benchstat.sh` | ✅ `benchmark_coldstart_test.go`, `scripts/bench/run_benchstat.sh` | ✅ green |
| BENCH-05 | Native allocator stats are exposed and reported beside Go alloc counts | Rust tests + benchmark smoke | `cargo test --release native_alloc -- --test-threads=1 && python3 tests/abi/check_header.py include/pure_simdjson.h && go test ./... -run '^$' -bench 'Benchmark(Tier2Typed|Tier3SelectivePlaceholder|Tier1Diagnostics)_' -benchtime=1x -count=1` | ✅ `tests/rust_shim_minimal.rs`, `benchmark_native_alloc_test.go`, `benchmark_diagnostics_test.go` | ✅ green |
| BENCH-06 | Correctness oracle matches vendored expectations | Go test | `go test ./... -run 'TestJSONTestSuiteOracle$' -count=1` | ✅ `benchmark_oracle_test.go::TestJSONTestSuiteOracle` | ✅ green |
| BENCH-07 | Public benchmark positioning is truthful and evidence-backed | Go contract + example tests | `go test ./... -run 'TestPhase7ReleaseArtifactContract$|^Example' -count=1` | ✅ `phase7_validation_contract_test.go::TestPhase7ReleaseArtifactContract` | ✅ green |
| DOC-01 | README contains installation, quick start, platform matrix, benchmark snapshot | Go contract + example tests | `go test ./... -run 'TestPhase7ReleaseArtifactContract$|^Example' -count=1` | ✅ `README.md`, `phase7_validation_contract_test.go::TestPhase7ReleaseArtifactContract` | ✅ green |
| DOC-06 | Changelog remains Keep-a-Changelog and captures Phase 7 work | Go contract test | `go test ./... -run 'TestPhase7ReleaseArtifactContract$' -count=1` | ✅ `CHANGELOG.md`, `phase7_validation_contract_test.go::TestPhase7ReleaseArtifactContract` | ✅ green |
| DOC-07 | MIT license and simdjson notice are committed | Go contract test | `go test ./... -run 'TestPhase7ReleaseArtifactContract$' -count=1` | ✅ `LICENSE`, `NOTICE`, `phase7_validation_contract_test.go::TestPhase7ReleaseArtifactContract` | ✅ green |

*Status: ✅ green · ❌ red · ⚠ flaky*

---

## Fault Injection Test Matrix

| Fault | Test Pattern | Expected Behavior | Requirement |
|-------|--------------|-------------------|-------------|
| benchmark fixture missing or renamed | remove one `testdata/bench/*.json` path and run benchmark loader | benchmark suite fails with a concrete missing-path error | BENCH-02 |
| comparator unsupported on the current target | run benchmark suite on an unsupported target/toolchain | comparator is omitted or skipped, not rendered as fake `N/A` | BENCH-03 |
| warm benchmark accidentally includes setup cost | inspect missing `b.ResetTimer()` or parser warm-up | validation fails grep gate before numbers are published | BENCH-04 |
| native allocator counters drift or never reset | run allocator test twice in one process | second run reports near-zero residual counters after explicit reset | BENCH-05 |
| correctness manifest and vendored files drift | add/remove a JSON file without touching `expectations.tsv` | oracle test fails on manifest/file mismatch | BENCH-06 |
| README claim updated without fresh benchmark evidence | edit README only | docs grep passes, but BENCH-07 still fails without the committed evidence snapshot and truthful-positioning lines | BENCH-07 |

---

## Wave 0 Requirements

- [x] `testdata/bench/README.md`
- [x] `testdata/bench/twitter.json`
- [x] `testdata/bench/citm_catalog.json`
- [x] `testdata/bench/canada.json`
- [x] `testdata/bench/mesh.json`
- [x] `testdata/bench/numbers.json`
- [x] `testdata/jsontestsuite/README.md`
- [x] `testdata/jsontestsuite/expectations.tsv`
- [x] `benchmark_fixtures_test.go`
- [x] `benchmark_oracle_test.go`
- [x] `benchmark_schema_test.go`
- [x] `benchmark_comparators_test.go`
- [x] `benchmark_fullparse_test.go`
- [x] `benchmark_coldstart_test.go`
- [x] `benchmark_typed_test.go`
- [x] `benchmark_selective_test.go`
- [x] `benchmark_native_alloc_test.go`
- [x] `benchmark_diagnostics_test.go`
- [x] `phase7_validation_contract_test.go`
- [x] `scripts/bench/run_benchstat.sh`
- [x] `README.md`
- [x] `docs/benchmarks.md`
- [x] `LICENSE`
- [x] `NOTICE`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Benchmark wording is acceptable for public docs | BENCH-07 | Requires human review of measured numbers and caveats | inspect `docs/benchmarks.md`, the README snapshot, and `docs/benchmarks/results-v0.1.1.md` together |
| Phase 7 closeout routes unresolved Tier 1 work to the right future phases | BENCH-07 | depends on project-level planning judgment | review `07-06-SUMMARY.md`, `.planning/ROADMAP.md`, and `.planning/STATE.md` after closeout |

---

## Security Domain

### Applicable ASVS L1 Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V1 Architecture | yes | benchmark tiers and comparator rules are documented explicitly so public claims match actual workloads |
| V5 Input Validation | yes | vendored corpora and expectation manifests are local files with explicit path checks |
| V6 Cryptography | yes | benchmark-source provenance is recorded with committed raw files and stable result links |
| V14 Build / Deploy | yes | benchmark/docs/legal closeout remains reproducible through local commands and committed artifacts |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation | Test |
|---------|--------|---------------------|------|
| benchmark claim overstates the actual workload | T | Tier labeling plus README/doc caveats; Tier 1 diagnostics show materialization dominates parse | BENCH-01 / BENCH-07 |
| missing or swapped corpus file invalidates results | T | vendored local corpus plus README checksums | BENCH-02 |
| native allocations are hidden behind Go-only stats | I | explicit FFI allocator snapshot/reset exports and benchmark custom metrics | BENCH-05 |
| docs drift from committed evidence | T | results doc carries fixed truthful-positioning lines and raw-file links; README must link back to it | BENCH-07 |

---

## Validation Sign-Off

- [x] All tasks have an automated verify path or an explicit manual-review boundary
- [x] Sampling continuity: no long run of tasks without executable checks
- [x] Wave 0 covers all missing benchmark/doc/legal scaffolding
- [x] No watch-mode tooling required
- [x] Feedback latency < 240s for quick, benchmark-smoke, and race verification
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** refreshed 2026-04-23

---

## Validation Audit 2026-04-23

| Metric | Count |
|--------|-------|
| Requirements audited | 10 (7 BENCH + 3 DOC) |
| Gaps found | 6 |
| Resolved | 6 |
| Escalated | 0 |
| Automated commands green | 8 |

Fresh audit evidence:

- `go test ./... -run 'Test(Phase7BenchmarkFixtureContract|Phase7BenchmarkComparatorContract|Phase7ReleaseArtifactContract|JSONTestSuiteOracle)$|^Example' -count=1` passed.
- `go test ./... -count=1 -timeout 180s` passed.
- `cargo test --release -- --test-threads=1` passed (`50` Rust tests).
- `go test ./... -race -count=1 -timeout 240s` passed.
- `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder|ColdStart|Warm|Tier1Diagnostics)_' -benchtime=1x -count=1` passed.
- `cargo test --release native_alloc -- --test-threads=1` passed (`3` native allocator tests).
- `python3 tests/abi/check_header.py include/pure_simdjson.h` passed.
- `bash -n scripts/bench/run_benchstat.sh` passed.
- The audit added `phase7_validation_contract_test.go` to lock the vendored benchmark corpus manifest, the comparator registry contract, the public benchmark/results/changelog/legal artifacts, and the Phase 7 closeout routing references.
- The audit corrected the stale placeholder `pending` statuses in this file. Remaining manual-only work is limited to human review of wording quality and project-level judgment around the Phase 8 / Phase 9 handoff.
