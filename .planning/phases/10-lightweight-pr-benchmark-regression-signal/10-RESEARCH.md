# Phase 10: Lightweight PR benchmark regression signal — Research

**Researched:** 2026-04-27
**Domain:** GitHub Actions CI workflow + Go benchstat regression parser
**Confidence:** HIGH (all 21 D-decisions reconcile with verified external behavior)

## Summary

Phase 10 wires up two new GitHub Actions workflows (a `pull_request` regression check + a `push: main` baseline-capture sibling) plus a small Python regression parser that consumes `benchstat` output. CONTEXT.md's 21 locked decisions hold up under verification: `actions/cache@v4` prefix matching does pick "most recently created," PR jobs can restore caches written by base-branch push workflows, the 7-day LRU eviction is current behavior, `marocchino/sticky-pull-request-comment` is healthily maintained at v3.0.4 (released 10 days before research), and benchstat's `~ (p=…)` / `−XX.XX% (p=… n=…)` notation matches what the in-repo Phase 9 evidence already contains. **No D-decision needs to be revised**, but two locked decisions deserve sharper planner guidance:

1. **D-07 cache key strategy** — for the PR job to read the main-branch cache, the push-on-main workflow MUST run on `branches: [main]` so the cache lives in the base-branch (= default-branch) scope; otherwise GitHub's cache scoping isolates PR-merge-ref caches from main-branch caches and the PR job will always cache-miss.
2. **D-17 paths-ignore expression for `.github/workflows/**`** — `paths-ignore` does NOT support negation (`!path`); the only way to "ignore everything in `.github/workflows/` except the new pr-bench workflow file" is to use `paths` (positive) with negation, which contradicts D-16 ("paths-ignore, not paths"). The correct resolution is to drop the negation: ignore all of `.github/workflows/**` AND let the PR-bench workflow file's own `on.pull_request.paths` trigger handle self-edits as a separate question (or accept that edits to it land via a normal PR like any other workflow). Recommendation in §2.

**Primary recommendation:** Plan two thin workflows + one ≤200-LOC bash orchestrator + one new Python parser (`scripts/bench/check_pr_regression.py`) that imports `parse_benchmark_file()` and `SIGNIFICANT_WIN_RE` semantics from `scripts/bench/check_benchmark_claims.py` rather than duplicating them. Pin `benchstat` to the same `@latest` Phase 9 currently uses (no Phase 9 pin to inherit, so use `@latest` for now and document the version actually installed in `summary.json`). Keep all artifact-write paths gated by `if: always()` per D-21.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| PR trigger + skip filter | GitHub Actions YAML (`on.pull_request.paths-ignore`) | — | Only the workflow file itself can express skip semantics |
| Native + Go build | Composite action (`./.github/actions/setup-rust`) | `actions/setup-go` | Reuse existing pinned toolchain |
| Bench execution | Bash orchestrator (`scripts/bench/run_pr_benchmark.sh`) | `go test -bench` | Shell composes Go bench + benchstat invocations; Go test owns measurement |
| Regression parse | Python (`scripts/bench/check_pr_regression.py`) | — | Same language as Phase 9 claim gate; reuses helpers |
| Baseline transport | `actions/cache@v4` | Workflow artifacts (diagnostic only) | Cache is the canonical baseline path; artifacts are supplementary evidence |
| PR surface | `$GITHUB_STEP_SUMMARY` (always) | `marocchino/sticky-pull-request-comment` (best-effort) | Step summary works on forks; sticky comment requires `pull-requests: write` |
| Future blocking-flip control | Workflow-file constant or env var | — | D-14 — must be a single grep-able knob |

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Total wall-clock budget for the PR job is `≤ 10 minutes` on `ubuntu-latest`. Budget overruns are a planning bug.
- **D-02:** Benchmark families on PR: `BenchmarkTier1FullParse_*`, `BenchmarkTier2Typed_*`, `BenchmarkTier3SelectivePlaceholder_*`. Tier 1 diagnostics + cold/warm are out.
- **D-03:** Fixtures: `twitter.json` and `canada.json` only. `citm_catalog.json` dropped (size).
- **D-04:** `-count=5`. Lower counts produce false positives.
- **D-05:** Comparator filter: only `pure-simdjson` and `encoding-json-any` / `encoding-json-struct` rows. `-bench` regex must omit `minio`, `sonic`, `goccy` rows.
- **D-06:** Rolling main baseline via `actions/cache`. Push-on-main workflow writes; PR workflow reads.
- **D-07:** Write key `pr-bench-baseline-${{ github.sha }}`; restore-keys prefix `pr-bench-baseline-`. Old entries evict via 7-day LRU.
- **D-08:** Cache miss = explicit "no baseline available — advisory bypass" notice, exit 0, upload own artifact so next push fills cache.
- **D-09:** Baseline staleness acceptable: rolling latest main is the comparison target. PR job does NOT recapture base in same run.
- **D-10:** Public release-scoped evidence (`testdata/benchmark-results/v<x>/`) is NOT used as PR baseline.
- **D-11:** Regression definition (per row): `p < 0.05` AND candidate median `≥ 5%` slower than baseline median. Both required.
- **D-12:** Granularity is per row, not per-tier-geomean. Every flagged row listed individually.
- **D-13:** Advisory-only initially: PR workflow exits `0` on regressions; surfaced via comment + step summary only. Not a required status check.
- **D-14:** Graduating to blocking is out of scope. Must leave a clearly named control surface.
- **D-15:** Regression rows cite exact bench row name, baseline median (time/op + B/op), candidate median, absolute + percentage delta, p-value.
- **D-16:** Use `paths-ignore` (not `paths`) so mixed PRs still run.
- **D-17:** Skip path globs: `**.md`, `docs/**`, `LICENSE`, `NOTICE`, `.planning/**`, `.github/workflows/**` *except* the new PR-bench workflow file, `.github/actions/**`, `testdata/benchmark-results/**`.
- **D-18:** Risk acknowledged for skipping `.github/actions/**`.
- **D-19:** Surface order: `$GITHUB_STEP_SUMMARY` always; sticky PR comment best-effort. On fork token deny: log notice, skip comment, post step summary.
- **D-20:** Concurrency `group: pr-bench-${{ github.event.pull_request.number }}`, `cancel-in-progress: true`. Push workflow uses different group.
- **D-21:** Workflow artifact upload (raw `.bench.txt`, `benchstat.txt`, `summary.json`) runs `if: always()`. PR retention: 14 days.

### Claude's Discretion

