---
phase: 10
phase_name: "Lightweight PR benchmark regression signal"
project: "pure-simdjson"
generated: "2026-07-22"
counts:
  decisions: 7
  lessons: 6
  patterns: 7
  surprises: 4
missing_artifacts: []
---

# Phase 10 Learnings: Lightweight PR benchmark regression signal

## Decisions

### Keep PR Regression Policy Separate from the Release Claim Gate
Phase 10 added a PR-specific regression parser while leaving Phase 9's public benchmark claim gate unchanged. The new parser reuses only `EvidenceError` and `parse_benchmark_file` from the Phase 9 implementation.

**Rationale:** PR regression signaling is bidirectional and advisory, while the release claim gate controls public benchmark wording. Isolating their policies prevents a lightweight CI check from changing release-evidence semantics.
**Source:** 10-01-SUMMARY.md

---

### Evaluate Only Significant `sec/op` Slowdowns
The parser flags individual benchmark rows only when the candidate is at least 5% slower with a benchstat p-value below 0.05. It ignores geomeans, non-significant rows, faster rows, and every non-runtime metric section.

**Rationale:** Positive deltas in throughput or allocation tables do not mean slower execution, and per-row evaluation exposes fixture-specific cliffs that a geomean can hide.
**Source:** 10-01-PLAN.md

---

### Start Advisory and Preserve a One-Variable Blocking Flip
Detected regressions exit successfully by default. Setting `REQUIRE_NO_REGRESSION=true` changes flagged regressions to exit 1, while malformed evidence still fails closed and a missing baseline produces an explicit green advisory bypass.

**Rationale:** The project can observe the noise profile before making the check mandatory without redesigning the parser or workflow later.
**Source:** 10-01-PLAN.md

---

### Keep the Benchmark Subset in One Orchestrator
The exact Tier 1/2/3 families, twitter/canada fixtures, comparator set, five-run count, and ten-minute budget are encoded once in `run_pr_benchmark.sh` through one anchored benchmark regex.

**Rationale:** Workflow YAML stays focused on CI wiring, and benchmark scope cannot drift between the PR and main-baseline jobs.
**Source:** 10-02-SUMMARY.md

---

### Make the Workflow Authoritative for Baseline Availability
The shell orchestrator accepts exactly one of `--baseline` and `--no-baseline`; it does not inspect GitHub Actions cache state itself.

**Rationale:** The workflow already owns cache restoration and can interpret `cache-matched-key`, while the orchestrator remains deterministic and easy to exercise locally.
**Source:** 10-02-SUMMARY.md

---

### Separate Baseline Production from PR Consumption
Only the push-on-main workflow saves canonical `pr-bench-baseline-*` entries. The PR workflow uses the restore-only cache action and never writes the baseline namespace.

**Rationale:** A pull request must not be able to poison the comparison baseline used by later pull requests.
**Source:** 10-03-PLAN.md

---

### Prefer Safe Degradation for PR Reporting
The PR job uses `pull_request`, not `pull_request_target`, pins third-party actions to commit SHAs, always writes a step summary and uploads artifacts, and treats sticky-comment failure as non-fatal.

**Rationale:** Fork code must not receive elevated tokens; fork PRs can lose the comment surface while retaining benchmark evidence and the workflow summary.
**Source:** 10-03-PLAN.md

---

## Lessons

### Real benchstat Output Is Metric-Section Sensitive
Matching a positive percentage alone is insufficient because benchstat repeats benchmark rows across runtime, throughput, allocation, and native telemetry tables.

**Context:** The parser had to track the active metric header and evaluate only `sec/op`; otherwise an improved positive `B/s` delta could be reported as a runtime regression.
**Source:** 10-01-PLAN.md

---

### Repository Test Layout Determines the Valid unittest Invocation
The dotted-module form `python3 -m unittest tests.bench.test_*` fails because `tests/bench` is not a Python package.

**Context:** UAT used the standalone test files successfully, and the plans also preserve discovery-mode commands plus `unittest.main()` shims.
**Source:** 10-UAT.md

---

### Prefix Cache Restores Must Use `cache-matched-key`
The PR workflow cannot use the `cache-hit` output to decide whether a baseline was restored.

**Context:** Its exact key is deliberately `pr-bench-baseline-NEVER-MATCHES`, so a valid prefix restore has `cache-hit=false` even though it produced a usable baseline; `cache-matched-key` records the real result.
**Source:** 10-03-PLAN.md

---

### Cache Save and Restore Paths Must Match Exactly
The main workflow copies the captured head file to `baseline.bench.txt` before saving, and the PR workflow restores that same path.

