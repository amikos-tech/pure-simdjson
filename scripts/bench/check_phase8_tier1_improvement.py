#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import statistics
import sys


MIN_DELTA_FRACTION = 0.10
METADATA_KEYS = ("goos", "goarch", "pkg", "cpu")
REQUIRED_BENCHMARKS = (
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-materialize-only",
    "BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-materialize-only",
    "BenchmarkTier1Diagnostics_canada_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_canada_json/pure-simdjson-materialize-only",
)
METADATA_RE = re.compile(r"^(goos|goarch|pkg|cpu):\s*(.+?)\s*$")
BENCHMARK_RE = re.compile(
    r"^(BenchmarkTier1Diagnostics_"
    r"(?:twitter_json|citm_catalog_json|canada_json)/"
    r"pure-simdjson-(?:full|materialize-only))-\d+\s+(.*)$"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verify Phase 8 Tier 1 benchmark improvement against Phase 7."
    )
    parser.add_argument("--old", required=True, type=pathlib.Path)
    parser.add_argument("--new", required=True, type=pathlib.Path)
    return parser.parse_args()


def parse_benchmark_file(
    path: pathlib.Path,
) -> tuple[dict[str, str], dict[str, list[float]]]:
    metadata: dict[str, str] = {}
    samples: dict[str, list[float]] = {}

    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as error:
        raise SystemExit(f"read {path}: {error}") from error

    for line in lines:
        metadata_match = METADATA_RE.match(line)
        if metadata_match is not None:
            metadata[metadata_match.group(1)] = metadata_match.group(2)
            continue

        benchmark_match = BENCHMARK_RE.match(line)
        if benchmark_match is None:
            continue

        benchmark_name = benchmark_match.group(1)
        trailing_fields = benchmark_match.group(2).split()
        ns_value = extract_ns_per_op(path, line, trailing_fields)
        samples.setdefault(benchmark_name, []).append(ns_value)

    return metadata, samples


def extract_ns_per_op(path: pathlib.Path, line: str, trailing_fields: list[str]) -> float:
    for index, token in enumerate(trailing_fields):
        if token != "ns/op":
            continue
        if index == 0:
            break
        numeric_token = trailing_fields[index - 1].replace(",", "")
        try:
            return float(numeric_token)
        except ValueError as error:
            raise SystemExit(
                f"parse {path}: benchmark row has invalid ns/op value: {line}"
            ) from error
    raise SystemExit(f"parse {path}: benchmark row missing ns/op value: {line}")


def format_ns(value: float) -> str:
    return f"{value:.2f}"


def format_delta(delta_fraction: float) -> str:
    return f"{delta_fraction * 100.0:.2f}%"


def print_metadata_mismatch(
    *,
    key: str,
    old_value: str | None,
    new_value: str | None,
) -> None:
    normalized_old = "missing" if old_value is None else old_value
    normalized_new = "missing" if new_value is None else new_value
    print(
        f"FAIL metadata old_{key}={normalized_old} "
        f"new_{key}={normalized_new} reason=metadata-mismatch"
    )


def compare_metadata(old_metadata: dict[str, str], new_metadata: dict[str, str]) -> bool:
    for key in METADATA_KEYS:
        old_value = old_metadata.get(key)
        new_value = new_metadata.get(key)
        if old_value != new_value:
            print_metadata_mismatch(key=key, old_value=old_value, new_value=new_value)
            return False
    return True


def compare_benchmarks(
    old_samples: dict[str, list[float]],
    new_samples: dict[str, list[float]],
) -> bool:
    success = True
    for benchmark_name in REQUIRED_BENCHMARKS:
        old_values = old_samples.get(benchmark_name)
        new_values = new_samples.get(benchmark_name)

        if not old_values:
            print(
                f"FAIL {benchmark_name} old=missing new="
                f"{format_ns(statistics.median(new_values)) if new_values else 'missing'} "
                "delta=n/a reason=missing-old-row"
            )
            success = False
            continue

        if not new_values:
            print(
                f"FAIL {benchmark_name} old={format_ns(statistics.median(old_values))} "
                "new=missing delta=n/a reason=missing-new-row"
            )
            success = False
            continue

        old_median = statistics.median(old_values)
        new_median = statistics.median(new_values)
        delta_fraction = (old_median - new_median) / old_median

        if new_median > old_median:
            print(
                f"FAIL {benchmark_name} old={format_ns(old_median)} "
                f"new={format_ns(new_median)} delta={format_delta(delta_fraction)} "
                "reason=regressed"
            )
            success = False
            continue

        if delta_fraction < MIN_DELTA_FRACTION:
            print(
                f"FAIL {benchmark_name} old={format_ns(old_median)} "
                f"new={format_ns(new_median)} delta={format_delta(delta_fraction)} "
                "reason=below-threshold"
            )
            success = False
            continue

        print(
            f"PASS {benchmark_name} old={format_ns(old_median)} "
            f"new={format_ns(new_median)} delta={format_delta(delta_fraction)} "
            f"threshold={MIN_DELTA_FRACTION * 100.0:.2f}%"
        )

    return success


def main() -> int:
    args = parse_args()

    old_metadata, old_samples = parse_benchmark_file(args.old)
    new_metadata, new_samples = parse_benchmark_file(args.new)

    if not compare_metadata(old_metadata, new_metadata):
        return 1

    if not compare_benchmarks(old_samples, new_samples):
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