- Exact workflow filenames (`pr-benchmark.yml`, `main-benchmark-baseline.yml` or planner's choice).
- Exact `summary.json` shape (must include flagged rows, target metadata, threshold values, `regression: true|false`).
- Exact sticky PR comment markdown layout.
- Exact name of the future "blocking flip" control knob (D-14).
- Share Python with `check_benchmark_claims.py` (extend) vs. separate `check_pr_regression.py`. Phase 9's gate has different semantics; new script is acceptable as long as it reuses `parse_benchmark_file()` and benchstat helpers.
- `actions/cache@v4` directly vs. `actions/cache/restore@v4` + `actions/cache/save@v4` for finer miss handling.
- Install `benchstat` from `golang.org/x/perf/cmd/benchstat@latest` vs. pinning to Phase 9's version (Phase 9 uses `@latest`, so `@latest` is the inherited choice; planner may choose to pin a SHA).

### Deferred Ideas (OUT OF SCOPE)

- Flipping the PR regression check to required/blocking.
- Tier 1 diagnostics + cold/warm coverage on PR.
- `citm_catalog.json` fixture on PR.
- Cross-comparator regression detection (sonic, minio, goccy).
- Multi-platform PR benchmarking (darwin, windows, linux/arm64).
- Same-run base + head capture.
- `benchmark-action/github-action-benchmark` for gh-pages history.
- Detecting regressions on `main` itself (silent absorbsion).
- `bench-strict` PR label to opt-in to blocking.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| (none mapped) | No REQ-IDs in REQUIREMENTS.md or ROADMAP.md map to Phase 10. | The 21 D-decisions in CONTEXT.md are the binding contract. Phase 10 inherits the BENCH-01..07 contract from Phase 7 implicitly: any PR-time regression the gate flags must be a regression against the Tier 1/2/3 harness Phase 7 already established. No new requirement IDs are introduced. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `actions/checkout` | pinned SHA `11bd71901bbe5b1630ceea73d27597364c9af683` | Repo checkout w/ submodules | Already in `benchmark-capture.yml`; reuse SHA for consistency |
| `actions/setup-go` | pinned SHA `40f1582b2485089dde7abd97c1529aa768e1baff` | Go toolchain via `go-version-file: go.mod` | Already in `benchmark-capture.yml` |
| `./.github/actions/setup-rust` | composite (in-repo) | Pinned Rust toolchain via `rust-toolchain.toml` | Reuse — no edits needed |
| `actions/cache` | `v4` (current major) | Baseline transport between push-on-main and PR jobs | Standard caching primitive; supports `restore-keys` prefix matching with most-recent-created semantics [VERIFIED: GitHub Docs] |
| `actions/upload-artifact` | pinned SHA `ea165f8d65b6e75b540449e92b4886f43607fa02` | Diagnostic raw bench artifacts | Already in `benchmark-capture.yml` |
| `marocchino/sticky-pull-request-comment` | `v3.0.4` (released 2026-04-10) | Idempotent PR comment editing | Actively maintained, recent release [VERIFIED: GitHub release page]; v2.9.x line still receives dependabot bumps but v3 is current major |
| `golang.org/x/perf/cmd/benchstat` | `@latest` (Phase 9 inherits this) | Statistical comparison of two `.bench.txt` files | Canonical Go benchmark stats tool [VERIFIED: pkg.go.dev] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `python3` (stdlib only) | `>= 3.11` (matches Phase 9 toolchain) | Regression parser | New `scripts/bench/check_pr_regression.py` — stdlib only, no `pip install` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `actions/cache@v4` | `actions/cache/restore@v4` + `actions/cache/save@v4` | Finer control: can `save` only on a specific condition. Locked-decision-compatible (D-Discretion). Recommend for the **push-on-main** baseline workflow because it should always save; the **PR** workflow only ever restores, so plain `cache` action with `save: false`-equivalent (or `cache/restore@v4` only) is cleaner. |
| `marocchino/sticky-pull-request-comment` v3 | v2.9.3 (still maintained on v2 line) | v3 is current major and receives active development. Pin to v3.0.4 SHA for supply-chain safety; v2 acceptable if planner wants to mirror what other internal repos use. |
| `benchstat@latest` | Pin to a specific SHA of `golang.org/x/perf` | Reproducibility vs. inertia. Phase 9's `benchmark-capture.yml` uses `@latest`; PR job should match (not introduce drift). Document the resolved version in `summary.json`. |

**Installation in workflow:**
```yaml
- name: Install benchstat
  run: |
    set -euo pipefail
    go install golang.org/x/perf/cmd/benchstat@latest
    echo "$(go env GOPATH)/bin" >>"$GITHUB_PATH"
```

**Version verification (run during planning, not committed):** `npm view`-equivalent for Go is `go list -m -versions golang.org/x/perf` — but Phase 9 already locked the operational choice. No version bump needed for Phase 10.

## Architecture Patterns

### System Architecture Diagram

```
                     ┌────────────────────────────────────────┐
                     │  push: branches: [main]                │
                     │  main-benchmark-baseline.yml           │
                     │                                        │
                     │  checkout → setup-go → setup-rust →    │
                     │  cargo build --release → install       │
                     │  benchstat → run subset (D-02..D-05)   │
                     │  → write to actions/cache key          │
                     │  pr-bench-baseline-<sha>               │
                     └────────────┬───────────────────────────┘
                                  │  (cache scope: refs/heads/main = default branch)
                                  │  visible to PR jobs as base-branch fallback
                                  ▼
                     ┌────────────────────────────────────────┐
                     │  pull_request: paths-ignore: [...]     │
                     │  pr-benchmark.yml                      │
                     │                                        │
                     │  concurrency: pr-bench-<PR#>           │
                     │  cancel-in-progress: true              │
                     │                                        │
                     │  checkout (head SHA) → setup-go →      │
                     │  setup-rust → cargo build --release →  │
                     │  install benchstat → restore cache     │
                     │     restore-keys: pr-bench-baseline-   │
                     │                                        │
                     │  ┌───── cache hit? ─────┐              │
                     │  │ YES: have baseline   │              │
                     │  │ .bench.txt           │              │
                     │  │                      │              │
                     │  │ NO:  bypass mode     │              │
                     │  │ (D-08)               │              │
                     │  └─────────┬────────────┘              │
                     │            ▼                           │
                     │  scripts/bench/run_pr_benchmark.sh     │
                     │   → go test -bench '...' -count=5      │
                     │     -run '^$' -benchmem  > head.bench  │
                     │   → if cache hit:                      │
                     │       run_benchstat.sh --old baseline  │
                     │                       --new head       │
                     │       > regression.benchstat           │
                     │   → check_pr_regression.py             │
                     │       → summary.json (regression flag) │
                     │       → markdown.fragment (per-row)    │
                     │                                        │
                     │  Surface:                              │
                     │   1. echo markdown.fragment >>          │
                     │      $GITHUB_STEP_SUMMARY (always)     │
                     │   2. sticky-pull-request-comment       │
                     │      (best-effort; skip on 403)        │
                     │   3. upload-artifact (if: always())    │
                     │      retention: 14 days                │
                     │                                        │
                     │  Exit 0 always (D-13 advisory).        │
                     │  Future: env var REQUIRE_NO_REGRESSION │
                     │  flips this to exit 1 on regression.   │
                     └────────────────────────────────────────┘
```

### Recommended Project Structure
```
.github/
├── actions/
│   └── setup-rust/                  # existing, reuse unchanged
└── workflows/
    ├── benchmark-capture.yml        # existing Phase 9 (DO NOT EDIT)
    ├── pr-benchmark.yml             # NEW Phase 10
    └── main-benchmark-baseline.yml  # NEW Phase 10

scripts/bench/
├── capture_release_snapshot.sh      # existing Phase 9
├── check_benchmark_claims.py        # existing — IMPORT helpers from this
├── run_benchstat.sh                 # existing — INVOKE this
├── check_pr_regression.py           # NEW Phase 10 (regression parser)
└── run_pr_benchmark.sh              # NEW Phase 10 (≤200 LOC orchestrator)

tests/bench/
├── test_check_benchmark_claims.py   # existing — MIRROR style
└── test_check_pr_regression.py      # NEW Phase 10
```

### Pattern 1: Workflow-as-trigger, script-as-logic
**What:** YAML workflow files compose orchestrator scripts; logic lives in `scripts/bench/*` so it is testable locally without GitHub Actions.
**When to use:** Always — Phase 9 set this precedent (`capture_release_snapshot.sh` + `check_benchmark_claims.py`).
**Example:** Phase 9's `benchmark-capture.yml` is 50 lines of YAML; the work happens in `capture_release_snapshot.sh` + `check_benchmark_claims.py`. Phase 10 mirrors this exactly.

### Pattern 2: Cache-as-baseline-transport
**What:** Use `actions/cache@v4` to ferry artifacts between two workflows that cannot otherwise share state (push-on-main and pull_request).
**When to use:** When the two workflows run on different events but need to compare outputs. Note: cache scoping rules dictate that the **producer** workflow MUST run on the **base branch** (default branch = main) so its cache is in the consumer's restore hierarchy.
**Example:**
```yaml
# Producer (main-benchmark-baseline.yml)
on:
  push:
    branches: [main]
    paths-ignore: [<same set as PR workflow>]
- uses: actions/cache/save@v4
  with:
    path: baseline.bench.txt
    key: pr-bench-baseline-${{ github.sha }}

# Consumer (pr-benchmark.yml)
on:
  pull_request:
    paths-ignore: [...]
- id: restore
  uses: actions/cache/restore@v4
  with:
    path: baseline.bench.txt
    key: pr-bench-baseline-DOES-NOT-EXIST  # force restore-keys path
    restore-keys: |
      pr-bench-baseline-
- name: Note baseline status
  run: |
    if [[ "${{ steps.restore.outputs.cache-matched-key }}" == "" ]]; then
      echo "no_baseline=true" >> "$GITHUB_OUTPUT"
    else
      echo "no_baseline=false" >> "$GITHUB_OUTPUT"
      echo "baseline_key=${{ steps.restore.outputs.cache-matched-key }}" >> "$GITHUB_OUTPUT"
    fi
```
Source: `[VERIFIED: docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching]` and `[VERIFIED: github.com/actions/cache#match-an-existing-cache]`.

### Pattern 3: Step-summary-first, sticky-comment-best-effort
**What:** Always write to `$GITHUB_STEP_SUMMARY` (works on every event including forks); attempt sticky comment second, swallow 403 errors.
**When to use:** Any workflow that may run on fork PRs (which `pull_request` does — `pull_request_target` is the only "with-write-token" alternative, and CONTEXT.md explicitly avoids it because executing fork code with write tokens is the well-known supply-chain-attack vector).
**Example:**
```yaml
- name: Write step summary (always works)
  if: always()
  run: cat summary.md >> "$GITHUB_STEP_SUMMARY"

- name: Post sticky comment (best-effort)
  if: always()
  continue-on-error: true        # swallow 403 from forks
  uses: marocchino/sticky-pull-request-comment@v3.0.4
  with:
    header: pr-benchmark-regression
    path: summary.md
```
Source: `[CITED: github.com/marocchino/sticky-pull-request-comment#permissions]` — `pull-requests: write` is required, which is denied on fork-originated `pull_request` (not `pull_request_target`) runs by GitHub's standard token policy.

### Pattern 4: Bidirectional benchstat regex (faster OR slower)
**What:** Phase 9's `SIGNIFICANT_WIN_RE = re.compile(r"(?<![\w.])-\d+(?:\.\d+)?%")` matches only **negative** deltas (faster). Phase 10 needs to detect **positive** deltas ≥ 5% (slower). The new regex is symmetric: `r"(?<![\w.])([-+]?)\d+(?:\.\d+)?%"` and the parser captures the sign.
**When to use:** Phase 10 regression parser ONLY — Phase 9 keeps its asymmetric "win" semantics.
**Example:**
```python
import re
DELTA_RE = re.compile(r"(?<![\w.])([+-])(\d+(?:\.\d+)?)%\s+\(p=(\d+\.\d+)\s+n=\d+\)")
# Matches "+3.15% (p=0.002 n=10)" and "-94.80% (p=0.000 n=10)"
# Does NOT match "~ (p=0.075 n=10)" — caller must check for "~" first.
```
Verified against `testdata/benchmark-results/v0.1.2/phase9.benchstat.txt` rows like:
```
Tier1FullParse_twitter_json/encoding-json-struct-4   4.842m ± 1%   4.994m ± 1%   +3.15% (p=0.002 n=10)
Tier1FullParse_canada_json/minio-simdjson-go-4       22.89m ± 1%  22.67m ± 1%        ~ (p=0.143 n=10)
Tier1FullParse_twitter_json/pure-simdjson-4          39.339m ± 1%  2.044m ± 2%  -94.80% (p=0.000 n=10)
```

### Pattern 5: Per-row regression rule (D-11/D-12)
**What:** A row is a regression when ALL of:
1. The benchstat output line for that row contains a positive delta (`+X.XX%`)
2. The numeric percentage `≥ 5.0`
3. The p-value `< 0.05` (i.e., the `~` sentinel is NOT present on that line)
**Example:**
```python
def is_regression(line: str, threshold_pct: float = 5.0, p_max: float = 0.05) -> bool:
    if "~" in line:
        return False  # p > 0.05 → not significant
    match = DELTA_RE.search(line)
    if not match:
        return False  # no delta on this line (e.g., header row, geomean row)
    sign, pct_str, p_str = match.groups()
    if sign != "+":
        return False  # faster, not slower
    return float(pct_str) >= threshold_pct and float(p_str) < p_max
```
Note: benchstat's own significance test already gates on `p < 0.05` — when the test says non-significant, benchstat replaces the delta with `~`. So the `"~" in line` check is sufficient, and the explicit `p < 0.05` check is belt-and-suspenders. Keep both for clarity.

### Anti-Patterns to Avoid

- **Recapturing the baseline in the PR job:** Doubles runtime, blows the 10-min budget. (D-09 explicitly rejects this.)
- **Per-tier geomean regression checks:** Hides per-fixture cliffs. (D-12 explicitly rejects this.)
- **Using `pull_request_target` to get write tokens on fork PRs:** Documented supply-chain hazard — runs the PR's code with the base repo's write tokens. CONTEXT.md correctly chose `pull_request` and accepts that fork-PR sticky-comments will fail; the step summary covers fork visibility.
- **`paths` (positive) instead of `paths-ignore`:** A docs-only PR that also touches one go file would NOT skip with `paths-ignore` (correct: PR runs because the go file isn't ignored). With `paths`, a docs-only PR is silently skipped — but a docs+code PR would run only if the code file matches `paths`, which is brittle. Stick with `paths-ignore` per D-16.
- **Negation in `paths-ignore`:** GitHub Actions does NOT support `!` negation in `paths-ignore` (it does in `paths`). D-17's "`.github/workflows/**` *except* the new PR-bench workflow file itself" is therefore not directly expressible. See §"D-17 paths-ignore expressibility" below.
- **Writing the cache from the PR workflow:** PR-merge-ref caches are scoped to the PR and cannot be restored by other PRs. Only the push-on-main workflow writes the canonical `pr-bench-baseline-*` cache. (PR jobs MAY upload artifacts for diagnostics per D-08.)

### D-17 paths-ignore expressibility (CRITICAL planner note)

CONTEXT.md D-17 lists:
> `.github/workflows/**` *except* the new PR-bench workflow file itself

GitHub Actions `paths-ignore` does NOT support `!` negation. The only ways to honor this:

1. **Drop the exception** — ignore all of `.github/workflows/**`. Edits to the PR-bench workflow file land via a normal PR like any other workflow change; the workflow won't run on its own definition-edit PR, but it'll run on the very next non-workflow-edit PR. Acceptable risk: workflow-only PRs don't get a perf check, but workflow-only PRs are rare and reviewer-driven.
2. **Switch to `paths` with negation** — `paths: ['!**.md', '!docs/**', ..., '.github/workflows/pr-benchmark.yml']`. This is a `paths` filter, contradicting D-16. The interaction is subtle: with `paths`, a workflow runs only if changed files match at least one positive pattern OR no negation excludes them. Using only negations (`!`) is a common trick to invert paths-ignore but is fragile.
3. **Accept that workflow-edits still trigger the bench** — if D-17's "except this file" was meant to ensure self-edits ARE checked, then the paths-ignore set should NOT include `.github/workflows/**` at all (i.e., remove that line); ANY workflow edit triggers the bench, which is fine but slightly noisy.

**Recommended resolution:** Option 1 (drop the exception). Reason: D-18 already accepts the symmetric risk for `.github/actions/**` ("composite-action edits could affect builds; acceptable tradeoff"). Edits to `pr-benchmark.yml` itself are similar — if a PR edits the bench workflow, the bench job won't run on that exact PR, but reviewers will manually `workflow_dispatch` if they need a smoke. This avoids the YAML-syntax cliff and keeps D-16's "paths-ignore" choice intact.

If the user disagrees, this is a planning-time decision the planner should surface, not silently choose.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bidirectional benchstat parsing | Custom shell `awk`/`sed` text munging | Python `re` over raw benchstat output | Phase 9 LEARNINGS specifically calls out "Claim gates need to understand real benchstat output, not idealized fixtures" — text munging is the trap |
| Statistical significance | Hand-rolled t-test on raw `.bench.txt` | `benchstat`'s built-in U-test | Mann-Whitney U-test is what benchstat applies by default; reimplementing it = bug surface |
| Sticky PR comment | Custom `gh api` curl POST + comment-id tracking | `marocchino/sticky-pull-request-comment@v3.0.4` | Action handles "find existing comment by header → edit OR create"; rolling this in shell is ~50 lines of brittle JSON-poking |
| Cache key generation | Manual SHA1 of (sha + date + os) | `actions/cache@v4` with `${{ github.sha }}` directly | The action handles miss-then-restore-keys-prefix-search; manual key construction adds drift risk |
| Step summary markdown | Building HTML or custom format | Plain markdown + GitHub-flavored tables + `<details>` | Up to 1MB rendered as markdown [VERIFIED: GitHub Docs]; tables and collapsed sections both supported |

**Key insight:** Every reusable piece in Phase 10 already exists in-repo (Phase 9 helpers) or as a maintained action. Phase 10 is glue, not invention.

## Runtime State Inventory

> Phase 10 is a CI/workflow phase, not a rename/migration. This section is included for transparency.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by inspecting `.planning/`, `internal/`, and `testdata/` for "pr-bench" / "pr-benchmark" references (zero hits) | None |
| Live service config | The cache namespace `pr-bench-baseline-*` will be created at first push-on-main run after Phase 10 ships. No external service has the string today. | None — namespace is created by first run |
| OS-registered state | None — no Task Scheduler / launchd / systemd registrations | None |
| Secrets/env vars | `GITHUB_TOKEN` is auto-provided by Actions; no new secrets. The future blocking-flip control knob (D-14) should be a workflow-file constant or env, NOT a secret | None |
| Build artifacts | None | None |

**Nothing found in any category** — verified by `grep -rE "pr-bench|pr-benchmark|pr_bench" --exclude-dir=.git --exclude-dir=node_modules` (zero hits in committed files).

## Common Pitfalls

### Pitfall 1: Cache scope mismatch — push-on-main writes, PR can't restore
**What goes wrong:** Push-on-main workflow runs but its cache is invisible to PR jobs, so every PR cache-misses forever.
**Why it happens:** GitHub Actions caches are scoped to the branch/ref where they were created. PR jobs can restore from current branch + base branch + default branch ONLY. If the producer workflow somehow runs on a non-default branch (e.g., `branches: [main, release/*]`), the `release/*` runs create caches that PR jobs can't see.
**How to avoid:** Push-on-main workflow trigger must be `on: push: branches: [main]` exactly — no other branches. Default-branch caches are visible to all PR jobs.
**Warning signs:** First-week dashboards show 0% cache-hit rate even though push-on-main has run multiple times. Verify by checking the "Cache" tab in the repo's Actions settings — entries should be tagged `refs/heads/main`.
Source: `[VERIFIED: docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching]` ("PR workflows can restore caches from the base branch")

### Pitfall 2: benchstat row-name format drift
**What goes wrong:** The Phase 9 `SIGNIFICANT_WIN_RE` matches `Tier1FullParse_twitter_json/pure-simdjson-4` rows but ALSO `geomean` rows and the comparison table header. Naive parsing reports a "regression" on the geomean row or panics on the header.
**Why it happens:** benchstat's table output prefixes data rows with the benchmark name BUT also emits `geomean` rows and a separator/header line. The 09-LEARNINGS document explicitly logs this: "Claim gates need to understand real benchstat output, not idealized fixtures."
**How to avoid:** Match only lines that begin with `Tier(1FullParse|2Typed|3SelectivePlaceholder)_` (after the leading whitespace strip) AND contain a `(p=...)` annotation. Reject rows where the row name is `geomean` or `¹ summaries...`.
**Warning signs:** Test fixture with all-significant deltas reports MORE flagged rows than there are benchmark rows (it's flagging geomean too).

### Pitfall 3: Native build cost eats the budget
**What goes wrong:** `cargo build --release` on `ubuntu-latest` cold takes ~3-4 minutes. Combined with checkout (1 min), Go setup (30s), Rust setup (30s), benchstat install (15s), and the actual benchmark run (3-5 min for the D-02..D-05 subset at count=5), total approaches 10 min.
**Why it happens:** simdjson is a ~1MB single-file C++ amalgamation; release-mode builds are not fast. No pre-built binary exists for hosted-runner consumption (PLAT-01 ships shared libs but the bench needs the same-tree-build).
**How to avoid:** Cache `target/release/` keyed on `Cargo.lock` + `third_party/simdjson` submodule SHA. This is a separate cache from the baseline cache (D-06..D-09); both can coexist. The Rust build cache hits in subsequent runs reduce build to ~30s.
**Warning signs:** First-run timing > 10 min; subsequent runs comfortably under budget. The first-run overshoot is acceptable per D-08's "no baseline available" graceful path, but a Rust-build-cache miss on every PR is the silent budget killer. Plan to add `actions/cache@v4` for `target/release/`.

### Pitfall 4: `cancel-in-progress` race with cache writes
**What goes wrong:** A second push to the same PR cancels the first run mid-benchmark. The first run's `actions/upload-artifact@v4` for diagnostic evidence is interrupted; nothing saved.
**Why it happens:** D-20 sets `cancel-in-progress: true`. GitHub kills the runner; `if: always()` runs but only if the cancellation reaches the upload step before the runner shuts down — race-prone on fast-cancel scenarios.
**How to avoid:** Accept the race for the diagnostic artifact (it's diagnostic-only — the next run will produce its own). Do NOT put the cache-WRITE in the PR workflow at all (CONTEXT.md correctly only writes from push-on-main, where `cancel-in-progress: false` per D-20).
**Warning signs:** Empty artifact uploads or 404 on artifact retrieval after rapid PR pushes. This is expected, not a bug — the next run produces fresh evidence.

### Pitfall 5: `~` (non-significant) misread as the row being absent
**What goes wrong:** Phase 9's `has_significant_win` raises `EvidenceError` if a required row is "not found"; for the PR regression case, a row showing `~` IS found but is non-significant. Confusing the two leads to false-positive alarms.
**Why it happens:** Phase 9 needs every row to exist AND be significant; Phase 10 needs every row to exist but only flags the SLOWER-AND-SIGNIFICANT subset.
**How to avoid:** In `check_pr_regression.py`, separate "row missing" (parse error → exit 1, blocks the gate) from "row present, `~` sentinel" (skip the row — not a regression). This is the asymmetry the new parser must own.

### Pitfall 6: Step-summary 1MB cap with verbose benchstat output
**What goes wrong:** The full benchstat output for D-02..D-05 with `count=5` is ~3KB; well under the 1MB cap. But if a future regression includes the raw `.bench.txt` AND the benchstat AND the per-row delta table, sizes grow. Edge case: the workflow runs against an unusual baseline that explodes row count.
**Why it happens:** No active risk for the locked subset; flagged here so the planner's `summary.json` shape doesn't accidentally embed the full raw bench.
**How to avoid:** Step summary contains: per-row regression table (≤6 rows for the subset = ~500 bytes), env metadata (~200 bytes), threshold values (~100 bytes), link to the diagnostic artifact for raw evidence. Keep raw `.bench.txt` OUT of step summary.

### Pitfall 7: `benchstat@latest` fetches a different version on push-on-main vs PR runs
**What goes wrong:** push-on-main captures baseline with benchstat v0.0.0-20250101 format; days later, PR runs with benchstat v0.0.0-20250715 format that has slightly different table padding. Regex-based parsing fails on the PR side.
**Why it happens:** Both jobs install via `@latest`. Module proxy caches generally make this stable within a day, but multi-week-old baselines could theoretically encounter a benchstat update.
**How to avoid:** Have `summary.json` record the resolved benchstat version (`benchstat -version` or `go list -m golang.org/x/perf`). If a future format change breaks parsing, the recorded version aids debugging. Pinning to a SHA in `go install` is the heavyweight option (planner discretion per CONTEXT.md).

## Code Examples

### Workflow shell — parsing benchstat output for regressions

```python
# scripts/bench/check_pr_regression.py (sketch)
"""
Reads benchstat comparison output (run_benchstat.sh --old <baseline> --new <head>)
and emits:
  - summary.json   {regression: bool, threshold_pct, p_max, flagged_rows: [...], target: {...}}
  - markdown.md    GitHub step-summary fragment + sticky comment body
Exit code: 0 always (D-13 advisory). A future env var REQUIRE_NO_REGRESSION=1 flips this.
"""
from __future__ import annotations
import argparse, json, os, pathlib, re, sys

# Reuse from Phase 9
sys.path.insert(0, str(pathlib.Path(__file__).parent))
from check_benchmark_claims import parse_benchmark_file  # noqa

DELTA_RE = re.compile(r"(?<![\w.])([+-])(\d+(?:\.\d+)?)%\s+\(p=(\d+\.\d+)\s+n=\d+\)")
ROW_PREFIX_RE = re.compile(r"^\s*(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_\S+")

def parse_benchstat_for_regressions(text: str, threshold_pct: float, p_max: float):
    flagged = []
    for line in text.splitlines():
        if not ROW_PREFIX_RE.match(line):
            continue
        if "~" in line:
            continue  # benchstat says non-significant
        m = DELTA_RE.search(line)
        if not m:
            continue
        sign, pct_str, p_str = m.groups()
        if sign != "+":
            continue
        pct = float(pct_str)
        p = float(p_str)
        if pct >= threshold_pct and p < p_max:
            flagged.append({"row": line.strip().split()[0], "delta_pct": pct, "p_value": p, "raw_line": line.strip()})
    return flagged

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--benchstat-output", required=True, type=pathlib.Path)
    parser.add_argument("--threshold-pct", type=float, default=5.0)
    parser.add_argument("--p-max", type=float, default=0.05)
    parser.add_argument("--summary-out", required=True, type=pathlib.Path)
    parser.add_argument("--markdown-out", required=True, type=pathlib.Path)
    parser.add_argument("--no-baseline", action="store_true",
                        help="Cache miss; emit bypass notice")
    args = parser.parse_args()

    if args.no_baseline:
        summary = {"regression": False, "bypassed": True, "reason": "no baseline cache available",
                   "threshold_pct": args.threshold_pct, "p_max": args.p_max, "flagged_rows": []}
        markdown = "## PR Benchmark — advisory bypass\n\nNo baseline cache was available...\n"
    else:
        text = args.benchstat_output.read_text()
        flagged = parse_benchstat_for_regressions(text, args.threshold_pct, args.p_max)
        summary = {"regression": bool(flagged), "bypassed": False,
                   "threshold_pct": args.threshold_pct, "p_max": args.p_max,
                   "flagged_rows": flagged}
        markdown = render_markdown(flagged, args.threshold_pct, args.p_max)

    args.summary_out.write_text(json.dumps(summary, indent=2, sort_keys=True))
    args.markdown_out.write_text(markdown)
    return 0  # D-13: always exit 0 in advisory mode

def render_markdown(flagged, threshold_pct, p_max):
    if not flagged:
        return f"## PR Benchmark — no regressions ✓\n\nThreshold: ≥{threshold_pct}% slower AND p<{p_max}.\n"
    lines = [f"## PR Benchmark — {len(flagged)} regression(s) flagged ⚠ (advisory)\n",
             f"Threshold: ≥{threshold_pct}% slower AND p<{p_max}\n",
             "| Row | Δ% | p-value |", "|-----|----|---------|"]
    for f in flagged:
        lines.append(f"| `{f['row']}` | +{f['delta_pct']:.2f}% | {f['p_value']:.3f} |")
    lines.append("\n_This check is advisory; the PR is not blocked. See artifact for raw evidence._")
    return "\n".join(lines)

if __name__ == "__main__":
    sys.exit(main())
```

### Workflow YAML skeleton — pr-benchmark.yml

```yaml
name: pr benchmark
on:
  pull_request:
    paths-ignore:
      - '**.md'
      - 'docs/**'
      - 'LICENSE'
      - 'NOTICE'
      - '.planning/**'
      - '.github/workflows/**'   # Recommended: drop the "except" exception (see Pitfall §D-17)
      - '.github/actions/**'
      - 'testdata/benchmark-results/**'

concurrency:
  group: pr-bench-${{ github.event.pull_request.number }}
  cancel-in-progress: true

permissions:
  contents: read
  pull-requests: write   # required for sticky-comment; will be denied on fork PRs

jobs:
  bench:
    runs-on: ubuntu-latest
    timeout-minutes: 12   # 2 min headroom over the D-01 budget
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
        with: { submodules: recursive }
      - uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
        with: { go-version-file: go.mod }
      - uses: ./.github/actions/setup-rust
        with: { toolchain-file: rust-toolchain.toml }
      - name: Cache Rust release build
        uses: actions/cache@v4
        with:
          path: target/release/
          key: pr-bench-rust-${{ hashFiles('Cargo.lock', 'third_party/simdjson/**') }}
      - name: Install benchstat
        run: |
          go install golang.org/x/perf/cmd/benchstat@latest
          echo "$(go env GOPATH)/bin" >>"$GITHUB_PATH"
      - name: Build native release
        run: cargo build --release
      - id: restore-baseline
        name: Restore main-baseline cache
        uses: actions/cache/restore@v4
        with:
          path: baseline.bench.txt
          key: pr-bench-baseline-NEVER-MATCHES   # force restore-keys path
          restore-keys: |
            pr-bench-baseline-
      - name: Run PR benchmark + regression check
        env:
          NO_BASELINE: ${{ steps.restore-baseline.outputs.cache-matched-key == '' }}
        run: bash scripts/bench/run_pr_benchmark.sh
      - name: Append step summary
        if: always()
        run: cat pr-bench-summary/markdown.md >> "$GITHUB_STEP_SUMMARY"
      - name: Post sticky PR comment
        if: always()
        continue-on-error: true   # fork PRs lack pull-requests:write
        uses: marocchino/sticky-pull-request-comment@v3.0.4
        with:
          header: pr-benchmark-regression
          path: pr-bench-summary/markdown.md
      - name: Upload diagnostic artifacts
        if: always()
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
        with:
          name: pr-bench-evidence-${{ github.event.pull_request.number }}-${{ github.run_id }}
          path: pr-bench-summary/
          retention-days: 14
          if-no-files-found: warn
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `actions/cache@v3` | `actions/cache@v4` | Feb 2025 (v3 deprecated) | v4 uses new cache service v2 APIs; v3 stops working after deprecation date |
| `actions/cache@v4` exclusively | Optionally `actions/cache/restore@v4` + `actions/cache/save@v4` | Available since v4.0 | Finer control: producer can save-only, consumer can restore-only |
| `marocchino/sticky-pull-request-comment@v2.x` | `@v3.0.4` (current) | April 2026 | v3 is current major; v2.9.x still maintained but v3 is the active line |
| `paths-ignore` with `!` negation | Not supported (never was) | n/a | Plan around it — don't try to negate inside `paths-ignore` |

**Deprecated/outdated:**
- `actions/cache@v3` and earlier: no longer functional after Feb 2025 deprecation enforcement.
- `actions/cache@v2` (`@actions/cache` toolkit < 4.0): same deadline.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Python `unittest` (matches `tests/bench/test_check_benchmark_claims.py` style) + `go test` for any Go-side wiring |
| Config file | none (Python stdlib `unittest`); `go.mod` for Go side |
| Quick run command | `python3 -m unittest tests/bench/test_check_pr_regression.py -v` |
| Full suite command | `python3 -m unittest discover tests/bench -v && go test ./... -run 'TestPR|TestRegression' -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-11 | Row flagged when ≥5% slower AND p<0.05 | unit | `python3 -m unittest tests.bench.test_check_pr_regression.TestThreshold -v` | ❌ Wave 0 |
| D-11 | Row NOT flagged when ≥5% but p≥0.05 (`~` sentinel) | unit | same | ❌ Wave 0 |
| D-11 | Row NOT flagged when p<0.05 but Δ<5% | unit | same | ❌ Wave 0 |
| D-11 | Boundary: exactly +5.00% with p=0.04999 → flagged | unit | same | ❌ Wave 0 |
| D-11 | Boundary: exactly +4.99% with p=0.001 → NOT flagged | unit | same | ❌ Wave 0 |
| D-11 | Negative deltas (faster) NEVER flagged | unit | same | ❌ Wave 0 |
| D-12 | Per-row granularity: every flagged row appears individually | unit | `TestPerRowGranularity` | ❌ Wave 0 |
| D-08 | Cache miss → bypass notice in summary, exit 0 | unit | `TestNoBaselineBypass` | ❌ Wave 0 |
| D-13 | Always exits 0 in advisory mode | unit | `TestAdvisoryAlwaysZero` | ❌ Wave 0 |
| D-15 | Markdown fragment includes row name, delta, p-value | unit | `TestMarkdownRenderer` | ❌ Wave 0 |
| D-02/D-05 | `-bench` regex includes Tier 1/2/3 + filters comparators | integration | `TestBenchRegexProducesExpectedRows` (runs `go test -bench=... -run='^$' -benchtime=1x` and asserts emitted row names) | ❌ Wave 0 |
| D-19 (degraded fork path) | When `pull-requests: write` denied (simulated via `continue-on-error`), step summary still posts | manual smoke | run a fork-PR through CI once after ship | manual-only |
| Real-benchstat fixture parse | Parser handles real `tier1-vs-stdlib.benchstat.txt` row format | integration | `TestRealBenchstatFixture` (uses `testdata/benchmark-results/v0.1.2/phase9.benchstat.txt` as input) | ❌ Wave 0 |
| End-to-end | `run_pr_benchmark.sh` produces summary.json + markdown.md when given mocked baseline + head | integration | `TestRunPRBenchmarkScriptSmoke` (bash + python via subprocess) | ❌ Wave 0 |
| Malformed input | Truncated benchstat output → parser errors clearly, doesn't silently pass | unit | `TestMalformedBenchstatFailsClosed` | ❌ Wave 0 |

### Sampling Sufficiency

Coverage matrix for the regression parser (cross product: delta sign × significance × magnitude):

| | p < 0.05 | p ≥ 0.05 (`~`) |
|---|----------|----------------|
| **Δ ≥ +5%** (slower, large) | FLAG ✓ | not flagged |
| **0 < Δ < +5%** (slower, small) | not flagged | not flagged |
| **Δ = 0%** (no change) | not flagged (impossible — `~`) | not flagged |
| **Δ < 0** (faster) | not flagged | not flagged |

Plus boundary cases at exactly Δ=5.00% and exactly p=0.05000 → fixture each.

Plus structural cases:
- `geomean` row in input → ignored
- Header/separator lines → ignored
- Comment column rows (raw `.bench.txt` instead of benchstat) → parser errors (wrong input)
- Empty input → parser errors
- Multi-tier all-significant input → all flagged, none missed

### Boundary Conditions (explicit fixtures)

1. `+5.00% (p=0.049 n=5)` → FLAG (≥5% AND <0.05)
2. `+4.99% (p=0.001 n=5)` → NOT (Δ<5%)
3. `+5.01% (p=0.050 n=5)` → NOT (`p` not strictly <0.05; benchstat would emit `~` here, but test it explicitly)
4. `~ (p=0.075 n=5)` → NOT (sentinel)
5. `-94.80% (p=0.000 n=5)` → NOT (faster)
6. `+5.00% (p=0.049 n=5)` AND `+10.00% (p=0.001 n=5)` AND `~ (p=0.300 n=5)` in same input → flag exactly two rows in order

### Failure Injection

| Failure | Test | Expected Behavior |
|---------|------|-------------------|
| `baseline.bench.txt` missing (cache miss) | `TestNoBaselineBypass` | Exit 0, summary.regression=false, summary.bypassed=true, markdown contains "advisory bypass" |
| `head.bench.txt` malformed (truncated row) | `TestMalformedHead` | Exit 1 (parser error), no false-positive regression flag |
| benchstat output empty (no rows) | `TestEmptyBenchstat` | Exit 1 (no rows = parsing failure, not "no regressions") |
| sticky-comment 403 (fork PR) | `continue-on-error: true` in workflow YAML; smoke-tested manually post-ship | Step summary still appears; comment skipped silently |
| benchstat install fails | (out of scope — workflow setup error) | Workflow fails before reaching parse step |

### Wave 0 Gaps

- [ ] `tests/bench/test_check_pr_regression.py` — covers D-08, D-11, D-12, D-13, D-15
- [ ] `tests/bench/fixtures/pr-regression/` directory with synthetic benchstat outputs (≥10 small fixtures)
- [ ] `scripts/bench/run_pr_benchmark.sh` — bash orchestrator, ≤200 LOC
- [ ] `scripts/bench/check_pr_regression.py` — regression parser
- [ ] `.github/workflows/pr-benchmark.yml` — PR trigger workflow
- [ ] `.github/workflows/main-benchmark-baseline.yml` — push-on-main producer workflow
- [ ] No new framework install needed — `python3 -m unittest` and `go test` already in CI

### Sampling Rate

- **Per task commit:** `python3 -m unittest tests.bench.test_check_pr_regression -v` (≤2s, 12+ assertions)
- **Per wave merge:** add `python3 -m unittest discover tests/bench -v && go test ./... -run 'TestPR|TestBenchRegex' -count=1`
- **Phase gate:** Full suite green + ONE manual `workflow_dispatch` smoke of the new pr-benchmark workflow (since `pull_request` can't fire from a feature branch trivially), validating D-08 (no-baseline path) and D-19 (step summary visibility) end-to-end on real ubuntu-latest. Then a real PR confirms the sticky-comment path.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | n/a — workflows use ambient `GITHUB_TOKEN`, not user auth |
| V3 Session Management | no | n/a |
| V4 Access Control | yes | `permissions:` block in workflow YAML; minimum-privilege (`contents: read`, `pull-requests: write` only) |
| V5 Input Validation | yes | Python regex parsing of benchstat output is the primary input boundary; `parse_benchmark_file()` from Phase 9 already rejects malformed rows |
| V6 Cryptography | no | No crypto — cache transport is plain bytes, signature is GitHub-internal |
| V14 Configuration | yes | Pin all third-party action SHAs (already the project pattern); never use `@main` or `@v1` floating tags |

### Known Threat Patterns for GitHub Actions PR Workflows

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Fork PR runs base-repo write tokens (e.g., `pull_request_target` mistake) | E (Elevation of Privilege) | Use `pull_request` (not `pull_request_target`); accept that fork PRs cannot post sticky comments. CONTEXT.md correctly chose this. |
| Cache poisoning via PR-controlled cache writes | T (Tampering) | PR workflow MUST NOT write to the `pr-bench-baseline-*` cache key. Only the push-on-main workflow writes. (Already CONTEXT.md D-06.) |
| Third-party action SHA pinning drift | T | Pin every action to its full commit SHA (already the project pattern in `benchmark-capture.yml`). v3.0.4 of sticky-comment must be pinned to the exact SHA, not the tag. |
| Untrusted PR code triggers expensive workflow on every push | D (Denial of Service / cost) | `cancel-in-progress: true` (D-20) limits stacked runs; `paths-ignore` (D-16/D-17) skips no-op changes. |
| Sticky-comment 403 on fork PR fails the whole workflow | A (Availability) | `continue-on-error: true` on the comment step; step summary is the always-works fallback (D-19). |
| Bench output exfiltrates secrets | I (Information Disclosure) | Bench output is `.bench.txt` only — no env, no `pwd`. Verify the orchestrator script does not echo `env` or `set -x` near credentials. |

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Bench harness | ✓ (provided by `actions/setup-go`) | go.mod-driven (current: 1.24+) | — |
| Rust + cargo | `cargo build --release` | ✓ (composite action `setup-rust`) | rust-toolchain.toml-driven | — |
| Python 3.11+ | Regression parser | ✓ (ubuntu-latest provides) | system | — |
| benchstat | Comparison | ✓ (`go install` step) | `@latest` | — |
| `actions/cache@v4` | Baseline transport | ✓ (GitHub-hosted) | v4 | — |
| `marocchino/sticky-pull-request-comment@v3.0.4` | PR comment | ✓ (Marketplace) | v3.0.4 | Step summary (always works) |
| Hosted runner `ubuntu-latest` | Workflow execution | ✓ | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** sticky-comment is degradable to step-summary-only on fork PRs (D-19 already plans for this).

## Project Constraints (from CLAUDE.md)

- **Conventional commits required.** Commit messages must follow `type(scope): subject` format. Phase 10 commits should be e.g. `feat(ci): add PR benchmark regression workflow`, `feat(bench): add PR regression parser`, `test(bench): cover boundary thresholds for regression parser`.
- **No internal teliacompany references** in commits, PRs, issues. Phase 10 has no contact with internal infra — automatic compliance.
- **No "Generated with Claude Code" attribution** in commits, PRs, issues, or any artifact.
- **GH issue prefixes:** `[BUG]`, `[ENH]`, `[PERF]`, `[TST]`, `[CLN]`, `[CHORE]`, `[DOC]`, `[BLD]`. Phase 10 issues, if any, would be `[CI]` is not in the list — closest match `[BLD]` (build/CI). Planner should clarify if any issue is filed; otherwise the phase ships entirely via PR.
- **Don't push to main without a PR.** Phase 10 ships through `gsd/phase-10-lightweight-pr-benchmark-regression-signal` → PR → main, mirroring every prior phase.
- **Keep things radically simple.** ≤200 LOC bash orchestrator, ≤300 LOC Python parser, two thin workflow YAMLs. Resist the temptation to build a "regression dashboard."
- **Self-explanatory code over verbose comments.** Bench-row regex needs ONE comment explaining the benchstat format reference; the rest should be plain Python.
- **Use `yq` for YAML validation.** Plan should include a `yq eval` step in the planner's commit checks for the two new workflow files.

## Assumptions Log

> Claims tagged `[ASSUMED]` in this research that need user/planner confirmation before becoming locked decisions. Empty if all claims were verified or cited.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Phase 9 uses `benchstat@latest` (not pinned to a SHA) | Standard Stack | Low — verified by reading `.github/workflows/benchmark-capture.yml` line 47-48; this is `[VERIFIED: in-repo]`, not assumed. Strike from list. |
| A2 | A `cargo build --release` Rust cache speeds first-run from ~4 min to ~30 s | Pitfall 3 | Medium — performance estimate from Phase 6/7 CI logs. Real number depends on simdjson amalgamation incremental rebuild behavior. Worth a measurement during planning's Wave 0 spike. |
| A3 | The workflow PR file's own self-edit triggering is acceptable risk per Pitfall §D-17 Option 1 | Anti-Patterns / D-17 | Medium — depends on user's preference. Pitfall §D-17 surfaces this for explicit choice. |
| A4 | Step-summary 1MB cap is sufficient for the locked subset (D-02..D-05) | Pitfall 6 | Low — math says ~3KB for 6 rows; very wide buffer. |
| A5 | `pull_request` event from a fork denies `pull-requests: write` on `GITHUB_TOKEN` | Threat patterns / D-19 | Low — well-documented GitHub policy. Sticky-comment issue #227 confirms the failure mode is "Resource not accessible by integration." |

Items A2 and A3 are the only meaningfully open assumptions; all others are verified.

## Open Questions

1. **Should the PR-bench workflow file's self-edits trigger the bench? (D-17 ambiguity)**
   - What we know: `paths-ignore` cannot express "ignore everything in `.github/workflows/` except this one file."
   - What's unclear: User's preference between Option 1 (drop the exception, ignore all workflow edits) and Option 2 (use `paths` with negation, contradicting D-16).
   - Recommendation: Option 1 (drop the exception). Surface this in plan-checker if planner chooses differently.

2. **Should `target/release/` be cached? (Not in CONTEXT.md)**
   - What we know: Cold `cargo build --release` is ~3-4 min on hosted runner; D-01 budget is 10 min.
   - What's unclear: Whether the planner is willing to add a second cache (separate from the baseline cache) to make the budget comfortable.
   - Recommendation: YES — cache `target/release/` keyed on `Cargo.lock` + simdjson submodule SHA. This is purely a performance optimization, not a correctness concern; failure mode is "first run after Rust dep change is slow," which is acceptable.

3. **Is the future blocking-flip control surface (D-14) an env var, workflow input, or a workflow-file constant?**
   - What we know: D-14 says "clearly named control surface." All three are candidates.
   - What's unclear: Which is most ergonomic for the eventual blocking PR (target audience: a maintainer who reads the file).
   - Recommendation: Workflow-file constant via a `step.env` like `REQUIRE_NO_REGRESSION: "false"` at the top of the regression-check step. Future PR flips it to `"true"`. One-line diff, grep-able, no secrets/inputs needed.

4. **`benchstat` version pinning policy?**
   - What we know: Phase 9 uses `@latest`; Phase 10 inherits.
   - What's unclear: Whether the planner wants to pin to a SHA for reproducibility insurance.
   - Recommendation: Match Phase 9 (`@latest`) for now; record resolved version in `summary.json`. Pin if Pitfall §7 ever fires.

5. **Should the push-on-main baseline workflow run on `paths-ignore` too?**
   - What we know: D-16/D-17 lock paths-ignore for the PR workflow. The push-on-main producer is not directly addressed.
   - What's unclear: Whether docs-only main pushes should regenerate the baseline cache (cheap insurance against staleness — they'd refresh the cache LRU timestamp) or skip (saves CI minutes on docs commits).
   - Recommendation: SAME `paths-ignore` set. A docs-only main push doesn't change benchmark behavior, so refreshing the cache is wasted CI minutes. The 7-day eviction is unchanged either way.

## Sources

### Primary (HIGH confidence)
- **GitHub Docs — Dependency caching reference** ([VERIFIED](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching)) — cache scope, restore-keys prefix matching, 7-day eviction, PR base-branch fallback hierarchy.
- **github.com/actions/cache (v4)** ([VERIFIED](https://github.com/actions/cache)) — `restore-keys` "most recently created" tie-breaking, v3 deprecation Feb 2025, v4 cache service v2 APIs.
- **pkg.go.dev/golang.org/x/perf/cmd/benchstat** ([VERIFIED](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)) — output format `Δ% (p=… n=…)`, `~` sentinel for `p > 0.05`, default Mann-Whitney U-test.
- **github.com/marocchino/sticky-pull-request-comment v3.0.4** ([VERIFIED](https://github.com/marocchino/sticky-pull-request-comment/releases/tag/v3.0.4)) — released 2026-04-10; `pull-requests: write` permission requirement; `header` parameter for stickiness; supports `path` input for file-based message.
- **GitHub Docs — workflow syntax `paths-ignore`** ([VERIFIED](https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions)) — "all paths must match for skip" semantics; no `!` negation in `paths-ignore`.
- **In-repo Phase 9 evidence** ([VERIFIED](file://testdata/benchmark-results/v0.1.2/phase9.benchstat.txt)) — actual benchstat output format with the exact rows the regression parser must handle.
- **In-repo `scripts/bench/check_benchmark_claims.py`** — `parse_benchmark_file()` and `SIGNIFICANT_WIN_RE` patterns to import from.

### Secondary (MEDIUM confidence)
- **GitHub Docs — `$GITHUB_STEP_SUMMARY`** ([CITED](https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#adding-a-job-summary)) — 1MB cap; markdown including tables and `<details>` supported. Verified via the `actions/dependency-review-action` 1MB-abort issue (real failure mode).
- **community discussion on `paths-ignore` PR semantics** ([CITED](https://github.com/orgs/community/discussions/54877)) — known issue: after a non-ignored commit lands in a PR, subsequent commits trigger the workflow even if they only touch ignored paths. Acceptable for Phase 10 (no false negatives).
- **sticky-pull-request-comment fork issue #227** ([CITED](https://github.com/marocchino/sticky-pull-request-comment/issues/227)) — confirms "Resource not accessible by integration" is the failure mode for fork PRs without write tokens.

### Tertiary (LOW confidence — none used)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every action/library was verified by URL fetch or in-repo inspection.
- Architecture: HIGH — the design directly mirrors Phase 9's `benchmark-capture.yml` pattern with one well-understood twist (cache-mediated baseline transport).
- Pitfalls: HIGH — six of seven pitfalls are sourced from official docs or real in-repo Phase 9 LEARNINGS; Pitfall 3 (Rust build cost) is informed by Phase 6 timing observations and is the one open assumption (A2).
- D-decisions reconciliation: HIGH — all 21 decisions verified against external reality. ZERO contradictions found. The only sharpening needed is on D-17's negation expressibility, surfaced as a planning-time choice.

**Research date:** 2026-04-27
**Valid until:** 2026-05-27 (30 days for stable surfaces — actions/cache, benchstat, paths-ignore semantics rarely change; sticky-comment v3 is current major and unlikely to break in a month)

---

*Phase: 10-lightweight-pr-benchmark-regression-signal*
*Researched: 2026-04-27*
