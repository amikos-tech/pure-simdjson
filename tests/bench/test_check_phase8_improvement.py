#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "check_phase8_tier1_improvement.py"
REQUIRED_ROWS = (
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-materialize-only",
    "BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-materialize-only",
    "BenchmarkTier1Diagnostics_canada_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_canada_json/pure-simdjson-materialize-only",
)
BASE_METADATA = {
    "goos": "darwin",
    "goarch": "arm64",
    "pkg": "github.com/amikos-tech/pure-simdjson",
    "cpu": "Apple M3 Max",
}


def make_samples(ns_value: float) -> list[float]:
    return [ns_value] * 5


def build_benchmark_text(
    *,
    metadata: dict[str, str],
    samples_by_row: dict[str, list[float]],
) -> str:
    lines = [f"{key}: {value}" for key, value in metadata.items()]
    for row_name, samples in samples_by_row.items():
        for run_index, ns_value in enumerate(samples, start=1):
            lines.append(
                f"{row_name}-16\t{run_index}\t{ns_value:.2f} ns/op\t"
                f"{1000.0 / ns_value:.2f} MB/s\t0 B/op\t0 allocs/op"
            )
    lines.append(
        "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-parse-only-16\t1\t"
        "25.00 ns/op\t40.00 MB/s\t0 B/op\t0 allocs/op"
    )
    lines.append("PASS")
    return "\n".join(lines) + "\n"


class CheckPhase8ImprovementTests(unittest.TestCase):
    def write_benchmark_file(
        self,
        directory: pathlib.Path,
        name: str,
        *,
        metadata: dict[str, str] | None = None,
        samples_by_row: dict[str, list[float]] | None = None,
    ) -> pathlib.Path:
        metadata = dict(BASE_METADATA if metadata is None else metadata)
        if samples_by_row is None:
            samples_by_row = {
                row_name: make_samples(100.0)
                for row_name in REQUIRED_ROWS
            }
        path = directory / name
        path.write_text(
            build_benchmark_text(metadata=metadata, samples_by_row=samples_by_row),
            encoding="utf-8",
        )
        return path

    def run_script(
        self,
        *,
        old_path: pathlib.Path,
        new_path: pathlib.Path,
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT_PATH),
                "--old",
                str(old_path),
                "--new",
                str(new_path),
            ],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_accepts_exact_ten_percent_improvement(self) -> None:
        new_samples = {
            row_name: make_samples(90.0)
            for row_name in REQUIRED_ROWS
        }

        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            old_path = self.write_benchmark_file(directory, "old.bench.txt")
            new_path = self.write_benchmark_file(
                directory,
                "new.bench.txt",
                samples_by_row=new_samples,
            )

            result = self.run_script(old_path=old_path, new_path=new_path)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.count("PASS BenchmarkTier1Diagnostics_"), 6)
        self.assertIn("threshold=10.00%", result.stdout)
        self.assertIn("delta=10.00%", result.stdout)

    def test_rejects_improvement_below_threshold(self) -> None:
        new_samples = {
            row_name: make_samples(90.0)
            for row_name in REQUIRED_ROWS
        }
        target_row = (
            "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full"
        )
        new_samples[target_row] = make_samples(99.0)

        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            old_path = self.write_benchmark_file(directory, "old.bench.txt")
            new_path = self.write_benchmark_file(
                directory,
                "new.bench.txt",
                samples_by_row=new_samples,
            )

            result = self.run_script(old_path=old_path, new_path=new_path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            "FAIL BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full "
            "old=100.00 new=99.00 delta=1.00% reason=below-threshold",
            result.stdout,
        )

    def test_rejects_regression(self) -> None:
        new_samples = {
            row_name: make_samples(90.0)
            for row_name in REQUIRED_ROWS
        }
        target_row = (
            "BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-materialize-only"
        )
        new_samples[target_row] = make_samples(101.0)

        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            old_path = self.write_benchmark_file(directory, "old.bench.txt")
            new_path = self.write_benchmark_file(
                directory,
                "new.bench.txt",
                samples_by_row=new_samples,
            )

            result = self.run_script(old_path=old_path, new_path=new_path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            "FAIL BenchmarkTier1Diagnostics_citm_catalog_json/"
            "pure-simdjson-materialize-only old=100.00 new=101.00 "
            "delta=-1.00% reason=regressed",
            result.stdout,
        )

    def test_rejects_missing_required_row(self) -> None:
        new_samples = {
            row_name: make_samples(90.0)
            for row_name in REQUIRED_ROWS
        }
        missing_row = "BenchmarkTier1Diagnostics_canada_json/pure-simdjson-full"
        new_samples.pop(missing_row)

        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            old_path = self.write_benchmark_file(directory, "old.bench.txt")
            new_path = self.write_benchmark_file(
                directory,
                "new.bench.txt",
                samples_by_row=new_samples,
            )

            result = self.run_script(old_path=old_path, new_path=new_path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            "FAIL BenchmarkTier1Diagnostics_canada_json/pure-simdjson-full "
            "old=100.00 new=missing delta=n/a reason=missing-new-row",
            result.stdout,
        )

    def test_rejects_metadata_mismatch(self) -> None:
        metadata = dict(BASE_METADATA)
        metadata["cpu"] = "Apple M2"
        new_samples = {
            row_name: make_samples(90.0)
            for row_name in REQUIRED_ROWS
        }

        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            old_path = self.write_benchmark_file(directory, "old.bench.txt")
            new_path = self.write_benchmark_file(
                directory,
                "new.bench.txt",
                metadata=metadata,
                samples_by_row=new_samples,
            )

            result = self.run_script(old_path=old_path, new_path=new_path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            "FAIL metadata old_cpu=Apple M3 Max new_cpu=Apple M2 "
            "reason=metadata-mismatch",
            result.stdout,
        )


if __name__ == "__main__":
    unittest.main()
