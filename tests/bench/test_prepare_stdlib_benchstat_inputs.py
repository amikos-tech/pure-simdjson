#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "prepare_stdlib_benchstat_inputs.py"


def source_text(include_missing: bool = False) -> str:
    lines = [
        "goos: linux",
        "goarch: amd64",
        "pkg: github.com/amikos-tech/pure-simdjson",
        "cpu: Synthetic CPU",
    ]
    for fixture in ("twitter_json", "citm_catalog_json", "canada_json"):
        lines.append(
            f"BenchmarkTier1FullParse_{fixture}/encoding-json-any-8\t10\t100.00 ns/op\t0 B/op\t0 allocs/op"
        )
        if not (include_missing and fixture == "canada_json"):
            lines.append(
                f"BenchmarkTier1FullParse_{fixture}/pure-simdjson-8\t10\t80.00 ns/op\t0 B/op\t0 allocs/op"
            )
        lines.append(
            f"BenchmarkTier2Typed_{fixture}/encoding-json-struct-8\t10\t90.00 ns/op\t0 B/op\t0 allocs/op"
        )
        lines.append(
            f"BenchmarkTier2Typed_{fixture}/pure-simdjson-8\t10\t50.00 ns/op\t0 B/op\t0 allocs/op"
        )
    for fixture in ("twitter_json", "citm_catalog_json"):
        lines.append(
            f"BenchmarkTier3SelectivePlaceholder_{fixture}/encoding-json-struct-8\t10\t70.00 ns/op\t0 B/op\t0 allocs/op"
        )
        lines.append(
            f"BenchmarkTier3SelectivePlaceholder_{fixture}/pure-simdjson-8\t10\t30.00 ns/op\t0 B/op\t0 allocs/op"
        )
    lines.append("PASS")
    return "\n".join(lines) + "\n"


class PrepareStdlibBenchstatInputsTests(unittest.TestCase):
    def run_helper(
        self,
        directory: pathlib.Path,
        *,
        family: str = "tier1",
        missing: bool = False,
    ) -> subprocess.CompletedProcess[str]:
        source = directory / "phase9.bench.txt"
        source.write_text(source_text(include_missing=missing), encoding="utf-8")
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT_PATH),
                "--source",
                str(source),
                "--family",
                family,
                "--base-comparator",
                "encoding-json-any" if family == "tier1" else "encoding-json-struct",
                "--candidate-comparator",
                "pure-simdjson",
                "--left-out",
                str(directory / "left.bench.txt"),
                "--right-out",
                str(directory / "right.bench.txt"),
            ],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_normalizes_comparator_suffixes_and_preserves_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            result = self.run_helper(directory)
            left = (directory / "left.bench.txt").read_text(encoding="utf-8")
            right = (directory / "right.bench.txt").read_text(encoding="utf-8")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("goos: linux", left)
        self.assertIn("BenchmarkTier1FullParse_twitter_json-8", left)
        self.assertIn("BenchmarkTier1FullParse_twitter_json-8", right)
        self.assertNotIn("/encoding-json-any-", left)
        self.assertNotIn("/pure-simdjson-", right)

    def test_tier3_uses_only_published_selective_fixtures(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            result = self.run_helper(directory, family="tier3")
            right = (directory / "right.bench.txt").read_text(encoding="utf-8")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("BenchmarkTier3SelectivePlaceholder_twitter_json-8", right)
        self.assertIn("BenchmarkTier3SelectivePlaceholder_citm_catalog_json-8", right)
        self.assertNotIn("BenchmarkTier3SelectivePlaceholder_canada_json", right)

    def test_missing_expected_fixture_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            result = self.run_helper(pathlib.Path(temp_dir), missing=True)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("BenchmarkTier1FullParse_canada_json", result.stderr)

    def test_truncated_benchmark_row_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            directory = pathlib.Path(temp_dir)
            source = directory / "phase9.bench.txt"
            source.write_text(
                source_text()
                + "BenchmarkTier1FullParse_twitter_json/pure-simdjson   <truncated\n",
                encoding="utf-8",
            )
            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT_PATH),
                    "--source",
                    str(source),
                    "--family",
                    "tier1",
                    "--base-comparator",
                    "encoding-json-any",
                    "--candidate-comparator",
                    "pure-simdjson",
                    "--left-out",
                    str(directory / "left.bench.txt"),
                    "--right-out",
                    str(directory / "right.bench.txt"),
                ],
                cwd=REPO_ROOT,
                capture_output=True,
                text=True,
                check=False,
            )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unparseable benchmark row", result.stderr)


if __name__ == "__main__":
    unittest.main()
