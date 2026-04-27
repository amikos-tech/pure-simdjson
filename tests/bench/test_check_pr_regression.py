#!/usr/bin/env python3

from __future__ import annotations

import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "check_pr_regression.py"
FIXTURES_DIR = REPO_ROOT / "tests" / "bench" / "fixtures" / "pr-regression"


class CheckPRRegressionTests(unittest.TestCase):
    def run_gate(
        self,
        benchstat_path: pathlib.Path | None,
        *,
        summary_out: pathlib.Path,
        markdown_out: pathlib.Path,
        threshold_pct: float = 5.0,
        p_max: float = 0.05,
        no_baseline: bool = False,
        env: dict[str, str] | None = None,
    ) -> subprocess.CompletedProcess[str]:
        cmd = [
            sys.executable,
            str(SCRIPT_PATH),
            "--summary-out",
            str(summary_out),
            "--markdown-out",
            str(markdown_out),
            "--threshold-pct",
            str(threshold_pct),
            "--p-max",
            str(p_max),
        ]
        if no_baseline:
            cmd.append("--no-baseline")
        else:
            self.assertIsNotNone(benchstat_path)
            cmd.extend(["--benchstat-output", str(benchstat_path)])
        return subprocess.run(
            cmd,
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
            env=env,
        )

    def run_fixture(
        self,
        fixture: str,
        *,
        env: dict[str, str] | None = None,
    ) -> tuple[subprocess.CompletedProcess[str], dict[str, object], str]:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = pathlib.Path(temp_dir)
            summary_out = temp_path / "summary.json"
            markdown_out = temp_path / "markdown.md"
            result = self.run_gate(
                FIXTURES_DIR / fixture,
                summary_out=summary_out,
                markdown_out=markdown_out,
                env=env,
            )
            payload = json.loads(summary_out.read_text(encoding="utf-8")) if summary_out.exists() else {}
            markdown = markdown_out.read_text(encoding="utf-8") if markdown_out.exists() else ""
            return result, payload, markdown

    def test_row_flagged_when_slower_and_significant(self) -> None:
        result, payload, _ = self.run_fixture("slower-significant.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])
        self.assertEqual(len(payload["flagged_rows"]), 1)
        flagged = payload["flagged_rows"][0]
        self.assertEqual(flagged["row"], "Tier1FullParse_twitter_json-4")
        self.assertEqual(flagged["delta_pct"], 10.0)
        self.assertEqual(flagged["p_value"], 0.001)

    def test_row_not_flagged_when_sentinel(self) -> None:
        result, payload, _ = self.run_fixture("slower-not-significant.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])
        self.assertEqual(payload["flagged_rows"], [])

    def test_row_not_flagged_when_delta_below_threshold(self) -> None:
        result, payload, _ = self.run_fixture("slower-tiny.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])

    def test_faster_never_flagged(self) -> None:
        result, payload, _ = self.run_fixture("faster-significant.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])

    def test_boundary_exactly_5pct_flagged(self) -> None:
        result, payload, _ = self.run_fixture("boundary-5pct.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])

    def test_boundary_just_below_5pct_not_flagged(self) -> None:
        result, payload, _ = self.run_fixture("boundary-499pct.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])

    def test_per_row_granularity(self) -> None:
        result, payload, _ = self.run_fixture("mixed-multi-row.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])
        self.assertEqual(len(payload["flagged_rows"]), 1)
        self.assertEqual(payload["flagged_rows"][0]["row"], "Tier1FullParse_twitter_json-4")

    def test_metric_sections_only_sec_op_flags(self) -> None:
        result, payload, _ = self.run_fixture("mixed-metric-sections.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])
        self.assertEqual(len(payload["flagged_rows"]), 1)
        self.assertEqual(payload["flagged_rows"][0]["row"], "Tier1FullParse_twitter_json-4")

    def test_geomean_row_ignored(self) -> None:
        result, payload, _ = self.run_fixture("with-geomean.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])
        self.assertEqual(payload["flagged_rows"], [])

    def test_no_baseline_bypass_mode(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = pathlib.Path(temp_dir)
            summary_out = temp_path / "summary.json"
            markdown_out = temp_path / "markdown.md"
            result = self.run_gate(
                None,
                summary_out=summary_out,
                markdown_out=markdown_out,
                no_baseline=True,
            )
            payload = json.loads(summary_out.read_text(encoding="utf-8"))
            markdown = markdown_out.read_text(encoding="utf-8")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["bypassed"])
        self.assertFalse(payload["regression"])
        self.assertIn("advisory bypass", markdown)

    def test_advisory_always_zero_on_regression(self) -> None:
        result, payload, _ = self.run_fixture("slower-significant.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])

    def test_blocking_flip_via_env_var(self) -> None:
        env = {**os.environ, "REQUIRE_NO_REGRESSION": "true"}
        result, payload, _ = self.run_fixture("slower-significant.benchstat.txt", env=env)

        self.assertEqual(result.returncode, 1)
        self.assertTrue(payload["regression"])

    def test_blocking_flip_default_off(self) -> None:
        env = {k: v for k, v in os.environ.items() if k != "REQUIRE_NO_REGRESSION"}
        result, payload, _ = self.run_fixture("slower-significant.benchstat.txt", env=env)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(payload["regression"])

    def test_markdown_renderer_includes_row_delta_pvalue(self) -> None:
        result, _, markdown = self.run_fixture("slower-significant.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Tier1FullParse_twitter_json-4", markdown)
        self.assertIn("+10.00%", markdown)
        self.assertIn("0.001", markdown)
        self.assertIn("advisory", markdown)

    def test_malformed_input_fails_closed(self) -> None:
        result, _, _ = self.run_fixture("truncated-row.benchstat.txt")

        self.assertEqual(result.returncode, 1)

    def test_empty_input_fails_closed(self) -> None:
        result, _, _ = self.run_fixture("empty.benchstat.txt")

        self.assertEqual(result.returncode, 1)

    def test_real_phase9_benchstat_format(self) -> None:
        result, payload, _ = self.run_fixture("real-tier1-vs-stdlib.benchstat.txt")

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(payload["regression"])
        self.assertEqual(payload["flagged_rows"], [])


if __name__ == "__main__":
    unittest.main()
