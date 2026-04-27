# Phase 10: Lightweight PR benchmark regression signal — Pattern Map

**Mapped:** 2026-04-27
**Files analyzed:** 7 (6 new + 1 modified)
**Analogs found:** 7 / 7 (every new file has a strong in-repo analog)

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.github/workflows/pr-benchmark.yml` (new) | workflow / event-trigger | event-driven (`pull_request` → restore cache → bench → benchstat → comment + summary + artifact) | `.github/workflows/benchmark-capture.yml` | role-match (event differs: `pull_request` vs `workflow_dispatch`; otherwise identical setup pipeline) |
| `.github/workflows/main-benchmark-baseline.yml` (new) | workflow / event-trigger | event-driven (`push: branches: [main]` → bench → write cache + diagnostic artifact) | `.github/workflows/benchmark-capture.yml` | role-match (same setup pipeline, smaller capture, writes to `actions/cache` instead of `upload-artifact` only) |
| `scripts/bench/run_pr_benchmark.sh` (new, ≤200 LOC) | shell orchestrator | batch (subprocess over `go test -bench`, `run_benchstat.sh`, `check_pr_regression.py`) | `scripts/bench/capture_release_snapshot.sh` | role-match (same bench-then-benchstat-then-summary sequence; PR variant is intentionally smaller — single subset, no metadata.json/normalization step) |
| `scripts/bench/check_pr_regression.py` (new) | python parser / regression gate | transform (benchstat text → `summary.json` + markdown fragment) | `scripts/bench/check_benchmark_claims.py` | role-match (same language, reuses `parse_benchmark_file()`; semantics differ — bidirectional regression vs unidirectional "win") |
| `tests/bench/test_check_pr_regression.py` (new) | python unittest contract test | request-response (subprocess over the parser; fixture file → asserted summary.json/markdown) | `tests/bench/test_check_benchmark_claims.py` | exact (same `unittest.TestCase` style, same `subprocess.run` invocation pattern, same temp-dir fixture write pattern) |
| `tests/bench/fixtures/pr-regression/` (new directory) | fixture / data | file-I/O (synthetic + real benchstat outputs read by tests) | `testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt` | role-match (real benchstat row format used as one fixture; synthetic fixtures cover boundary cases per RESEARCH § Boundary Conditions) |
| `CHANGELOG.md` (modify) | doc | append-only | n/a — small "Unreleased" line edit, no analog needed | n/a |

---

## Pattern Assignments

### `.github/workflows/pr-benchmark.yml` (workflow, event-driven)

**Analog:** `.github/workflows/benchmark-capture.yml`

**Pinned-action imports + setup pipeline** (analog lines 28-51) — copy verbatim, swap trigger and add cache-restore + summary/comment steps:

```yaml
      - name: Check out repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
        with:
          submodules: recursive

      - name: Set up Go
        uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
        with:
          go-version-file: go.mod

      - name: Install pinned Rust toolchain
        uses: ./.github/actions/setup-rust
        with:
          toolchain-file: rust-toolchain.toml

      - name: Install benchstat
        run: |
          set -euo pipefail
          go install golang.org/x/perf/cmd/benchstat@latest
          echo "$(go env GOPATH)/bin" >>"$GITHUB_PATH"

      - name: Build native release library
        run: cargo build --release
```

**Permissions + concurrency block** (NEW vs analog — analog uses `workflow_dispatch` so no PR concerns; PR workflow needs `pull-requests: write` and the cancellable concurrency group from D-20):

```yaml
permissions:
  contents: read
  pull-requests: write   # required for sticky-comment; denied on fork PRs (continue-on-error swallows)

concurrency:
  group: pr-bench-${{ github.event.pull_request.number }}
  cancel-in-progress: true
```

**Trigger block** (NEW vs analog — D-16/D-17, with the §D-17 expressibility resolution from RESEARCH: drop the workflow-file negation):

```yaml
on:
  pull_request:
    paths-ignore:
      - '**.md'
      - 'docs/**'
      - 'LICENSE'
      - 'NOTICE'
      - '.planning/**'
      - '.github/workflows/**'   # see RESEARCH §D-17 — paths-ignore cannot negate a single file
      - '.github/actions/**'
      - 'testdata/benchmark-results/**'
