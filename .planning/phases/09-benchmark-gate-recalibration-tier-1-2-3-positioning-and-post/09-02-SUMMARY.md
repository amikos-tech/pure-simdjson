---
phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
plan: 02
subsystem: benchmarking
tags: [benchmarks, evidence, linux-amd64, claim-gate]

requires:
  - phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
    provides: Plan 09-01 benchmark capture workflow and claim gate
provides:
  - Real linux/amd64 pre-Phase-8 baseline evidence for future gates
  - Real linux/amd64 v0.1.2 benchmark evidence
  - Passing machine-readable claim gate summary for Plan 09-03
affects: [benchmark-docs, readme-positioning, release-evidence]

tech-stack:
  added: []
  patterns:
    - GitHub Actions linux/amd64 is the canonical benchmark evidence target
    - Failed capture gates preserve complete benchmark evidence for diagnosis
    - summary.json target metadata is structured for downstream verification

key-files:
  created:
    - testdata/benchmark-results/v0.1.1-linux-amd64/phase7.bench.txt
    - testdata/benchmark-results/v0.1.1-linux-amd64/coldwarm.bench.txt
    - testdata/benchmark-results/v0.1.1-linux-amd64/tier1-diagnostics.bench.txt
    - testdata/benchmark-results/v0.1.1-linux-amd64/metadata.json
    - testdata/benchmark-results/v0.1.2/phase9.bench.txt
    - testdata/benchmark-results/v0.1.2/coldwarm.bench.txt
    - testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt
    - testdata/benchmark-results/v0.1.2/summary.json
  modified:
    - scripts/bench/check_benchmark_claims.py
    - tests/bench/test_check_benchmark_claims.py
    - scripts/bench/capture_release_snapshot.sh
    - .github/workflows/benchmark-capture.yml

key-decisions:
  - "Use linux/amd64 CI evidence as the durable old/new benchmark baseline because future gates run on GitHub Actions, not developer machines."
  - "Keep the old darwin/arm64 v0.1.1 evidence as historical context only; it is not comparable for linux/amd64 claim gating."
  - "Plan 09-03 may proceed because summary.json now has empty errors and all claim booleans are true."

patterns-established:
  - "Benchmark capture workflow uploads evidence even when the claim gate exits nonzero."
  - "Claim-gate summaries expose target.goos and target.goarch as structured JSON fields."
  - "Claim-gate old/new comparisons fail closed when baseline and snapshot target metadata do not match."

requirements-completed: [BENCH-01, BENCH-02, BENCH-03, BENCH-04, BENCH-05, BENCH-07]

duration: 58m + follow-up baseline correction
completed: 2026-04-24
---

# Phase 09 Plan 02: Benchmark Evidence Gate Summary

**Linux/amd64 baseline and v0.1.2 evidence are committed, and the claim gate now passes against the CI target baseline.**

## Accomplishments

- Landed PR #20 so `benchmark-capture.yml` can be dispatched from GitHub Actions.
- Captured real linux/amd64 v0.1.2 evidence on GitHub Actions run `24889843441`.
- Captured a same-target pre-Phase-8 linux/amd64 baseline on GitHub Actions run `24892718570`.
- Imported the pre-Phase-8 baseline under `testdata/benchmark-results/v0.1.1-linux-amd64/`.
- Regenerated v0.1.2 old/new benchstat outputs against the linux/amd64 baseline.
- Fixed claim-gate metadata handling so cross-target baselines fail with metadata mismatch instead of fake regression errors.
- Fixed claim-gate benchstat row matching for real benchstat output, which omits the `Benchmark` prefix.
- Regenerated `testdata/benchmark-results/v0.1.2/summary.json`; it now exits 0 with empty errors.

## Claim Result

`summary.json` now reports:

- `claims.readme_mode`: `tier1_headline`
- `claims.tier1_headline_allowed`: `true`
- `claims.tier2_headline_allowed`: `true`
- `claims.tier3_headline_allowed`: `true`
- `errors`: `[]`

## Verification

- `python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py` - PASS
- `python3 tests/bench/test_check_benchmark_claims.py` - PASS
- `bash -n scripts/bench/capture_release_snapshot.sh` - PASS
- `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > testdata/benchmark-results/v0.1.2/summary.json` - PASS

## Next Phase Readiness

Plan 09-03 is unblocked. Public benchmark docs can now consume `testdata/benchmark-results/v0.1.2/summary.json` and the committed linux/amd64 evidence.

## Self-Check: PASSED

Evidence files exist, baseline and snapshot target metadata match linux/amd64, and the claim gate exits 0.

---
*Phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post*
*Completed: 2026-04-24*
