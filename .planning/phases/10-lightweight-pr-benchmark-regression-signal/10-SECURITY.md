---
phase: 10
slug: lightweight-pr-benchmark-regression-signal
status: blocked
threats_open: 1
asvs_level: 1
created: 2026-04-27
---

# Phase 10 - Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| benchstat text -> parser | `scripts/bench/check_pr_regression.py` reads generated benchstat output and emits JSON/Markdown advisory results. | Benchmark timing evidence; no secrets. |
| `--baseline` path -> orchestrator | `scripts/bench/run_pr_benchmark.sh` accepts a restored baseline path from the workflow and copies it into a staged output directory. | Benchmark baseline file; filesystem path. |
| `go test -bench` stdout -> staged evidence | PR code influences benchmark values, while Go's benchmark harness owns the output format. | Raw benchmark evidence. |
| staged directory -> published summary directory | Orchestrator builds evidence under `mktemp -d` and promotes it to `pr-bench-summary/` only after required outputs are produced. | Local artifact files. |
| `pull_request` event -> workflow runner | PR workflow executes untrusted PR code with limited fork-token permissions and best-effort comment posting. | Repository checkout, benchmark execution, GitHub token scoped by event. |
| third-party action -> workflow runner | External actions execute in the runner and are pinned to immutable commit SHAs. | Runner environment and workflow token. |
| main baseline cache -> PR workflow | Main workflow saves `baseline.bench.txt`; PR workflow restores the same path read-only for comparison. | Cross-workflow benchmark baseline. |
| sticky comment action -> pull request comments API | Comment posting needs `pull-requests: write` and may be denied for fork PRs. | Markdown summary; PR comment API call. |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-10-01 | T (Tampering) | Benchstat parser | mitigate | Parser is regex-only, reads input via `Path.read_text`, and has no `eval`, subprocess, shell-out, or OS command execution path. Evidence: `scripts/bench/check_pr_regression.py`. | closed |
| T-10-02 | D (DoS) | Benchstat parser | mitigate | Malformed or empty benchstat input raises `EvidenceError` and exits non-zero, so parser failures fail closed. Evidence: `test_malformed_input_fails_closed`, `test_empty_input_fails_closed`. | closed |
| T-10-03 | T (Tampering) | Benchstat parser | mitigate | Only `sec/op` sections are evaluated; `B/s`, allocation, and native-memory metric tables are ignored to prevent faster throughput rows from becoming false regressions. Evidence: `METRIC_HEADER_RE` and `test_metric_sections_only_sec_op_flags`. | closed |
| T-10-04 | T (Tampering) | Regression policy | mitigate | Advisory mode exits 0 by default, while the future blocking flip is isolated to `REQUIRE_NO_REGRESSION=true` and is covered by tests. Evidence: workflow env, CHANGELOG, parser constant. | closed |
| T-10-05 | T (Tampering) | Orchestrator baseline evidence | accept | Benchstat is the trust anchor for comparing raw `.bench.txt` files; a malicious PR cannot directly author benchstat output without replacing the trusted tool. | closed |
| T-10-06 | T (Tampering) | PR orchestrator cache behavior | mitigate | Orchestrator never writes actions/cache; it only consumes an explicit `--baseline` path or `--no-baseline`. Evidence: no `actions/cache` reference in `scripts/bench/run_pr_benchmark.sh`. | closed |
| T-10-07 | I (Information Disclosure) | Shell and workflow logs | mitigate | Scripts use `set -euo pipefail` without `set -x`, do not print env, and do not source `.env` files. Workflow inline commands avoid secret logging. | closed |
| T-10-08 | D (DoS) | Benchmark runtime | mitigate | Orchestrator uses `go test ... -timeout 600s`; workflows also set `timeout-minutes: 15` and PR concurrency cancels superseded pushes. | closed |
| T-10-09 | E (Elevation of Privilege) | Shell path handling | mitigate | Orchestrator double-quotes variable-expanded paths, does not use `eval`, and validates missing baseline paths before use. | closed |
| T-10-10 | E (Elevation of Privilege) | PR workflow trigger | mitigate | PR workflow uses `pull_request`, not `pull_request_target`, preserving fork-token restrictions and avoiding base-repo secret exposure. | closed |
| T-10-11 | T (Tampering) | Rolling baseline cache | mitigate | PR workflow uses `actions/cache/restore` only; main workflow is the only workflow using `actions/cache/save`, and both save/restore the same `baseline.bench.txt` path. | closed |
| T-10-12 | T (Tampering) | Workflow semantic correctness | mitigate | Local lint and parsing pass, and a live verification checklist exists, but post-merge cache-hit, cache-miss, sticky-comment, and concurrency behavior have not been recorded yet. Evidence required: complete `10-03-LIVE-VERIFICATION.md`. | open |

*Status: open - closed*
*Disposition: mitigate (implementation required) - accept (documented risk) - transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-10-01 | T-10-05 | Benchstat is treated as the trusted statistical comparator. Subverting it requires compromising the external benchmark tool itself, which is outside Phase 10 scope. | GSD security audit | 2026-04-27 |

---

## Verification Evidence

| Check | Result |
|-------|--------|
| `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v` | passed, 17 tests |
| `python3 -m unittest discover -s tests/bench -p "test_run_pr_benchmark.py" -v` | passed, 3 tests |
| `bash -n scripts/bench/run_pr_benchmark.sh` | passed |
| `wc -l scripts/bench/run_pr_benchmark.sh` | 142 lines |
| `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml` | passed |
| YAML parse for both workflow files | passed |
| External workflow actions pinned to 40-character SHAs | passed |
| `go test -run=^$ -bench=^$ ./...` | passed |
| `grep -nE "REQUIRE_NO_REGRESSION" .github/workflows/pr-benchmark.yml CHANGELOG.md scripts/bench/check_pr_regression.py` | exactly 3 matches |
| `python3 -m unittest discover -s tests/bench -v` | failed in pre-existing Phase 9 docs contract: README does not contain `Phase 09.1` |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-27 | 12 | 11 | 1 | Codex |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [ ] `threats_open: 0` confirmed
- [ ] `status: verified` set in frontmatter

**Approval:** blocked pending live workflow verification.
