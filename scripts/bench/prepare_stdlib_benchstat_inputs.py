#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import sys


FAMILY_FIXTURES = {
    "tier1": ("BenchmarkTier1FullParse", ("twitter_json", "citm_catalog_json", "canada_json")),
    "tier2": ("BenchmarkTier2Typed", ("twitter_json", "citm_catalog_json", "canada_json")),
    "tier3": ("BenchmarkTier3SelectivePlaceholder", ("twitter_json", "citm_catalog_json")),
}
METADATA_RE = re.compile(r"^(goos|goarch|pkg|cpu):\s*.+$")
BENCHMARK_RE = re.compile(r"^(Benchmark[^\s]+)-(\d+)(\s+.*)$")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Prepare same-snapshot stdlib/pure-simdjson benchstat inputs."
    )
    parser.add_argument("--source", required=True, type=pathlib.Path)
    parser.add_argument("--family", required=True, choices=tuple(FAMILY_FIXTURES))
    parser.add_argument("--base-comparator", required=True)
    parser.add_argument("--candidate-comparator", required=True)
    parser.add_argument("--left-out", required=True, type=pathlib.Path)
    parser.add_argument("--right-out", required=True, type=pathlib.Path)
    return parser.parse_args()


def read_lines(path: pathlib.Path) -> list[str]:
    try:
        return path.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as error:
        raise SystemExit(f"read {path}: {error}") from error


def normalize_rows(
    *,
    lines: list[str],
    family: str,
    comparator: str,
) -> tuple[list[str], list[str]]:
    prefix, fixtures = FAMILY_FIXTURES[family]
    expected = {f"{prefix}_{fixture}" for fixture in fixtures}
    found: set[str] = set()
    output: list[str] = []

    for line in lines:
        match = BENCHMARK_RE.match(line)
        if match is None:
            if line.startswith("Benchmark"):
                raise SystemExit(f"unparseable benchmark row: {line}")
            continue
        benchmark_name, procs, trailing = match.groups()
        suffix = f"/{comparator}"
        if not benchmark_name.endswith(suffix):
            continue
        normalized_name = benchmark_name[: -len(suffix)]
        if normalized_name not in expected:
            continue
        found.add(normalized_name)
        output.append(f"{normalized_name}-{procs}{trailing}")

    missing = sorted(expected - found)
    return output, missing


def write_output(path: pathlib.Path, metadata: list[str], rows: list[str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    try:
        path.write_text("\n".join([*metadata, *rows, ""]) , encoding="utf-8")
    except OSError as error:
        raise SystemExit(f"write {path}: {error}") from error


def rows_per_fixture(rows: list[str]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for row in rows:
        name = row.split("\t", 1)[0].rsplit("-", 1)[0]
        counts[name] = counts.get(name, 0) + 1
    return counts


def main() -> int:
    args = parse_args()
    lines = read_lines(args.source)
    metadata: list[str] = []
    seen_keys: set[str] = set()
    for line in lines:
        match = METADATA_RE.match(line)
        if match is None:
            continue
        key = match.group(1)
        if key in seen_keys:
            continue
        seen_keys.add(key)
        metadata.append(line)

    left_rows, left_missing = normalize_rows(
        lines=lines,
        family=args.family,
        comparator=args.base_comparator,
    )
    right_rows, right_missing = normalize_rows(
        lines=lines,
        family=args.family,
        comparator=args.candidate_comparator,
    )

    errors = [
        *[f"missing {args.base_comparator} row for {row}" for row in left_missing],
        *[f"missing {args.candidate_comparator} row for {row}" for row in right_missing],
    ]

    left_counts = rows_per_fixture(left_rows)
    right_counts = rows_per_fixture(right_rows)
    for name in sorted(set(left_counts) | set(right_counts)):
        if left_counts.get(name, 0) != right_counts.get(name, 0):
            errors.append(
                f"row count mismatch for {name}: "
                f"{args.base_comparator}={left_counts.get(name, 0)} "
                f"{args.candidate_comparator}={right_counts.get(name, 0)}"
            )

    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1

    write_output(args.left_out, metadata, left_rows)
    write_output(args.right_out, metadata, right_rows)
    return 0


if __name__ == "__main__":
    sys.exit(main())
