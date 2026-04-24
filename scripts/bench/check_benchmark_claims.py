#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import pathlib
import re
import statistics
import sys
from typing import Any


SNAPSHOT_FILES = (
    "phase9.bench.txt",
    "coldwarm.bench.txt",
    "tier1-diagnostics.bench.txt",
    "phase9.benchstat.txt",
    "coldwarm.benchstat.txt",
    "tier1-diagnostics.benchstat.txt",
    "tier1-vs-stdlib.benchstat.txt",
    "tier2-vs-stdlib.benchstat.txt",
    "tier3-vs-stdlib.benchstat.txt",
    "metadata.json",
)
BASELINE_FILES = (
    "phase7.bench.txt",
    "coldwarm.bench.txt",
    "tier1-diagnostics.bench.txt",
)
TOOLCHAIN_KEYS = (
    "snapshot",
    "goos",
    "goarch",
    "pkg",
    "cpu",
    "go_version",
    "rustc_version",
    "commit",
    "runner_os",
    "runner_arch",
    "captured_at_utc",
    "commands",
)
RAW_METADATA_KEYS = ("goos", "goarch", "pkg", "cpu")
TIER123_FIXTURES = ("twitter_json", "citm_catalog_json", "canada_json")
TIER3_FIXTURES = ("twitter_json", "citm_catalog_json")
DIAGNOSTIC_COMPARATORS = (
    "pure-simdjson-full",
    "pure-simdjson-parse-only",
    "pure-simdjson-materialize-only",
    "encoding-json-any-full",
)
BENCHMARK_RE = re.compile(r"^(Benchmark[^\s]+)-\d+\s+(.*)$")
METADATA_RE = re.compile(r"^(goos|goarch|pkg|cpu):\s*(.+?)\s*$")
SIGNIFICANT_WIN_RE = re.compile(r"(?<![\w.])-\d+(?:\.\d+)?%")


class EvidenceError(ValueError):
    pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate Phase 9 benchmark claim allowances from release evidence."
    )
    parser.add_argument("--baseline-dir", required=True, type=pathlib.Path)
    parser.add_argument("--snapshot-dir", required=True, type=pathlib.Path)
    parser.add_argument("--snapshot", required=True)
    parser.add_argument("--require-target", required=True)
    return parser.parse_args()


def empty_payload(snapshot: str, target: str) -> dict[str, Any]:
    goos, goarch = (target.split("/", 1) + [""])[:2] if "/" in target else (target, "")
    return {
        "snapshot": snapshot,
        "target": {
            "goos": goos,
            "goarch": goarch,
            "pkg": "",
            "cpu": "",
            "go_version": "",
            "rustc_version": "",
            "runner_os": "",
            "runner_arch": "",
            "commit": "",
            "captured_at_utc": "",
        },
        "thresholds": {
            "tier1_headline": "benchstat_significant_win_vs_encoding_json_any_every_fixture",
            "tier2_tier3": "no_material_regression_vs_linux_amd64_baseline_and_win_vs_encoding_json_struct_every_fixture",
        },
        "claims": {
            "tier1_headline_allowed": False,
            "tier2_headline_allowed": False,
            "tier3_headline_allowed": False,
            "readme_mode": "conservative_current_strengths",
        },
        "fixtures": {},
        "errors": [],
    }


