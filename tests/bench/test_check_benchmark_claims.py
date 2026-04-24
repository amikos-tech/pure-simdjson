#!/usr/bin/env python3

from __future__ import annotations

import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "check_benchmark_claims.py"
SNAPSHOT = "v0.1.2"
FIXTURES = ("twitter_json", "citm_catalog_json", "canada_json")
TIER3_FIXTURES = ("twitter_json", "citm_catalog_json")
METADATA = {
    "goos": "linux",
    "goarch": "amd64",
    "pkg": "github.com/amikos-tech/pure-simdjson",
    "cpu": "Synthetic CPU",
}
TOOLCHAIN_METADATA = {
    "snapshot": SNAPSHOT,
    "goos": "linux",
    "goarch": "amd64",
    "pkg": "github.com/amikos-tech/pure-simdjson",
    "cpu": "Synthetic CPU",
    "go_version": "go1.24.0",
    "rustc_version": "rustc 1.85.0",
    "commit": "0123456789abcdef",
    "runner_os": "Linux",
    "runner_arch": "X64",
    "captured_at_utc": "2026-04-24T00:00:00Z",
    "commands": ["synthetic"],
}


def bench_header(metadata: dict[str, str] | None = None) -> list[str]:
    values = METADATA if metadata is None else metadata
    return [f"{key}: {value}" for key, value in values.items()]


def bench_row(name: str, ns: float, *, metric: str = "ns/op") -> str:
    return (
        f"{name}-8\t10\t{ns:.2f} {metric}\t1.00 MB/s\t"
        "64 B/op\t1 allocs/op\t2 native-bytes/op\t3 native-allocs/op\t"
        "4 native-live-bytes"
    )


def build_phase9_text(
    *,
    tier1_pure: float = 80.0,
    tier1_any: float = 100.0,
    tier2_pure: float = 60.0,
    tier2_struct: float = 100.0,
    tier3_pure: float = 50.0,
    tier3_struct: float = 90.0,
    metadata: dict[str, str] | None = None,
    malformed_row: str | None = None,
) -> str:
    lines = bench_header(metadata)
    for fixture in FIXTURES:
        lines.append(bench_row(f"BenchmarkTier1FullParse_{fixture}/pure-simdjson", tier1_pure))
        lines.append(bench_row(f"BenchmarkTier1FullParse_{fixture}/encoding-json-any", tier1_any))
        lines.append(bench_row(f"BenchmarkTier2Typed_{fixture}/pure-simdjson", tier2_pure))
        lines.append(bench_row(f"BenchmarkTier2Typed_{fixture}/encoding-json-struct", tier2_struct))
    for fixture in TIER3_FIXTURES:
        lines.append(bench_row(f"BenchmarkTier3SelectivePlaceholder_{fixture}/pure-simdjson", tier3_pure))
        lines.append(bench_row(f"BenchmarkTier3SelectivePlaceholder_{fixture}/encoding-json-struct", tier3_struct))
    if malformed_row is not None:
        lines.append(malformed_row)
    lines.append("PASS")
    return "\n".join(lines) + "\n"


def build_baseline_phase7_text(
    *,
    tier2_ns: float = 75.0,
    tier3_ns: float = 65.0,
    metadata: dict[str, str] | None = None,
) -> str:
    lines = bench_header(metadata)
    for fixture in FIXTURES:
        lines.append(bench_row(f"BenchmarkTier2Typed_{fixture}/pure-simdjson", tier2_ns))
    for fixture in TIER3_FIXTURES:
        lines.append(bench_row(f"BenchmarkTier3SelectivePlaceholder_{fixture}/pure-simdjson", tier3_ns))
    lines.append("PASS")
    return "\n".join(lines) + "\n"


def build_coldwarm_text(metadata: dict[str, str] | None = None) -> str:
    lines = bench_header(metadata)
    for fixture in FIXTURES:
        lines.append(bench_row(f"BenchmarkColdStart_{fixture}", 200.0))
        lines.append(bench_row(f"BenchmarkWarm_{fixture}", 100.0))
    lines.append("PASS")
    return "\n".join(lines) + "\n"


