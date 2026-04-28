#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import pathlib
import re
import sys
from typing import Any


sys.path.insert(0, str(pathlib.Path(__file__).parent))
from check_benchmark_claims import EvidenceError  # noqa: E402


# Benchstat delta - see https://pkg.go.dev/golang.org/x/perf/cmd/benchstat for output format.
DELTA_RE = re.compile(r"(?<![\w.])([+-])(\d+(?:\.\d+)?)%\s+\(p=(\d+\.\d+)\s+n=\d+\)")
# Benchstat can emit a row without a p-value when variance is too high or
# sample data is otherwise statistically inconclusive. That is not actionable
# as a regression signal, so advisory PR checks skip it instead of failing.
INCONCLUSIVE_RE = re.compile(r"(?:±|\+/-)\s*∞|~\s+\(p=")
# Must stay in sync with PR_BENCH_REGEX in scripts/bench/run_pr_benchmark.sh.
ROW_PREFIX_RE = re.compile(r"^\s*(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_\S+")
METRIC_HEADER_RE = re.compile(
    r"(?:│|\|)\s*(sec/op|B/s|B/op|allocs/op|native-allocs/op|native-bytes/op|native-live-bytes)\s*(?:│|\|)"
)
# Identifies metric header rows even when the specific metric is unsupported.
ANY_METRIC_HEADER_RE = re.compile(r"(?:│|\|).*\bvs base\b.*(?:│|\|)?")
# The advisory signal is intentionally small: flag only material slowdown >=5%
# with p<0.05, matching the PR comment's human-readable default thresholds.
DEFAULT_THRESHOLD_PCT = 5.0
DEFAULT_P_MAX = 0.05
BLOCKING_FLIP_ENV = "REQUIRE_NO_REGRESSION"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Detect advisory pull-request benchmark regressions from benchstat output."
    )
    parser.add_argument("--benchstat-output", type=pathlib.Path)
    parser.add_argument("--summary-out", required=True, type=pathlib.Path)
    parser.add_argument("--markdown-out", required=True, type=pathlib.Path)
    parser.add_argument("--threshold-pct", type=float, default=DEFAULT_THRESHOLD_PCT)
    parser.add_argument("--p-max", type=float, default=DEFAULT_P_MAX)
    parser.add_argument("--no-baseline", action="store_true")
    return parser.parse_args()


def read_text(path: pathlib.Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as error:
        raise EvidenceError(f"read {path}: {error}") from error


def parse_benchstat_for_regressions(
    text: str,
    threshold_pct: float,
    p_max: float,
) -> list[dict[str, Any]]:
    flagged: list[dict[str, Any]] = []
    seen_any_sec_op_row = False
    current_metric: str | None = None

    for line in text.splitlines():
        header = METRIC_HEADER_RE.search(line)
        if header is not None:
            current_metric = header.group(1)
            continue
        if ANY_METRIC_HEADER_RE.search(line):
            current_metric = None
            continue

        if not ROW_PREFIX_RE.match(line):
            continue
        if current_metric is None:
            raise EvidenceError(f"unrecognized metric section for benchmark row: {line}")
        if current_metric != "sec/op":
            continue

        seen_any_sec_op_row = True
        if INCONCLUSIVE_RE.search(line):
            continue

        match = DELTA_RE.search(line)
        if match is None:
            raise EvidenceError(f"malformed sec/op benchstat row: {line}")

        sign, pct_str, p_str = match.groups()
        if sign != "+":
            continue

        pct = float(pct_str)
        p_value = float(p_str)
        if pct >= threshold_pct and p_value < p_max:
            flagged.append(
                {
                    "row": line.strip().split(maxsplit=1)[0],
                    "delta_pct": pct,
                    "p_value": p_value,
                    "raw_line": line.strip(),
                }
            )

    if not seen_any_sec_op_row:
        raise EvidenceError("no sec/op benchmark rows found in benchstat output")

    return flagged


def build_summary(
    *,
    regression: bool,
    bypassed: bool,
    threshold_pct: float,
    p_max: float,
    flagged_rows: list[dict[str, Any]],
) -> dict[str, Any]:
    return {
        "regression": regression,
        "bypassed": bypassed,
        "threshold_pct": threshold_pct,
        "p_max": p_max,
        "flagged_rows": flagged_rows,
        "thresholds_source": BLOCKING_FLIP_ENV,
    }


def render_markdown(summary: dict[str, Any]) -> str:
    threshold_pct = summary["threshold_pct"]
    p_max = summary["p_max"]

    if summary["bypassed"]:
        return (
            "## PR Benchmark - advisory bypass\n\n"
            "No baseline cache was available, so this run captured head benchmark evidence "
            "without comparing regressions.\n\n"
            f"Threshold when a baseline exists: >= {threshold_pct:.2f}% slower and p < {p_max:.3f}.\n"
        )

    flagged_rows = summary["flagged_rows"]
    if not flagged_rows:
        return (
            "## PR Benchmark - no regressions\n\n"
            f"Threshold: >= {threshold_pct:.2f}% slower and p < {p_max:.3f}.\n\n"
            "_This check is advisory; the PR is not blocked._\n"
        )

    lines = [
        f"## PR Benchmark - {len(flagged_rows)} regression(s) flagged (advisory)",
        "",
        f"Threshold: >= {threshold_pct:.2f}% slower and p < {p_max:.3f}.",
        "",
        "| Row | Delta | p-value |",
        "|-----|-------|---------|",
    ]
    for row in flagged_rows:
        lines.append(f"| `{row['row']}` | +{row['delta_pct']:.2f}% | {row['p_value']:.3f} |")
    lines.extend(["", "_This check is advisory; the PR is not blocked._", ""])
    return "\n".join(lines)


def write_outputs(summary: dict[str, Any], summary_out: pathlib.Path, markdown_out: pathlib.Path) -> None:
    summary_out.parent.mkdir(parents=True, exist_ok=True)
    markdown_out.parent.mkdir(parents=True, exist_ok=True)
    summary_out.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    markdown_out.write_text(render_markdown(summary), encoding="utf-8")


def main() -> int:
    args = parse_args()

    if args.no_baseline:
        summary = build_summary(
            regression=False,
            bypassed=True,
            threshold_pct=args.threshold_pct,
            p_max=args.p_max,
            flagged_rows=[],
        )
        write_outputs(summary, args.summary_out, args.markdown_out)
        return 0

    if args.benchstat_output is None:
        print("--benchstat-output is required unless --no-baseline is set", file=sys.stderr)
        return 1

    try:
        flagged = parse_benchstat_for_regressions(
            read_text(args.benchstat_output),
            args.threshold_pct,
            args.p_max,
        )
    except EvidenceError as error:
        print(f"benchmark regression evidence error: {error}", file=sys.stderr)
        return 1

    summary = build_summary(
        regression=bool(flagged),
        bypassed=False,
        threshold_pct=args.threshold_pct,
        p_max=args.p_max,
        flagged_rows=flagged,
    )
    write_outputs(summary, args.summary_out, args.markdown_out)

    require_no_regression = os.environ.get(BLOCKING_FLIP_ENV, "false").lower() == "true"
    return 1 if require_no_regression and summary["regression"] else 0


if __name__ == "__main__":
    sys.exit(main())
