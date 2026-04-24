---
phase: 9
slug: benchmark-gate-recalibration-tier-1-2-3-positioning-and-post-abi-evidence-refresh
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-24
audited: 2026-04-24
---

# Phase 9 - Validation Strategy

Per-phase validation contract for benchmark evidence capture, claim gating,
and release-facing documentation updates.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | Go `testing`, Python `unittest`, Cargo test, Makefile contract checks |
| Config file | `go.mod`, `Cargo.toml`, `Makefile`, `.github/workflows/benchmark-capture.yml` |
| Quick run command | `python3 tests/bench/test_check_benchmark_claims.py && python3 tests/bench/test_phase9_validation_contracts.py && go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle|TestPhase7ReleaseArtifactContract' -count=1` |
| Full suite command | `go test ./... && cargo test -- --test-threads=1 && make verify-contract && python3 tests/bench/test_check_benchmark_claims.py && python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py && python3 tests/bench/test_phase9_validation_contracts.py` |
| Benchmark capture command | `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/phase9.bench.txt` |
| Cold/warm capture command | `go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/coldwarm.bench.txt` |
| Diagnostic capture command | `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt` |
| Claim gate command | `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > testdata/benchmark-results/v0.1.2/summary.json` |
| Estimated runtime | ~2-5 minutes for full non-benchmark suite locally; public benchmark capture varies by runner |

---

## Sampling Rate

- After every script task commit: run `python3 tests/bench/test_check_benchmark_claims.py` and `python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py`.
- After every benchmark harness or comparator task commit: run `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1`.
- After every workflow or docs task commit: run `python3 tests/bench/test_phase9_validation_contracts.py`.
- After every evidence import/update: rerun the claim gate against the committed linux/amd64 baseline before touching docs.
- After every plan wave: run `go test ./... && cargo test -- --test-threads=1 && make verify-contract && python3 tests/bench/test_check_benchmark_claims.py && python3 tests/bench/test_phase9_validation_contracts.py`.
- Before phase verification: real `linux/amd64` benchmark evidence must be captured, committed, and accepted by the claim gate.
- Max feedback latency: no three consecutive task commits without at least the quick run command.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-W0-01 | 01 | 1 | BENCH-07, D-12, D-19, D-20, D-21, D-22 | T-09-01, T-09-03 | Claim gate rejects missing rows, wrong target metadata, non-significant Tier 1 headline rows, Tier 2/3 regressions, and malformed benchmark input. | Python unit | `python3 tests/bench/test_check_benchmark_claims.py` | ✅ `tests/bench/test_check_benchmark_claims.py` | ✅ green |
| 09-W0-02 | 01 | 1 | BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05 | T-09-02 | Existing benchmark row names, comparator omission behavior, fixture loading, native allocator metrics, and correctness oracle remain stable. | Go unit/smoke | `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` | ✅ `benchmark_comparators_test.go`, `benchmark_oracle_test.go` | ✅ green |
| 09-W1-01 | 02 | 1 | D-03, D-04, D-12, D-22 | T-09-02, T-09-04 | Dispatchable workflow captures real `linux/amd64` evidence with least-privilege permissions and uploads temporary artifacts only. | Python contract | `python3 tests/bench/test_phase9_validation_contracts.py` | ✅ `tests/bench/test_phase9_validation_contracts.py` | ✅ green |
| 09-W1-02 | 02 | 1 | D-01, D-03, D-04, D-12 | T-09-02, T-09-03 | Release-scoped raw, benchstat, metadata, and summary artifacts exist under `testdata/benchmark-results/v0.1.2/` and the claim gate accepts the `linux/amd64` target. | benchmark/gate + contract | `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 >/tmp/phase9-summary-check.json && python3 tests/bench/test_phase9_validation_contracts.py` | ✅ `testdata/benchmark-results/v0.1.2/{phase9,coldwarm,tier1-diagnostics}.bench.txt`, `metadata.json`, `summary.json` | ✅ green |
| 09-W2-01 | 03 | 2 | BENCH-07, DOC-01, DOC-06, D-05, D-06, D-07, D-08, D-13, D-17, D-18 | T-09-03 | README, benchmark methodology, result doc, and changelog use the claim allowances from `summary.json` and keep release/default-install wording bounded. | Python contract + Go contract | `python3 tests/bench/test_phase9_validation_contracts.py && go test ./... -run 'TestPhase7ReleaseArtifactContract$' -count=1` | ✅ `tests/bench/test_phase9_validation_contracts.py`, `phase7_validation_contract_test.go` | ✅ green |

---

## Wave 0 Requirements