```

**Upload-artifact pattern with `if: always()` and retention** (analog lines 59-66) — copy structure, change retention from 30 → 14 (D-21):

```yaml
      - name: Upload diagnostic artifacts
        if: always()
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
        with:
          name: pr-bench-evidence-${{ github.event.pull_request.number }}-${{ github.run_id }}
          path: pr-bench-summary/
          retention-days: 14
          if-no-files-found: warn
```

**Defaults block to use bash** (analog lines 25-27) — copy verbatim:

```yaml
    defaults:
      run:
        shell: bash
```

**Named pattern for planner:** "Mirror benchmark-capture.yml's setup pipeline byte-for-byte (same pinned SHAs); add `pull_request` trigger + paths-ignore + concurrency + permissions + cache-restore + step-summary + sticky-comment + 14-day artifact upload."

---

### `.github/workflows/main-benchmark-baseline.yml` (workflow, event-driven)

**Analog:** `.github/workflows/benchmark-capture.yml`

**Same setup pipeline** as the PR workflow above (checkout / setup-go / setup-rust / install benchstat / cargo build --release — copy from analog lines 28-51 verbatim).

**Trigger block** (NEW — D-06, plus RESEARCH § Open Question 5 recommendation: same paths-ignore as PR):

```yaml
on:
  push:
    branches: [main]   # MUST be exactly main — RESEARCH § Pitfall 1 (cache scope)
    paths-ignore:
      # match the PR workflow's set so docs-only main pushes don't churn the baseline
      - '**.md'
      - 'docs/**'
      - 'LICENSE'
      - 'NOTICE'
      - '.planning/**'
      - '.github/workflows/**'
      - '.github/actions/**'
      - 'testdata/benchmark-results/**'

concurrency:
  group: main-bench-baseline   # different group from PR (D-20) so it never cancels itself
  cancel-in-progress: false
```

**Cache-write step** (NEW — D-06/D-07; RESEARCH Pattern 2):

```yaml
      - name: Save baseline to actions/cache
        uses: actions/cache/save@v4
        with:
          path: baseline.bench.txt
          key: pr-bench-baseline-${{ github.sha }}
