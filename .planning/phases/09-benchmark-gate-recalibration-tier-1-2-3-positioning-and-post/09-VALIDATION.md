---
phase: 9
slug: benchmark-gate-recalibration-tier-1-2-3-positioning-and-post-abi-evidence-refresh
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-24
---

# Phase 9 - Validation Strategy

Per-phase validation contract for benchmark evidence capture, claim gating,
and release-facing documentation updates.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | Go `testing` benchmarks, Python `unittest`, Cargo test, Makefile contract checks |
| Config file | `go.mod`, `Cargo.toml`, `Makefile`, `.github/workflows/benchmark-capture.yml` |
| Quick run command | `python3 tests/bench/test_check_benchmark_claims.py && go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` |
| Full suite command | `go test ./... && cargo test -- --test-threads=1 && make verify-contract && python3 tests/bench/test_check_benchmark_claims.py` |
| Benchmark capture command | `go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/phase9.bench.txt` |
| Cold/warm capture command | `go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/coldwarm.bench.txt` |
| Diagnostic capture command | `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt` |
| Claim gate command | `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > testdata/benchmark-results/v0.1.2/summary.json` |
| Estimated runtime | ~2-5 minutes for full non-benchmark suite locally; public benchmark capture varies by runner |

---

## Sampling Rate

- After every script task commit: run `python3 tests/bench/test_check_benchmark_claims.py`.
- After every benchmark harness or comparator task commit: run `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1`.
- After every workflow task commit: run local grep/yaml checks and review workflow permissions for least privilege.
- After every docs task commit: verify docs reference `docs/benchmarks.md`, `docs/benchmarks/results-v0.1.2.md`, and `testdata/benchmark-results/v0.1.2/summary.json` consistently.
- After every plan wave: run `go test ./... && cargo test -- --test-threads=1 && make verify-contract && python3 tests/bench/test_check_benchmark_claims.py`.
- Before phase verification: real `linux/amd64` benchmark evidence must be captured, committed, and accepted by the claim gate.
- Max feedback latency: no three consecutive task commits without at least the quick run command.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 09-W0-01 | 01 | 1 | BENCH-07, D-12, D-19, D-20, D-21, D-22 | T-09-01, T-09-03 | Claim gate rejects missing rows, wrong target metadata, non-significant Tier 1 headline rows, Tier 2/3 regressions, and malformed benchmark input. | Python unit | `python3 tests/bench/test_check_benchmark_claims.py` | no - Wave 0 creates | pending |
| 09-W0-02 | 01 | 1 | BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05 | T-09-02 | Existing benchmark row names, comparator omission behavior, fixture loading, and native allocator metrics remain stable. | Go unit/smoke | `go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` | yes | pending |
| 09-W1-01 | 02 | 1 | D-03, D-04, D-12, D-22 | T-09-02, T-09-04 | Dispatchable workflow captures real `linux/amd64` evidence with least-privilege permissions and uploads temporary artifacts only. | workflow/static | `rg 'workflow_dispatch|runs-on: ubuntu-latest|contents: read|actions/upload-artifact' .github/workflows/benchmark-capture.yml` | no - Wave 1 creates | pending |
| 09-W1-02 | 02 | 1 | D-01, D-03, D-04, D-12 | T-09-02, T-09-03 | Release-scoped raw, benchstat, metadata, and summary artifacts exist under `testdata/benchmark-results/v0.1.2/` and gate target is `linux/amd64`. | benchmark/gate | `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64` | pending benchmark capture | pending |
| 09-W2-01 | 03 | 2 | BENCH-07, DOC-01, DOC-06, D-05, D-06, D-07, D-08, D-13, D-17, D-18 | T-09-03 | README, benchmark methodology, result doc, and changelog use the claim allowances from `summary.json` and keep release/default-install wording bounded. | doc grep/review | `rg 'docs/benchmarks.md|results-v0.1.2|v0.1.2|Tier 1|Tier 2|Tier 3' README.md docs/benchmarks.md docs/benchmarks/results-v0.1.2.md CHANGELOG.md` | pending docs update | pending |

---

## Wave 0 Requirements

- [ ] `scripts/bench/check_benchmark_claims.py` - generalized Tier 1/2/3 claim gate derived from the Phase 8 improvement gate.
- [ ] `tests/bench/test_check_benchmark_claims.py` - synthetic coverage for metadata mismatch, missing rows, malformed rows, unsupported/non-significant Tier 1 headline, Tier 2/3 regression, and claim allowance output.
- [ ] `.github/workflows/benchmark-capture.yml` - dispatchable `linux/amd64` capture workflow with least-privilege permissions and artifact upload.
- [ ] `testdata/benchmark-results/v0.1.2/` - release/upcoming-release scoped destination for committed raw benchmark, benchstat, metadata, and summary artifacts.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Benchmark workflow ran on real `linux/amd64` before docs changed | D-03, D-07, D-19 | Local machines may not match the required public target. | Confirm raw files and `summary.json` metadata include `goos: linux`, `goarch: amd64`, runner/CPU/toolchain details, and the release-candidate commit SHA. |
| README claim mode matches `summary.json` | D-05, D-06, D-08, D-17 | Wording must follow generated allowances, not human optimism. | Compare README benchmark wording against `claims.readme_mode` and allowed Tier 1/2/3 booleans in `summary.json`. |
| Phase 9 did not perform release publication or bootstrap alignment | D-15, D-16, Phase 09.1 boundary | Release tagging and default-install validation belong to Phase 09.1. | Review diff and summary for absence of `git tag`, `git push`, release artifact publication, checksum state alignment, or default-install validation claims. |
| Optional GitHub Pages/history was not silently enabled | D-11 | Pages/history requires write/deploy permissions and is auxiliary. | If workflow permissions include `pages: write`, `id-token: write`, or benchmark-action usage, confirm the plan explicitly treats it as optional and approved. |

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

- [x] All Phase 9 decisions D-01 through D-22 have at least one automated or manual validation route.
- [x] Sampling continuity requires script, Go, Cargo, contract, workflow, and docs checks at appropriate wave boundaries.
- [x] Wave 0 identifies missing gate, workflow, and evidence scaffolding before public docs are updated.
- [x] No watch-mode flags are used.
- [x] Real `linux/amd64` benchmark evidence is a phase gate, not an optional nicety.
- [x] `nyquist_compliant: true` set in frontmatter.

Approval: pending implementation verification
