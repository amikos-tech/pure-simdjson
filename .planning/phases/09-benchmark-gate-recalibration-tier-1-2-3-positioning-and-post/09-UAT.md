---
status: complete
phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
source:
  - 09-01-SUMMARY.md
  - 09-02-SUMMARY.md
  - 09-03-SUMMARY.md
started: 2026-04-24T16:40:43Z
updated: 2026-04-24T16:42:15Z
completed: 2026-04-24T16:42:15Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

[testing complete]

## Tests

### 1. Claim Gate Accepts the Committed linux/amd64 Evidence
expected: Run `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64`. The command should exit 0 and emit JSON with `target.goos=linux`, `target.goarch=amd64`, `claims.readme_mode=tier1_headline`, all three `*_headline_allowed` booleans set to true, and `errors` as an empty list.
result: pass
evidence: The command exited 0 and emitted JSON with `target.goos`=`linux`, `target.goarch`=`amd64`, `claims.readme_mode`=`tier1_headline`, `tier1_headline_allowed`=`true`, `tier2_headline_allowed`=`true`, `tier3_headline_allowed`=`true`, and `errors`=`[]`.

### 2. Result Snapshot Page Matches the Committed Evidence
expected: Open `docs/benchmarks/results-v0.1.2.md`. It should identify the snapshot as `linux/amd64`, show claim mode `tier1_headline`, list durable evidence links under `testdata/benchmark-results/v0.1.2/`, include Tier 1, Tier 2, Tier 3, diagnostics, and cold/warm tables, and keep the release-boundary note that Phase 09.1 still owns tag readiness.
result: pass
evidence: `rg -n` confirmed `Claim mode: tier1_headline`, all three headline-allowed flags set to `true`, `Target: linux/amd64`, the durable `summary.json` link, the Tier 1/Tier 2/Tier 3 and cold/warm section headers, and the release-boundary note that Phase 09.1 still owns bootstrap artifact and default-install alignment before tagging.

### 3. README Benchmark Positioning Uses the Phase 9 Claims
expected: Open `README.md` and review the Benchmark Snapshot section. It should link to `docs/benchmarks/results-v0.1.2.md`, state the linux/amd64 Tier 1 headline ratios (`3.15x`, `3.39x`, `2.47x`), state the Tier 2 and Tier 3 ranges, and include the caveat that headline numbers come from linux/amd64 and other platforms may differ.
result: pass
evidence: `README.md` lines 63-67 link to `docs/benchmarks/results-v0.1.2.md`, include the Tier 1 ratios `3.15x`, `3.39x`, `2.47x`, include the Tier 2 range `12.49x` to `14.56x`, the Tier 3 range `15.18x` to `15.97x`, and keep the linux/amd64 caveat plus the Phase 09.1 release-boundary note.

### 4. Methodology and Changelog Point to the Same Source of Truth
expected: Open `docs/benchmarks.md` and `CHANGELOG.md`. The methodology doc should point at committed `testdata/benchmark-results/v0.1.2/` evidence, use the `-count=10` and `-timeout 1200s` rerun commands, and explain that the linux/amd64 CI baseline lives under `v0.1.1-linux-amd64`. The changelog should mention the committed linux/amd64 baseline, the committed `v0.1.2` evidence, and the README/docs recalibration to the new snapshot.
result: pass
evidence: `docs/benchmarks.md` points to committed `testdata/benchmark-results/v0.1.2/` evidence, keeps the `-count=10` and `-timeout 1200s` rerun commands, and references the linux/amd64 baseline under `v0.1.1-linux-amd64`; `CHANGELOG.md` records the committed linux/amd64 baseline, the committed `v0.1.2` evidence, and the README benchmark-positioning recalibration to the linux/amd64 `v0.1.2` snapshot.

### 5. Benchmark Capture Stays Manual, Read-Only, and Artifact-Preserving
expected: Open `.github/workflows/benchmark-capture.yml` and `scripts/bench/capture_release_snapshot.sh`. The workflow should be `workflow_dispatch` only, run on `ubuntu-latest`, keep permissions to `contents: read` and `actions: read`, and upload artifacts with `if: always()`. The capture script should stage a full snapshot, promote it only when complete, and preserve the staged snapshot even when the claim gate fails so the evidence can be diagnosed.
result: pass
evidence: `bash -n scripts/bench/capture_release_snapshot.sh` passed. `rg -n` confirmed `.github/workflows/benchmark-capture.yml` is `workflow_dispatch` only, runs on `ubuntu-latest`, keeps `contents: read` and `actions: read`, and uploads artifacts with `if: always()`. The capture script uses `stage_dir=\"$(mktemp -d ...)\"`, tracks `complete_snapshot=\"false\"` then `\"true\"`, defines `promote_stage()`, and logs `benchmark claim gate failed; preserving complete snapshot in $out_dir` before promoting the complete staged snapshot for diagnosis.

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[]