- [x] `scripts/bench/check_benchmark_claims.py` - generalized Tier 1/2/3 claim gate derived from the Phase 8 improvement gate.
- [x] `tests/bench/test_check_benchmark_claims.py` - synthetic coverage for metadata mismatch, missing rows, malformed rows, unsupported/non-significant Tier 1 headline, Tier 2/3 regression, and claim allowance output.
- [x] `scripts/bench/prepare_stdlib_benchstat_inputs.py` - same-snapshot stdlib benchstat normalizer used by the capture path.
- [x] `tests/bench/test_prepare_stdlib_benchstat_inputs.py` - normalization coverage for metadata preservation and missing-fixture failures.
- [x] `scripts/bench/capture_release_snapshot.sh` - staged benchmark capture and summary generation entrypoint.
- [x] `.github/workflows/benchmark-capture.yml` - dispatchable `linux/amd64` capture workflow with least-privilege permissions and artifact upload.
- [x] `testdata/benchmark-results/v0.1.1-linux-amd64/` - durable linux/amd64 baseline for old/new gating.
- [x] `testdata/benchmark-results/v0.1.2/` - release-scoped raw benchmark, benchstat, metadata, and summary artifacts.
- [x] `docs/benchmarks/results-v0.1.2.md` - release-scoped public results snapshot backed by committed evidence.
- [x] `tests/bench/test_phase9_validation_contracts.py` - Phase 9 workflow/evidence/docs validation contract added during audit.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Benchmark workflow provenance matches the imported evidence bundle | D-03, D-07, D-19 | CI artifact origin and capture-source narrative still require human review beyond static file checks. | Confirm `metadata.json` and `09-02-SUMMARY.md` agree on the linux/amd64 run source, commit SHA, runner metadata, and capture timing. |
| README/result wording remains acceptable public copy | D-05, D-06, D-08, D-17 | Truthfulness is machine-gated, but phrasing quality is still a human judgment. | Compare README and `docs/benchmarks/results-v0.1.2.md` against `summary.json` and ensure the wording stays bounded to the allowed claims. |
| Phase 09.1 release boundary remains intact | D-15, D-16 | Absence of release publication/bootstrap alignment is best verified at the phase-diff level. | Review the Phase 9 summaries and the current diff for absence of `git tag`, `git push`, published-artifact claims, checksum alignment claims, or default-install validation claims. |
| Optional Pages/history stayed disabled | D-11 | This is a policy check on what was intentionally not added. | Confirm `.github/workflows/benchmark-capture.yml` does not introduce `pages: write`, `id-token: write`, or GitHub Pages deployment behavior. |

---

## Threat Model

| Ref | Threat | Mitigation | Verification |
|-----|--------|------------|--------------|
| T-09-01 | Benchmark gate accepts incomplete, malformed, or wrong-target data and unlocks unsupported public claims. | Strict script tests, required `linux/amd64` metadata, row completeness checks, and fail-closed claim allowances. | `python3 tests/bench/test_check_benchmark_claims.py` and claim gate exit status. |
| T-09-02 | Benchmark capture changes harness semantics while recalibrating claims, invalidating comparison with the prior public snapshot. | Reuse stable benchmark row names and existing fixture/comparator tests; do not redesign benchmark semantics in Phase 9. | `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` plus raw row grep. |
| T-09-03 | Public docs overstate Tier 1, Tier 2, Tier 3, platform, or release/default-install claims. | Generate docs from committed raw evidence plus `summary.json`; keep README stdlib-relative and route full tables to benchmark docs. | Grep docs for v0.1.2 paths and manually compare wording to `summary.json`. |
| T-09-04 | Workflow uses excessive token permissions or treats retention-limited artifacts as durable evidence. | Use `contents: read`/`actions: read`, upload artifacts only as transport, and require committed evidence under `testdata/benchmark-results/v0.1.2/`. | Workflow permission grep and committed evidence file checks. |

---

## Validation Sign-Off

- [x] All Phase 9 task rows have at least one automated verify path or an explicit manual-review boundary.
- [x] Sampling continuity requires script, Go, Cargo, workflow, evidence, and docs checks at appropriate wave boundaries.
- [x] Wave 0 scaffolding now exists on disk and is covered by passing automated checks.
- [x] No watch-mode flags are used.
- [x] Real `linux/amd64` benchmark evidence is a phase gate, not an optional nicety.
- [x] `nyquist_compliant: true` set in frontmatter.

Approval: validated 2026-04-24

---

## Validation Audit 2026-04-24

| Metric | Count |
|--------|-------|
| Task rows audited | 5 |
| Gaps found | 2 |
| Resolved | 2 |
| Escalated | 0 |
| Automated commands green | 6 |

Fresh audit evidence:

- `python3 tests/bench/test_check_benchmark_claims.py` passed.
- `python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py` passed.
- `python3 tests/bench/test_phase9_validation_contracts.py` passed.
- `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > /tmp/phase9-summary-check.json` passed.
- `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle|TestPhase7ReleaseArtifactContract' -count=1` passed.
- `cargo test -- --test-threads=1` passed.
- `make verify-contract` passed.

Audit notes:

- The audit added `tests/bench/test_phase9_validation_contracts.py` to lock the benchmark-capture workflow, committed linux/amd64 evidence snapshot, README/docs/changelog boundary language, and release-boundary invariants.
- The audit corrected the stale `pending` task statuses in this file so they now reflect the passing implementation and verification commands on current `HEAD`.