def read_text(path: pathlib.Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as error:
        raise EvidenceError(f"read {path}: {error}") from error


def require_files(directory: pathlib.Path, filenames: tuple[str, ...]) -> list[str]:
    errors: list[str] = []
    for filename in filenames:
        if not (directory / filename).is_file():
            errors.append(f"missing required file: {directory / filename}")
    return errors


def extract_metrics(path: pathlib.Path, line: str, fields: list[str]) -> dict[str, float]:
    metrics: dict[str, float] = {}
    for index, token in enumerate(fields):
        if token not in {
            "ns/op",
            "B/op",
            "allocs/op",
            "native-bytes/op",
            "native-allocs/op",
            "native-live-bytes",
        }:
            continue
        if index == 0:
            raise EvidenceError(f"parse {path}: benchmark row missing value before {token}: {line}")
        raw_value = fields[index - 1].replace(",", "")
        try:
            metrics[token] = float(raw_value)
        except ValueError as error:
            raise EvidenceError(f"parse {path}: benchmark row has invalid {token} value: {line}") from error
    if "ns/op" not in metrics:
        raise EvidenceError(f"parse {path}: benchmark row missing ns/op value: {line}")
    return metrics


def parse_benchmark_file(path: pathlib.Path) -> tuple[dict[str, str], dict[str, list[dict[str, float]]]]:
    metadata: dict[str, str] = {}
    samples: dict[str, list[dict[str, float]]] = {}

    for line in read_text(path).splitlines():
        metadata_match = METADATA_RE.match(line)
        if metadata_match is not None:
            metadata[metadata_match.group(1)] = metadata_match.group(2)
            continue

        benchmark_match = BENCHMARK_RE.match(line)
        if benchmark_match is None:
            if line.startswith("Benchmark"):
                raise EvidenceError(f"parse {path}: unparseable benchmark row: {line}")
            continue

        benchmark_name = benchmark_match.group(1)
        metrics = extract_metrics(path, line, benchmark_match.group(2).split())
        samples.setdefault(benchmark_name, []).append(metrics)

    return metadata, samples


def median_ns(samples: dict[str, list[dict[str, float]]], row: str) -> float | None:
    row_samples = samples.get(row)
    if not row_samples:
        return None
    return statistics.median(sample["ns/op"] for sample in row_samples)


def require_rows(
    samples: dict[str, list[dict[str, float]]],
    rows: list[str],
    *,
    source_name: str,
) -> list[str]:
    return [f"missing required row in {source_name}: {row}" for row in rows if row not in samples]


def required_phase9_rows() -> list[str]:
    rows: list[str] = []
    for fixture in TIER123_FIXTURES:
        rows.append(f"BenchmarkTier1FullParse_{fixture}/pure-simdjson")
        rows.append(f"BenchmarkTier1FullParse_{fixture}/encoding-json-any")
        rows.append(f"BenchmarkTier2Typed_{fixture}/pure-simdjson")
        rows.append(f"BenchmarkTier2Typed_{fixture}/encoding-json-struct")
    for fixture in TIER3_FIXTURES:
        rows.append(f"BenchmarkTier3SelectivePlaceholder_{fixture}/pure-simdjson")
        rows.append(f"BenchmarkTier3SelectivePlaceholder_{fixture}/encoding-json-struct")
    return rows


def required_coldwarm_rows() -> list[str]:
    rows: list[str] = []
    for fixture in TIER123_FIXTURES:
        rows.append(f"BenchmarkColdStart_{fixture}")
        rows.append(f"BenchmarkWarm_{fixture}")
    return rows


def required_diagnostic_rows() -> list[str]:
    rows: list[str] = []
    for fixture in TIER123_FIXTURES:
        for comparator in DIAGNOSTIC_COMPARATORS:
            rows.append(f"BenchmarkTier1Diagnostics_{fixture}/{comparator}")
    return rows


def required_baseline_rows() -> list[str]:
    rows: list[str] = []
    for fixture in TIER123_FIXTURES:
        rows.append(f"BenchmarkTier2Typed_{fixture}/pure-simdjson")
    for fixture in TIER3_FIXTURES:
        rows.append(f"BenchmarkTier3SelectivePlaceholder_{fixture}/pure-simdjson")
    return rows


def load_metadata(path: pathlib.Path) -> tuple[dict[str, Any], list[str]]:
    try:
        payload = json.loads(read_text(path))
    except json.JSONDecodeError as error:
        return {}, [f"parse {path}: invalid JSON: {error}"]
    if not isinstance(payload, dict):
        return {}, [f"parse {path}: metadata.json must contain an object"]
    errors = [f"metadata.json missing required key: {key}" for key in TOOLCHAIN_KEYS if key not in payload]
    return payload, errors


def verify_metadata(
    *,
    metadata_json: dict[str, Any],
    raw_metadatas: list[tuple[str, dict[str, str]]],
    required_target: str,
    snapshot: str,
) -> list[str]:
    errors: list[str] = []
    if metadata_json.get("snapshot") != snapshot:
        errors.append(
            f"metadata.json snapshot mismatch: expected {snapshot}, got {metadata_json.get('snapshot')}"
        )
    if "/" not in required_target:
        errors.append(f"--require-target must be goos/goarch: {required_target}")
        return errors
    required_goos, required_goarch = required_target.split("/", 1)
    if metadata_json.get("goos") != required_goos or metadata_json.get("goarch") != required_goarch:
        errors.append(
            "metadata.json does not match required target "
            f"{required_target}: got {metadata_json.get('goos')}/{metadata_json.get('goarch')}"
        )
    for source_name, raw_metadata in raw_metadatas:
        for key in RAW_METADATA_KEYS:
            raw_value = raw_metadata.get(key)
            metadata_value = metadata_json.get(key)
            if raw_value is None:
                errors.append(f"{source_name} missing metadata key: {key}")
            elif raw_value != metadata_value:
                errors.append(
                    f"{source_name} metadata {key} mismatch: raw={raw_value} metadata.json={metadata_value}"
                )
    return errors


def verify_baseline_target_matches_snapshot(
    *,
    baseline_metadatas: list[tuple[str, dict[str, str]]],
    snapshot_metadata: dict[str, str],
) -> list[str]:
    errors: list[str] = []
    for source_name, baseline_metadata in baseline_metadatas:
        for key in RAW_METADATA_KEYS:
            baseline_value = baseline_metadata.get(key)
            snapshot_value = snapshot_metadata.get(key)
            if baseline_value is None:
                errors.append(f"{source_name} missing metadata key: {key}")
                continue
            if snapshot_value is None:
                errors.append(f"snapshot phase9.bench.txt missing metadata key: {key}")
                continue
            if baseline_value != snapshot_value:
                errors.append(
                    "baseline target metadata mismatch: "
                    f"{source_name} {key}={baseline_value} snapshot {key}={snapshot_value}"
                )
    return errors


def has_significant_win(benchstat_text: str, row: str) -> bool:
    row_aliases = (row, row.removeprefix("Benchmark"))
    for line in benchstat_text.splitlines():
        if not any(row_alias in line for row_alias in row_aliases):
            continue
        if "~" in line:
            return False
        if SIGNIFICANT_WIN_RE.search(line):
            return True
    return False


def ratio(winner_ns: float, loser_ns: float) -> float:
    if winner_ns <= 0:
        return 0.0
    return loser_ns / winner_ns


def tier1_status(
    phase9_samples: dict[str, list[dict[str, float]]],
    benchstat_text_value: str,
) -> tuple[bool, dict[str, Any]]:
    fixtures: dict[str, Any] = {}
    allowed = True
    for fixture in TIER123_FIXTURES:
        row = f"BenchmarkTier1FullParse_{fixture}"
        pure = median_ns(phase9_samples, f"{row}/pure-simdjson")
        stdlib = median_ns(phase9_samples, f"{row}/encoding-json-any")
        faster = pure is not None and stdlib is not None and pure < stdlib
        significant = has_significant_win(benchstat_text_value, row)
        row_allowed = bool(faster and significant)
        fixtures[fixture] = {
            "pure_ns_op": pure,
            "stdlib_ns_op": stdlib,
            "ratio_vs_encoding_json_any": ratio(pure, stdlib) if pure and stdlib else None,
            "median_win": bool(faster),
            "benchstat_significant_win": significant,
            "allowed": row_allowed,
        }
        allowed = allowed and row_allowed
    return allowed, fixtures


def tier_status(
    *,
    tier_name: str,
    benchmark_prefix: str,
    fixtures: tuple[str, ...],
    phase9_samples: dict[str, list[dict[str, float]]],
    baseline_samples: dict[str, list[dict[str, float]]],
    benchstat_text_value: str,
    errors: list[str],
    compare_baseline: bool,
) -> tuple[bool, dict[str, Any]]:
    fixture_statuses: dict[str, Any] = {}
    allowed = True
    for fixture in fixtures:
        row = f"{benchmark_prefix}_{fixture}"
        pure_row = f"{row}/pure-simdjson"
        struct_row = f"{row}/encoding-json-struct"
        old = median_ns(baseline_samples, pure_row)
        pure = median_ns(phase9_samples, pure_row)
        stdlib = median_ns(phase9_samples, struct_row)
        no_regression = compare_baseline and old is not None and pure is not None and pure <= old
        median_win = pure is not None and stdlib is not None and pure < stdlib
        significant = has_significant_win(benchstat_text_value, row)
        row_allowed = bool(no_regression and median_win and significant)
        if compare_baseline and old is not None and pure is not None and pure > old:
            errors.append(f"{tier_name} regression for {fixture}: old={old:.2f}ns/op new={pure:.2f}ns/op")
        fixture_statuses[fixture] = {
            "baseline_pure_ns_op": old,
            "pure_ns_op": pure,
            "stdlib_ns_op": stdlib,
            "ratio_vs_encoding_json_struct": ratio(pure, stdlib) if pure and stdlib else None,
            "no_material_regression_vs_v0.1.1": bool(no_regression),
            "median_win": bool(median_win),
            "benchstat_significant_win": significant,
            "allowed": row_allowed,
        }
        allowed = allowed and row_allowed
    return allowed, fixture_statuses


def choose_readme_mode(
    *,
    tier1_allowed: bool,
    tier2_allowed: bool,
    tier3_allowed: bool,
) -> str:
    if tier1_allowed:
        return "tier1_headline"
    if tier2_allowed and tier3_allowed:
        return "tier1_improved_but_tier2_tier3_headline"
    return "conservative_current_strengths"


def generate_payload(args: argparse.Namespace) -> tuple[dict[str, Any], int]:
    payload = empty_payload(args.snapshot, args.require_target)
    errors: list[str] = payload["errors"]
    errors.extend(require_files(args.baseline_dir, BASELINE_FILES))
    errors.extend(require_files(args.snapshot_dir, SNAPSHOT_FILES))
    if errors:
        return payload, 1

    try:
        baseline_metadata, baseline_phase7 = parse_benchmark_file(args.baseline_dir / "phase7.bench.txt")
        baseline_coldwarm_metadata, _baseline_coldwarm = parse_benchmark_file(
            args.baseline_dir / "coldwarm.bench.txt"
        )
        baseline_diagnostics_metadata, _baseline_diagnostics = parse_benchmark_file(
            args.baseline_dir / "tier1-diagnostics.bench.txt"
        )
        phase9_metadata, phase9_samples = parse_benchmark_file(args.snapshot_dir / "phase9.bench.txt")
        coldwarm_metadata, coldwarm_samples = parse_benchmark_file(args.snapshot_dir / "coldwarm.bench.txt")
        diagnostics_metadata, diagnostic_samples = parse_benchmark_file(
            args.snapshot_dir / "tier1-diagnostics.bench.txt"
        )
    except EvidenceError as error:
        errors.append(str(error))
        return payload, 1

    metadata_json, metadata_errors = load_metadata(args.snapshot_dir / "metadata.json")
    errors.extend(metadata_errors)
    if metadata_json:
        errors.extend(
            verify_metadata(
                metadata_json=metadata_json,
                raw_metadatas=[
                    ("phase9.bench.txt", phase9_metadata),
                    ("coldwarm.bench.txt", coldwarm_metadata),
                    ("tier1-diagnostics.bench.txt", diagnostics_metadata),
                ],
                required_target=args.require_target,
                snapshot=args.snapshot,
            )
        )
        payload["target"] = {
            "goos": metadata_json.get("goos", ""),
            "goarch": metadata_json.get("goarch", ""),
            "pkg": metadata_json.get("pkg", ""),
            "cpu": metadata_json.get("cpu", ""),
            "go_version": metadata_json.get("go_version", ""),
            "rustc_version": metadata_json.get("rustc_version", ""),
            "runner_os": metadata_json.get("runner_os", ""),
            "runner_arch": metadata_json.get("runner_arch", ""),
            "commit": metadata_json.get("commit", ""),
            "captured_at_utc": metadata_json.get("captured_at_utc", ""),
        }

    errors.extend(require_rows(phase9_samples, required_phase9_rows(), source_name="phase9.bench.txt"))
    errors.extend(require_rows(coldwarm_samples, required_coldwarm_rows(), source_name="coldwarm.bench.txt"))
    errors.extend(
        require_rows(diagnostic_samples, required_diagnostic_rows(), source_name="tier1-diagnostics.bench.txt")
    )
    errors.extend(require_rows(baseline_phase7, required_baseline_rows(), source_name="baseline phase7.bench.txt"))

    baseline_target_errors = verify_baseline_target_matches_snapshot(
        baseline_metadatas=[
            ("baseline phase7.bench.txt", baseline_metadata),
            ("baseline coldwarm.bench.txt", baseline_coldwarm_metadata),
            ("baseline tier1-diagnostics.bench.txt", baseline_diagnostics_metadata),
        ],
        snapshot_metadata=phase9_metadata,
    )
    errors.extend(baseline_target_errors)
    compare_baseline = not baseline_target_errors

    tier1_benchstat = read_text(args.snapshot_dir / "tier1-vs-stdlib.benchstat.txt")
    tier2_benchstat = read_text(args.snapshot_dir / "tier2-vs-stdlib.benchstat.txt")
    tier3_benchstat = read_text(args.snapshot_dir / "tier3-vs-stdlib.benchstat.txt")

    tier1_allowed, tier1_fixtures = tier1_status(phase9_samples, tier1_benchstat)
    tier2_allowed, tier2_fixtures = tier_status(
        tier_name="tier2",
        benchmark_prefix="BenchmarkTier2Typed",
        fixtures=TIER123_FIXTURES,
        phase9_samples=phase9_samples,
        baseline_samples=baseline_phase7,
        benchstat_text_value=tier2_benchstat,
        errors=errors,
        compare_baseline=compare_baseline,
    )
    tier3_allowed, tier3_fixtures = tier_status(
        tier_name="tier3",
        benchmark_prefix="BenchmarkTier3SelectivePlaceholder",
        fixtures=TIER3_FIXTURES,
        phase9_samples=phase9_samples,
        baseline_samples=baseline_phase7,
        benchstat_text_value=tier3_benchstat,
        errors=errors,
        compare_baseline=compare_baseline,
    )

    payload["claims"] = {
        "tier1_headline_allowed": tier1_allowed,
        "tier2_headline_allowed": tier2_allowed,
        "tier3_headline_allowed": tier3_allowed,
        "readme_mode": choose_readme_mode(
            tier1_allowed=tier1_allowed,
            tier2_allowed=tier2_allowed,
            tier3_allowed=tier3_allowed,
        ),
    }
    payload["fixtures"] = {
        "tier1": tier1_fixtures,
        "tier2": tier2_fixtures,
        "tier3": tier3_fixtures,
    }

    if errors:
        payload["claims"] = {
            "tier1_headline_allowed": False,
            "tier2_headline_allowed": False,
            "tier3_headline_allowed": False,
            "readme_mode": "conservative_current_strengths",
        }

    return payload, 1 if errors else 0


def main() -> int:
    args = parse_args()
    payload, exit_code = generate_payload(args)
    print(json.dumps(payload, indent=2, sort_keys=True))
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