```

**Named pattern for planner:** "Producer-only baseline workflow: same setup pipeline as PR workflow + same paths-ignore set; runs the same `run_pr_benchmark.sh` (or a thin caller) but writes the head bench output to the canonical cache key instead of restoring."

---

### `scripts/bench/run_pr_benchmark.sh` (shell orchestrator, batch)

**Analog:** `scripts/bench/capture_release_snapshot.sh`

**Bash strict mode + arg-parser shape** (analog lines 1-52) — copy the `set -euo pipefail` + `usage()` + `while [[ $# -gt 0 ]]` parser idiom:

```bash
#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 [--baseline <path>] [--out-dir <path>]" >&2
}

# parse --baseline / --out-dir / -h same way capture_release_snapshot.sh does
```

**Tool preflight** (analog lines 69-74) — copy verbatim, narrow tool list to PR-needed set:

```bash
for tool in go rustc git python3 benchstat; do
	if ! command -v "$tool" >/dev/null 2>&1; then
		echo "$tool not found in PATH" >&2
		exit 1
	fi
done
```

**Stage-dir + cleanup-trap pattern** (analog lines 76-111) — copy structure, simpler since PR has no "promote to public path" step. Keep `mktemp -d` + `trap cleanup EXIT` so partial outputs don't leak.

**The bench-then-benchstat-then-summary sequence** (analog lines 113-137; this is THE PATTERN to mirror, scaled down to D-02/D-04/D-05):

```bash
# Capture head bench (analog: phase9.bench.txt at -count=10; PR uses -count=5 per D-04, single subset per D-02/D-05)
go test ./... -run '^$' \
    -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_(twitter|canada)_json/(pure-simdjson|encoding-json-any|encoding-json-struct)$' \
    -benchmem -count=5 -timeout 600s >"$stage_dir/head.bench.txt"

# Run benchstat against restored baseline (analog: scripts/bench/run_benchstat.sh --old <baseline> --new <new>)
if [[ -f "$baseline_path" ]]; then
    scripts/bench/run_benchstat.sh --old "$baseline_path" --new "$stage_dir/head.bench.txt" \
        >"$stage_dir/regression.benchstat.txt"
fi

# Generate summary.json + markdown.md (analog: check_benchmark_claims.py invocation lines 202-215)
python3 scripts/bench/check_pr_regression.py \
    --benchstat-output "$stage_dir/regression.benchstat.txt" \
    --summary-out "$stage_dir/summary.json" \
    --markdown-out "$stage_dir/markdown.md" \
    ${baseline_missing:+--no-baseline}
```

**Critical excerpt — D-02/D-05 `-bench` regex** (constructed from in-repo function names; verified against `benchmark_fullparse_test.go` lines 7-17, `benchmark_typed_test.go`, `benchmark_selective_test.go`, and `benchmark_comparators_test.go` lines 22-37):

- Tier 1 family functions (verified): `BenchmarkTier1FullParse_twitter_json`, `BenchmarkTier1FullParse_citm_catalog_json`, `BenchmarkTier1FullParse_canada_json` → drop `citm_catalog` per D-03
- Tier 2 family functions (verified): `BenchmarkTier2Typed_twitter_json`, `BenchmarkTier2Typed_citm_catalog_json`, `BenchmarkTier2Typed_canada_json` → drop `citm_catalog` per D-03
- Tier 3 family functions (verified): `BenchmarkTier3SelectivePlaceholder_twitter_json`, `BenchmarkTier3SelectivePlaceholder_citm_catalog_json` → only `twitter_json` survives D-03 (canada is not a Tier 3 fixture)
- Comparators registered (verified, `benchmark_comparators_test.go` lines 70-86): `pure-simdjson`, `encoding-json-any`, `encoding-json-struct`, `goccy-go-json`. Minio + sonic are platform-stub-gated. Goccy is registered unconditionally — the regex must explicitly EXCLUDE it (D-05).

The sub-test row name is `<bench>/<comparator>` (verified by `phase9.bench.txt` row format: `BenchmarkTier1FullParse_twitter_json/pure-simdjson-4`). The Go `-bench` flag accepts a regex matched against the `/`-joined path; an anchored `$` on the comparator group is the cleanest filter.

**Named pattern for planner:**
1. "Copy `capture_release_snapshot.sh`'s skeleton (strict mode → arg parse → tool preflight → stage-dir trap → numbered-step `current_step` updates) but TRIM to a single bench capture (no coldwarm, no diagnostics, no normalization) plus one benchstat run plus one Python summary call."
2. "Cap at ≤200 LOC. Resist the urge to add metadata.json — D-discretion allows reusing only the slim `summary.json` shape produced by `check_pr_regression.py`."
3. "The `-bench` regex MUST be constructed from the verified function-name list above; do not use a wildcard like `Benchmark.*` (that would re-include Tier 1 diagnostics, coldwarm, and the goccy/minio/sonic comparators)."

**Anti-pattern (explicitly NOT to copy from analog):** Lines 117-155 of `capture_release_snapshot.sh` run THREE separate captures (phase9 + coldwarm + diagnostics) plus three normalization calls plus three benchstats. That is release-grade, not PR-grade. PR script runs ONE capture and ONE benchstat.

---

### `scripts/bench/check_pr_regression.py` (python parser, transform)

**Analog:** `scripts/bench/check_benchmark_claims.py`

**Imports + module-level constants pattern** (analog lines 1-56):

```python
#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys
from typing import Any
```

Reuse `parse_benchmark_file()` (analog lines 144-164). Copy the import idiom from RESEARCH § Code Examples:

```python
# scripts/bench/check_pr_regression.py
sys.path.insert(0, str(pathlib.Path(__file__).parent))
from check_benchmark_claims import parse_benchmark_file, EvidenceError  # noqa: E402
```

**The regex divergence to flag explicitly** (analog line 56 — UNIDIRECTIONAL, faster only):

```python
# in check_benchmark_claims.py — DO NOT reuse for Phase 10
SIGNIFICANT_WIN_RE = re.compile(r"(?<![\w.])-\d+(?:\.\d+)?%")
```

vs. the new bidirectional regex Phase 10 must own (per RESEARCH Pattern 4, refined for the `n=` segment seen in real fixtures):

```python
# scripts/bench/check_pr_regression.py — bidirectional, captures sign + p
DELTA_RE = re.compile(
    r"(?<![\w.])([+-])(\d+(?:\.\d+)?)%\s+\(p=(\d+\.\d+)\s+n=\d+\)"
)
ROW_PREFIX_RE = re.compile(
    r"^\s*(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_\S+"
)
```

**Critical anchor — real benchstat row format** (verified from `testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt` line 7):

```
Tier1FullParse_twitter_json-4   6.443m ± 1%   2.044m ± 2%  -68.27% (p=0.000 n=10)
geomean                         15.74m        5.289m       -66.40%
```

→ benchstat strips the `Benchmark` prefix in its rendered table; rows start with the bench name (no leading `Benchmark`). The `geomean` line MUST be filtered out — `ROW_PREFIX_RE` above does this by requiring `Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder` at the start. (RESEARCH Pitfall 2 explicitly warns about geomean masquerading as a regression row.)

**The `~` sentinel handling** (analog lines 290-303 — `has_significant_win` checks `if "~" in line: return False`):

```python
def has_significant_win(benchstat_text: str, row: str, *, source: str) -> bool:
    row_aliases = (row, row.removeprefix("Benchmark"))
    seen = False
    for line in benchstat_text.splitlines():
        if not any(row_alias in line for row_alias in row_aliases):
            continue
        seen = True
        if "~" in line:
            return False                      # benchstat's non-significance sentinel
        if SIGNIFICANT_WIN_RE.search(line):
            return True
    if not seen:
        raise EvidenceError(f"{source}: required row not found: {row}")
    return False
```

The new parser inverts the polarity — flag `+X.XX%` rows where `pct ≥ 5.0` AND `~` is absent (delta is significant). RESEARCH Pattern 5 supplies the canonical implementation; copy it verbatim.

**`argparse` + JSON-output exit-code pattern** (analog lines 63-71 + 530-538):

```python
def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="...")
    parser.add_argument("--benchstat-output", required=True, type=pathlib.Path)
    parser.add_argument("--summary-out", required=True, type=pathlib.Path)
    parser.add_argument("--markdown-out", required=True, type=pathlib.Path)
    parser.add_argument("--threshold-pct", type=float, default=5.0)
    parser.add_argument("--p-max", type=float, default=0.05)
    parser.add_argument("--no-baseline", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    # ... build summary dict, write summary.json + markdown.md ...
    return 0   # D-13: always exit 0 advisory; future blocking flip flips this


if __name__ == "__main__":
    sys.exit(main())
```

**D-13 control surface** (RESEARCH Open Question 3 recommendation — workflow-file env constant):

```python
# Future blocking flip (D-14): one grep-able knob.
# When env REQUIRE_NO_REGRESSION="true" is set in the workflow YAML,
# main() returns 1 if summary["regression"] is True. Default: "false" → always 0.
require_no_regression = os.environ.get("REQUIRE_NO_REGRESSION", "false").lower() == "true"
return 1 if (require_no_regression and summary["regression"]) else 0
```

**Named pattern for planner:**
1. "Reuse `parse_benchmark_file()` and `EvidenceError` via `sys.path.insert + from check_benchmark_claims import …`. DO NOT copy-paste the parser body."
2. "Copy `check_benchmark_claims.py`'s argparse + main + sys.exit shape; replace the unidirectional `SIGNIFICANT_WIN_RE` with the bidirectional `DELTA_RE` above."
3. "Match the geomean filter via `ROW_PREFIX_RE` — anchor on `Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder`."
4. "Land the `REQUIRE_NO_REGRESSION` env var as the D-14 control surface; document it in a one-line code comment."
5. "Stay ≤300 LOC. The parser is small by design — no tier-aggregation, no metadata verification, no readme-mode chooser."

---

### `tests/bench/test_check_pr_regression.py` (python unittest, request-response)

**Analog:** `tests/bench/test_check_benchmark_claims.py`

**Imports + module constants** (analog lines 1-37):

```python
#!/usr/bin/env python3

from __future__ import annotations

import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "check_pr_regression.py"
```

**Subprocess-driven gate-runner pattern** (analog lines 216-238):

```python
def run_gate(self, benchstat_path: pathlib.Path, *,
             summary_out: pathlib.Path, markdown_out: pathlib.Path,
             threshold_pct: float = 5.0, p_max: float = 0.05) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            sys.executable, str(SCRIPT_PATH),
            "--benchstat-output", str(benchstat_path),
            "--summary-out", str(summary_out),
            "--markdown-out", str(markdown_out),
            "--threshold-pct", str(threshold_pct),
            "--p-max", str(p_max),
        ],
        cwd=REPO_ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
```

**Tempdir-fixture-builder pattern** (analog lines 141-214 — the `write_evidence` helper). Phase 10 simpler: write a single benchstat file, call the gate, parse summary.json:

```python
class CheckPRRegressionTests(unittest.TestCase):
    def test_row_flagged_when_slower_and_significant(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = pathlib.Path(tmp)
            benchstat = tmp_path / "regression.benchstat.txt"
            benchstat.write_text(
                "goos: linux\n"
                "goarch: amd64\n"
                "Tier1FullParse_twitter_json-4   2.000m ± 1%   2.200m ± 1%  +10.00% (p=0.001 n=5)\n"
            )
            result = self.run_gate(
                benchstat,
                summary_out=tmp_path / "summary.json",
                markdown_out=tmp_path / "markdown.md",
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads((tmp_path / "summary.json").read_text())
            self.assertTrue(payload["regression"])
            self.assertEqual(len(payload["flagged_rows"]), 1)
```

**Test-case naming convention** (analog uses `test_<scenario>` — single-line descriptions): the RESEARCH § Phase Requirements → Test Map table dictates the test list. Each row in that table → one test method here.

**Named pattern for planner:**
1. "Copy `test_check_benchmark_claims.py`'s class-based `unittest.TestCase` style + the `subprocess.run(..., capture_output=True, text=True, check=False)` shape."
2. "Mirror `write_evidence` as a smaller `write_benchstat_fixture` helper that writes a single file under `tmp`."
3. "One test per row of RESEARCH § Phase Requirements → Test Map (12+ unit tests + the `TestRealBenchstatFixture` integration test that consumes `testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt`)."
4. "Quick-run command stays `python3 -m unittest tests/bench/test_check_pr_regression.py -v`."

---

### `tests/bench/fixtures/pr-regression/` (fixture directory)

**Analog:** `testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt` (real-data fixture) + `tests/bench/test_check_benchmark_claims.py` `benchstat_text(...)` helper at lines 117-137 (synthetic-fixture style).

**Real-data fixture excerpt** (one of the test inputs — verified, real Phase 9 output):

```
goos: linux
goarch: amd64
pkg: github.com/amikos-tech/pure-simdjson
cpu: AMD EPYC 7763 64-Core Processor
                                   │ ...tier1-base.bench.txt │ ...tier1-candidate.bench.txt │
                                   │   sec/op                │   sec/op       vs base       │
Tier1FullParse_twitter_json-4         6.443m ± 1%              2.044m ± 2%   -68.27% (p=0.000 n=10)
Tier1FullParse_citm_catalog_json-4   17.338m ± 1%              5.113m ± 1%   -70.51% (p=0.000 n=10)
geomean                              15.74m                    5.289m        -66.40%
```

This row format is what the parser MUST handle. The synthetic fixtures should reproduce the same column shape but with controlled deltas:

| Fixture | Purpose | Sample line |
|---|---|---|
| `slower-significant.benchstat.txt` | flag (D-11 happy path) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  2.200m ± 1%  +10.00% (p=0.001 n=5)` |
| `slower-not-significant.benchstat.txt` | NOT flagged (`~` sentinel) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  2.080m ± 5%  ~ (p=0.143 n=5)` |
| `slower-tiny.benchstat.txt` | NOT flagged (Δ<5%) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  2.060m ± 1%  +3.00% (p=0.001 n=5)` |
| `faster-significant.benchstat.txt` | NOT flagged (faster) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  1.000m ± 1%  -50.00% (p=0.000 n=5)` |
| `boundary-5pct.benchstat.txt` | flag (Δ exactly 5.00%, p=0.049) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  2.100m ± 1%  +5.00% (p=0.049 n=5)` |
| `boundary-499pct.benchstat.txt` | NOT flagged (Δ exactly 4.99%) | `Tier1FullParse_twitter_json-4  2.000m ± 1%  2.0998m ± 1%  +4.99% (p=0.001 n=5)` |
| `mixed-multi-row.benchstat.txt` | flag exactly 2 of 3 rows | combination of above |
| `with-geomean.benchstat.txt` | geomean line ignored even when followed by big delta | `geomean   15.74m   16.50m   +4.83%` |
| `empty.benchstat.txt` | parser errors (no rows) | empty / metadata-only |
| `truncated-row.benchstat.txt` | parser errors | row missing `(p=…)` segment |
| `real-tier1-vs-stdlib.benchstat.txt` | integration: actual Phase 9 file copied in (or symlinked) | exact bytes of `testdata/.../tier1-vs-stdlib.benchstat.txt` |

**Named pattern for planner:** "Synthetic fixtures use the same column structure as the real Phase 9 file; one fixture per RESEARCH § Boundary Conditions row; one fixture is a verbatim copy of the real Phase 9 benchstat output to lock in real-row-format coverage (RESEARCH Pitfall 2 fence)."

---

### `CHANGELOG.md` (modify — append-only)

**Analog:** none needed — pattern is "append a one-line bullet under `## Unreleased`." Per CONTEXT § Integration Points: "planner discretion."

**Named pattern for planner:** "Add a single bullet under `## Unreleased` describing the new advisory PR regression check. Conventional-commit subject: `docs(changelog): note PR benchmark regression workflow`."

---

## Shared Patterns

### Pinned-action SHA + composite-action reuse
**Source:** `.github/workflows/benchmark-capture.yml` lines 30, 35, 40, 61
**Apply to:** Both new workflows

```yaml
- uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683     # full SHA — never @v4
- uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
- uses: ./.github/actions/setup-rust                                  # composite, reuse as-is
- uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
```

Project pattern (verified in-repo): every third-party action is pinned to a 40-char commit SHA. New action references in Phase 10 (`actions/cache@v4`, `marocchino/sticky-pull-request-comment@v3.0.4`) MUST be pinned to their commit SHAs before the planner ships, per RESEARCH § Security Domain V14.

### `set -euo pipefail` + `usage()` + `while [[ $# -gt 0 ]]` arg parser
**Source:** `scripts/bench/run_benchstat.sh` lines 1-42; `scripts/bench/capture_release_snapshot.sh` lines 1-52
**Apply to:** `scripts/bench/run_pr_benchmark.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 --old <path> --new <path>" >&2
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--old)
			shift
			[[ $# -gt 0 ]] || { usage; echo "missing value for --old" >&2; exit 1; }
			old_path="$1"
			;;
		# ... -h|--help ; *) error ; esac
	shift
done
```

### Step-summary-first, sticky-comment-best-effort surface
**Source:** RESEARCH Pattern 3 (no in-repo analog yet — Phase 10 introduces this surface pattern; document inline so future phases inherit it)
**Apply to:** `.github/workflows/pr-benchmark.yml` only

```yaml
- name: Append step summary
  if: always()
  run: cat pr-bench-summary/markdown.md >> "$GITHUB_STEP_SUMMARY"

- name: Post sticky PR comment
  if: always()
  continue-on-error: true                            # fork PRs lack pull-requests:write — D-19 degraded path
  uses: marocchino/sticky-pull-request-comment@<v3.0.4 SHA pinned at plan time>
  with:
    header: pr-benchmark-regression
    path: pr-bench-summary/markdown.md
```

### `if: always()` on artifact-upload steps
**Source:** `.github/workflows/benchmark-capture.yml` line 60
**Apply to:** Both new workflows. D-21 explicit. Same pattern, change retention to 14 (PR) vs 30 (release).

### Reuse `parse_benchmark_file()` and `EvidenceError`
**Source:** `scripts/bench/check_benchmark_claims.py` lines 59-60, 144-164
**Apply to:** `scripts/bench/check_pr_regression.py`

The `parse_benchmark_file()` function is the single source of truth for raw `.bench.txt` parsing. The new regression script imports it; do NOT duplicate the body. RESEARCH § Don't Hand-Roll explicitly calls this out: "Every reusable piece in Phase 10 already exists in-repo (Phase 9 helpers) or as a maintained action."

### Fixture-driven `subprocess.run` contract test
**Source:** `tests/bench/test_check_benchmark_claims.py` lines 216-238
**Apply to:** `tests/bench/test_check_pr_regression.py`

```python
result = subprocess.run(
    [sys.executable, str(SCRIPT_PATH), ...],
    cwd=REPO_ROOT,
    capture_output=True,
    text=True,
    check=False,                                      # never raise — assert returncode explicitly
)
```

---

## Project-Specific Constraints (encoded for the planner)

From global `~/.claude/CLAUDE.md` and `RESEARCH § Project Constraints`:

| Constraint | Where it lands in Phase 10 |
|---|---|
| Conventional commits (`type(scope): subject`) | Plan's commit checklist; e.g. `feat(ci): add PR benchmark regression workflow`, `feat(bench): add PR regression parser`, `test(bench): cover boundary thresholds for regression parser`, `docs(changelog): note PR benchmark regression workflow` |
| No `github.teliacompany.net` / internal references | Auto-compliance — Phase 10 has zero internal infra contact. Sanity-check: no env vars from internal repos, no hardcoded URLs |
| No "🤖 Generated with Claude Code" attribution | Plan's commit/PR checklists |
| GH issue prefixes | If any spillover issue is filed: `[BLD]` (closest match for CI), `[TST]` (parser test gap), `[ENH]` (future blocking flip). Phase 10 likely needs `[ENH]` follow-up issue for "graduate PR regression check to required status" (D-14) |
| Don't push to main without a PR | Plan ships via `gsd/phase-10-...` → PR → main. Same as every prior phase |
| Radically simple | ≤200 LOC bash orchestrator, ≤300 LOC Python parser, two thin workflow YAMLs (~70 lines each). RESEARCH "Don't Hand-Roll" table calls out every temptation |
| Self-explanatory code, sparse comments | One comment in the parser explaining `DELTA_RE`'s benchstat-format reference. No verbose docstrings |
| Use `yq` for YAML validation | Plan's commit-time check should include `yq eval '.' .github/workflows/pr-benchmark.yml >/dev/null` and same for `main-benchmark-baseline.yml` |
| `.github/actions/setup-rust` reused unchanged | Phase 10 MUST NOT edit this composite action (D-18 acknowledges the symmetric risk; Phase 10 only consumes it) |

---

## No Analog Found

| File | Role | Data Flow | Reason | Planner guidance |
|---|---|---|---|---|
| `actions/cache` integration | cache transport | event-driven | No prior workflow uses `actions/cache` in this repo (verified by grep — `benchmark-capture.yml` does not cache anything). | Use RESEARCH Pattern 2 verbatim; pin `actions/cache/save@v4` and `actions/cache/restore@v4` to their commit SHAs. |
| `marocchino/sticky-pull-request-comment` integration | PR comment surface | request-response | First introduction in this repo. | Use RESEARCH § Code Examples block; pin `@v3.0.4` to its commit SHA. Verify the SHA at plan time per security V14. |

These are the two surfaces where the planner MUST follow RESEARCH directly (Pattern 2 and Pattern 3) since no in-repo analog exists. All other surfaces have a concrete in-repo analog above.

---

## Metadata

**Analog search scope:** `.github/workflows/`, `.github/actions/`, `scripts/bench/`, `tests/bench/`, `benchmark_*_test.go`, `testdata/benchmark-results/v0.1.2/`
**Files scanned:** ~25 (workflows, composite actions, scripts, Python tests, Go benchmark families, real Phase 9 evidence)
**Pattern extraction date:** 2026-04-27
**File created:** `.planning/phases/10-lightweight-pr-benchmark-regression-signal/10-PATTERNS.md`