def build_diagnostics_text(metadata: dict[str, str] | None = None) -> str:
    lines = bench_header(metadata)
    for fixture in FIXTURES:
        for comparator in (
            "pure-simdjson-full",
            "pure-simdjson-parse-only",
            "pure-simdjson-materialize-only",
            "encoding-json-any-full",
        ):
            lines.append(bench_row(f"BenchmarkTier1Diagnostics_{fixture}/{comparator}", 100.0))
    lines.append("PASS")
    return "\n".join(lines) + "\n"


def benchstat_text(rows: list[str], *, significant: bool = True) -> str:
    marker = "-20.00%" if significant else "~"
    return "\n".join(
        [
            "goos: linux",
            "goarch: amd64",
            "pkg: github.com/amikos-tech/pure-simdjson",
            "cpu: Synthetic CPU",
            "              │ old ns/op │ new ns/op │   delta │",
            *[f"{row}   100.0 ± 1%   80.0 ± 1%   {marker}" for row in rows],
        ]
    ) + "\n"


class CheckBenchmarkClaimsTests(unittest.TestCase):
    def write_evidence(
        self,
        root: pathlib.Path,
        *,
        metadata: dict[str, str] | None = None,
        toolchain_metadata: dict[str, object] | None = None,
        phase9_text: str | None = None,
        tier1_benchstat_significant: bool = True,
        tier2_benchstat_significant: bool = True,
        tier3_benchstat_significant: bool = True,
        tier2_snapshot_ns: float = 60.0,
        tier3_snapshot_ns: float = 50.0,
        baseline_metadata: dict[str, str] | None = None,
    ) -> tuple[pathlib.Path, pathlib.Path]:
        baseline = root / "baseline"
        snapshot = root / "snapshot"
        baseline.mkdir()
        snapshot.mkdir()

        (baseline / "phase7.bench.txt").write_text(
            build_baseline_phase7_text(metadata=baseline_metadata),
            encoding="utf-8",
        )
        (baseline / "coldwarm.bench.txt").write_text(
            build_coldwarm_text(baseline_metadata),
            encoding="utf-8",
        )
        (baseline / "tier1-diagnostics.bench.txt").write_text(
            build_diagnostics_text(baseline_metadata),
            encoding="utf-8",
        )

        if phase9_text is None:
            phase9_text = build_phase9_text(
                metadata=metadata,
                tier2_pure=tier2_snapshot_ns,
                tier3_pure=tier3_snapshot_ns,
            )
        (snapshot / "phase9.bench.txt").write_text(phase9_text, encoding="utf-8")
        (snapshot / "coldwarm.bench.txt").write_text(build_coldwarm_text(metadata), encoding="utf-8")
        (snapshot / "tier1-diagnostics.bench.txt").write_text(build_diagnostics_text(metadata), encoding="utf-8")
        for name in ("phase9.benchstat.txt", "coldwarm.benchstat.txt", "tier1-diagnostics.benchstat.txt"):
            (snapshot / name).write_text(benchstat_text(["BenchmarkPlaceholder"]), encoding="utf-8")
        (snapshot / "tier1-vs-stdlib.benchstat.txt").write_text(
            benchstat_text(
                [f"BenchmarkTier1FullParse_{fixture}" for fixture in FIXTURES],
                significant=tier1_benchstat_significant,
            ),
            encoding="utf-8",
        )
        (snapshot / "tier2-vs-stdlib.benchstat.txt").write_text(
            benchstat_text(
                [f"BenchmarkTier2Typed_{fixture}" for fixture in FIXTURES],
                significant=tier2_benchstat_significant,
            ),
            encoding="utf-8",
        )
        (snapshot / "tier3-vs-stdlib.benchstat.txt").write_text(
            benchstat_text(
                [f"BenchmarkTier3SelectivePlaceholder_{fixture}" for fixture in TIER3_FIXTURES],
                significant=tier3_benchstat_significant,
            ),
            encoding="utf-8",
        )
        metadata_payload = TOOLCHAIN_METADATA if toolchain_metadata is None else toolchain_metadata
        (snapshot / "metadata.json").write_text(
            json.dumps(metadata_payload, sort_keys=True),
            encoding="utf-8",
        )
        return baseline, snapshot

    def run_gate(self, baseline: pathlib.Path, snapshot: pathlib.Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT_PATH),
                "--baseline-dir",
                str(baseline),
                "--snapshot-dir",
                str(snapshot),
                "--snapshot",
                SNAPSHOT,
                "--require-target",
                "linux/amd64",
            ],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )

    def parse_stdout(self, result: subprocess.CompletedProcess[str]) -> dict[str, object]:
        self.assertTrue(result.stdout, result.stderr)
        return json.loads(result.stdout)

    def test_all_complete_significant_evidence_allows_tier1_headline(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(pathlib.Path(temp_dir))
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["claims"]["tier1_headline_allowed"])
        self.assertTrue(payload["claims"]["tier2_headline_allowed"])
        self.assertTrue(payload["claims"]["tier3_headline_allowed"])
        self.assertEqual(payload["claims"]["readme_mode"], "tier1_headline")
        self.assertEqual(payload["target"]["goos"], "linux")
        self.assertEqual(payload["target"]["goarch"], "amd64")
        self.assertEqual(payload["target"]["pkg"], "github.com/amikos-tech/pure-simdjson")

    def test_tier1_improved_but_not_faster_than_stdlib_uses_tier2_tier3_mode(self) -> None:
        phase9_text = build_phase9_text(tier1_pure=120.0, tier1_any=100.0)
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(pathlib.Path(temp_dir), phase9_text=phase9_text)
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["claims"]["tier1_headline_allowed"])
        self.assertEqual(payload["claims"]["readme_mode"], "tier1_improved_but_tier2_tier3_headline")

    def test_noisy_stdlib_comparison_keeps_errors_empty_and_uses_conservative_mode(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(
                pathlib.Path(temp_dir),
                tier1_benchstat_significant=False,
            )
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(payload["errors"], [])
        self.assertEqual(payload["claims"]["readme_mode"], "conservative_current_strengths")

    def test_missing_required_files_returns_machine_readable_error(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(pathlib.Path(temp_dir))
            (snapshot / "tier3-vs-stdlib.benchstat.txt").unlink()
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(payload["errors"])
        self.assertIn("tier3-vs-stdlib.benchstat.txt", payload["errors"][0])

    def test_require_target_rejects_non_linux_amd64_metadata(self) -> None:
        metadata = dict(METADATA)
        metadata["goos"] = "darwin"
        toolchain = dict(TOOLCHAIN_METADATA)
        toolchain["goos"] = "darwin"
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(
                pathlib.Path(temp_dir),
                metadata=metadata,
                toolchain_metadata=toolchain,
            )
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any("required target linux/amd64" in error for error in payload["errors"]))

    def test_malformed_benchmark_rows_fail_closed(self) -> None:
        phase9_text = build_phase9_text(
            malformed_row="BenchmarkTier1FullParse_twitter_json/pure-simdjson-8\t10\tbad ns/op"
        )
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(pathlib.Path(temp_dir), phase9_text=phase9_text)
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any("invalid ns/op" in error for error in payload["errors"]))

    def test_tier2_or_tier3_regression_fails_nonzero(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(pathlib.Path(temp_dir), tier2_snapshot_ns=90.0)
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(payload["claims"]["tier2_headline_allowed"])
        self.assertTrue(any("tier2" in error and "regression" in error for error in payload["errors"]))

    def test_cross_platform_baseline_fails_with_metadata_mismatch_not_regression(self) -> None:
        baseline_metadata = {
            "goos": "darwin",
            "goarch": "arm64",
            "pkg": "github.com/amikos-tech/pure-simdjson",
            "cpu": "Apple M3 Max",
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            baseline, snapshot = self.write_evidence(
                pathlib.Path(temp_dir),
                baseline_metadata=baseline_metadata,
                tier2_snapshot_ns=90.0,
                tier3_snapshot_ns=90.0,
            )
            result = self.run_gate(baseline, snapshot)

        payload = self.parse_stdout(result)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(
            any("baseline target metadata mismatch" in error for error in payload["errors"])
        )
        self.assertFalse(any("regression for" in error for error in payload["errors"]))


if __name__ == "__main__":
    unittest.main()
