---
phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
plan: 02
subsystem: benchmarking
tags: [benchmarks, evidence, linux-amd64, claim-gate]

requires:
  - phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
    provides: Plan 09-01 benchmark capture workflow and claim gate
provides:
  - Real linux/amd64 v0.1.2 benchmark evidence artifact imported for diagnosis
  - Machine-readable claim gate summary showing public docs are blocked
  - Workflow diagnostics for failed benchmark capture gates
affects: [benchmark-docs, readme-positioning, release-evidence]

tech-stack:
  added: []
  patterns:
    - Failed claim gates preserve complete benchmark evidence for diagnosis
    - summary.json target metadata is structured for downstream verification

key-files:
  created:
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
  - "Do not continue to public benchmark docs because the claim gate found material Tier 2 and Tier 3 regressions versus v0.1.1."
  - "Keep the real linux/amd64 evidence committed so the regression decision is inspectable and reproducible."

patterns-established:
  - "Benchmark capture workflow uploads evidence even when the claim gate exits nonzero."
  - "Claim-gate summaries expose target.goos and target.goarch as structured JSON fields."

requirements-completed: []

duration: 58m
completed: 2026-04-24
---

# Phase 09 Plan 02: Benchmark Evidence Gate Summary

**Linux/amd64 v0.1.2 evidence was captured and imported, but the claim gate blocks docs because Tier 2 and Tier 3 regressed versus v0.1.1.**

## Performance

- **Duration:** 58m
- **Started:** 2026-04-24T11:58:00Z
- **Completed:** 2026-04-24T12:55:56Z
- **Tasks:** 1 completed, 1 blocked
- **Files modified:** 15

## Accomplishments

- Made the benchmark workflow dispatchable by first landing PR #20 on the default branch, then pushed the Phase 9 branch and dispatched `benchmark-capture.yml` against it.
- Captured real linux/amd64 benchmark evidence on GitHub Actions run `24889843441` at commit `a47c561fd14ac1c580a38ba705ae7edab2debd1d`.
- Imported all 11 evidence files into `testdata/benchmark-results/v0.1.2/`.
- Fixed the claim gate to emit structured `target.goos` and `target.goarch` fields required by Plan 09-02 and downstream docs.
- Preserved the complete failed claim-gate summary for diagnosis.

## Task Commits

1. **Capture diagnostics** - `a47c561` (ci)
2. **Claim target metadata fix** - `774dfa9` (fix)
3. **Imported linux/amd64 evidence** - `9a3e1ca` (testdata)

## Files Created/Modified

- `testdata/benchmark-results/v0.1.2/phase9.bench.txt` - Raw Tier 1/2/3 linux/amd64 benchmark output.
- `testdata/benchmark-results/v0.1.2/coldwarm.bench.txt` - Raw cold/warm benchmark output.
- `testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt` - Raw Tier 1 diagnostic benchmark output.
- `testdata/benchmark-results/v0.1.2/*.benchstat.txt` - Old/new and same-snapshot stdlib benchstat comparisons.
- `testdata/benchmark-results/v0.1.2/metadata.json` - Target, toolchain, runner, commit, and command metadata.
- `testdata/benchmark-results/v0.1.2/summary.json` - Claim gate output with non-empty errors.
- `scripts/bench/check_benchmark_claims.py` - Emits structured target metadata.
- `scripts/bench/capture_release_snapshot.sh` - Reports capture substep failures and preserves complete snapshots when the claim gate fails.
- `.github/workflows/benchmark-capture.yml` - Uploads evidence artifacts even when capture exits nonzero.

## Decisions Made

No public docs or README benchmark wording were updated. The evidence is complete and target-correct, but `summary.json.errors` is non-empty, so Plan 09-03 remains blocked.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Structured target metadata**
- **Found during:** Task 1 (claim gate verification)
- **Issue:** `summary.json.target` was emitted as a string, but Plan 09-02 and Plan 09-03 require `target.goos` and `target.goarch`.
- **Fix:** Changed `check_benchmark_claims.py` to emit a target object and updated tests.
- **Files modified:** `scripts/bench/check_benchmark_claims.py`, `tests/bench/test_check_benchmark_claims.py`
- **Verification:** `python3 tests/bench/test_check_benchmark_claims.py`
- **Committed in:** `774dfa9`

**2. [Rule 3 - Blocking] Capture failure diagnostics**
- **Found during:** Task 1 (first Actions run)
- **Issue:** The workflow failed after benchmark capture without uploading artifacts or showing which gate failed.
- **Fix:** Added capture substep diagnostics, claim-gate summary logging, and `if: always()` artifact upload.
- **Files modified:** `scripts/bench/capture_release_snapshot.sh`, `.github/workflows/benchmark-capture.yml`
- **Verification:** `bash -n scripts/bench/capture_release_snapshot.sh`; rerun uploaded artifact `benchmark-evidence-v0.1.2-linux-amd64`.
- **Committed in:** `a47c561`

---

**Total deviations:** 2 auto-fixed (Rule 2: 1, Rule 3: 1)  
**Impact on plan:** Both fixes were necessary to preserve evidence and make the failure diagnosable. They did not relax benchmark gates or continue to docs.

## Issues Encountered

The claim gate exits nonzero with five regression errors:

- `tier2 regression for twitter_json: old=299108.00ns/op new=368603.00ns/op`
- `tier2 regression for citm_catalog_json: old=983510.00ns/op new=1231068.50ns/op`
- `tier2 regression for canada_json: old=2882704.00ns/op new=3133063.50ns/op`
- `tier3 regression for twitter_json: old=191788.00ns/op new=313001.00ns/op`
- `tier3 regression for citm_catalog_json: old=587661.00ns/op new=1175721.50ns/op`

Because this is a material claim-gate failure rather than incomplete, corrupted, wrong-target, or interrupted evidence, Plan 09-03 must not run.

## Verification

- `python3 tests/bench/test_prepare_stdlib_benchstat_inputs.py` - PASS
- `python3 tests/bench/test_check_benchmark_claims.py` - PASS
- `python3 -m json.tool testdata/benchmark-results/v0.1.2/metadata.json >/dev/null` - PASS
- `python3 -m json.tool testdata/benchmark-results/v0.1.2/summary.json >/dev/null` - PASS
- `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64 > testdata/benchmark-results/v0.1.2/summary.json` - FAIL EXPECTED, exit 1 with the regression errors above.

## User Setup Required

None.

## Next Phase Readiness

Blocked. Plan 09-03 docs work cannot proceed until the benchmark-positioning decision is revised or the measured Tier 2/Tier 3 regressions are addressed by a new plan.

## Self-Check: FAILED

The evidence files exist and metadata proves linux/amd64, but the plan success criteria require the claim gate to pass and the human provenance checkpoint to approve evidence with empty `summary.json.errors`. That gate failed.

---
*Phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post*
*Completed: 2026-04-24*