**Context:** GitHub's cache action restores archived files to their saved paths; changing only the restore input does not rename `pr-bench-summary/head.bench.txt` into `baseline.bench.txt`.
**Source:** 10-03-PLAN.md

---

### Static Workflow Validation Cannot Prove Hosted Behavior
Parser tests, orchestrator tests, workflow contract tests, actionlint, and yq all passed, but cache visibility, sticky comments, cache-miss behavior, and concurrency cancellation still required live runs after merge.

**Context:** UAT first passed five local/static checks. The remaining three hosted Actions checks were completed after merge, bringing the final result to 8/8 passed.
**Source:** 10-UAT.md

---

### Review Feedback Should Become Executable Boundary Contracts
Follow-up work tightened empty-capture failure, metric-section parsing, success-only baseline saves, and stale-output replacement behavior.

**Context:** The project state records these review-driven fixes and focused regression tests after the initial Phase 10 implementation, showing that operational edge cases need explicit tests rather than comments alone.
**Source:** STATE.md

---

## Patterns

### Synthetic Boundaries Plus a Real-Format Fixture
Use small fixtures for threshold edges, p-value sentinels, mixed rows, malformed input, and metric sections, then add a byte-for-byte copy of real prior output.

**When to use:** Use when a text parser needs precise edge coverage without drifting away from the producer's actual output format.
**Source:** 10-01-SUMMARY.md

---

### Black-Box Regression Parser Contract
Expose regression analysis as a CLI that writes fixed-shape JSON and Markdown and communicates advisory, bypass, malformed-input, and blocking states through documented exit codes.

**When to use:** Use when shell scripts and CI workflows need stable behavior without depending on Python implementation details.
**Source:** 10-01-PLAN.md

---

### Script as Workflow Logic
Put benchmark selection, capture, comparison, and summary generation in a repository script, then make each workflow call that script with only mode-specific arguments.

**When to use:** Use when local runs and multiple CI workflows must share one exact execution path.
**Source:** 10-02-SUMMARY.md

---

### Atomic Staged Output Promotion
Build the complete evidence directory under `mktemp`, clean it on failure, and move it into place only after every required artifact exists.

**When to use:** Use for CI evidence bundles where stale or partial files could falsely imply that a failed run completed successfully.
**Source:** 10-02-SUMMARY.md

---

### PATH-Shadowed Integration Stubs
Exercise a shell orchestrator with temporary `go` and `benchstat` executables that emit controlled output.

**When to use:** Use when the orchestration contract needs fast deterministic tests but the real command is slow, statistical, or environment-sensitive.
**Source:** 10-02-PLAN.md

---

### Trusted Producer and Read-Only Consumer Cache
Create canonical benchmark baselines only from the default branch and let pull requests restore them through a read-only path.

**When to use:** Use when untrusted change sets need comparisons against shared CI state without gaining authority to modify that state.
**Source:** 10-03-PLAN.md

---

### Layered Advisory Reporting
Publish the same result through the job summary, a best-effort sticky PR comment, and retained diagnostic artifacts.

**When to use:** Use when one reporting channel may be unavailable, especially for fork pull requests with restricted token permissions.
**Source:** 10-03-PLAN.md

---

## Surprises

### A Uniform Regex Harmlessly Overmatched a Missing Tier 3 Case
The shared benchmark regex includes a Tier 3 canada combination even though no such benchmark function exists.

**Impact:** Go silently skips the nonexistent function, so the simpler single regex was retained without producing an extra benchmark row.
**Source:** 10-02-SUMMARY.md

---

### The First UAT Commands Failed Before Running Tests
Both dotted-module unittest commands failed because the test directories lack package markers.

**Impact:** UAT switched to direct test-file execution; all 17 parser tests and all 4 orchestrator tests then passed.
**Source:** 10-UAT.md

---

### Phase Shipping Preceded Full Live Verification
Project state records Phase 10 as shipped, while UAT still lists three blocked hosted-Actions checks and the live-verification artifact says `10-03-SUMMARY.md` should be created only after those checks pass.

**Impact:** The implementation shipped before hosted evidence was complete. Later UAT verified cache-hit/cache-miss behavior, sticky comments, and concurrency cancellation, but `10-03-SUMMARY.md` was never added, leaving a tracking-only gap.
**Source:** 10-UAT.md; 10-03-LIVE-VERIFICATION.md; STATE.md

---

### Review Found More Operational Edges After Exact Plan Execution
The first two summaries report no deviations or issues, yet later review follow-ups tightened empty output, cache-save conditions, metric parsing, and stale output replacement.

**Impact:** The phase gained stronger failure contracts after initial execution, and future CI features should include review-driven operational scenarios in their first test matrix where possible.
**Source:** 10-01-SUMMARY.md; 10-02-SUMMARY.md; STATE.md
