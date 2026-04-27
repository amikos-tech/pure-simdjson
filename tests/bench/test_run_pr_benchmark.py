#!/usr/bin/env python3

from __future__ import annotations

import json
import os
import pathlib
import subprocess
import tempfile
import textwrap
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "run_pr_benchmark.sh"


class PRBenchmarkOrchestratorTests(unittest.TestCase):
    def test_script_locks_benchmark_subset_and_budget(self) -> None:
        script = SCRIPT_PATH.read_text(encoding="utf-8")

        self.assertIn(
            "PR_BENCH_REGEX='Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_"
            "(twitter|canada)_json/(pure-simdjson|encoding-json-any|encoding-json-struct)$'",
            script,
        )
        self.assertIn("PR_BENCH_COUNT=5", script)
        self.assertIn("PR_BENCH_TIMEOUT=600s", script)
        self.assertIn("-benchmem", script)
        self.assertIn('go test ./... -run \'^$\' -bench "$PR_BENCH_REGEX"', script)
        self.assertNotIn("citm_catalog", script)
        self.assertNotIn("minio-simdjson-go", script)
        self.assertNotIn("bytedance-sonic", script)
        self.assertNotIn("goccy-go-json", script)
        self.assertNotIn("cargo build --release", script)
        self.assertNotIn("actions/cache", script)

    def write_stub(self, directory: pathlib.Path, name: str, body: str) -> None:
        path = directory / name
        path.write_text(body, encoding="utf-8")
        path.chmod(0o755)

    def make_stub_path(self, directory: pathlib.Path) -> pathlib.Path:
        stub_dir = directory / "bin"
        stub_dir.mkdir()
        self.write_stub(
            stub_dir,
            "go",
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -euo pipefail
                echo "goos: linux"
                echo "goarch: amd64"
                echo "pkg: github.com/amikos-tech/pure-simdjson"
                echo "cpu: Synthetic CPU"
                for run in 1 2 3 4 5; do
                  for fixture in twitter_json canada_json; do
                    for cmp in pure-simdjson encoding-json-any encoding-json-struct; do
                      echo "BenchmarkTier1FullParse_${fixture}/${cmp}-4 ${run} 2000000 ns/op 64 B/op 1 allocs/op"
                      echo "BenchmarkTier2Typed_${fixture}/${cmp}-4 ${run} 1500000 ns/op 64 B/op 1 allocs/op"
                    done
                  done
                  for cmp in pure-simdjson encoding-json-any encoding-json-struct; do
                    echo "BenchmarkTier3SelectivePlaceholder_twitter_json/${cmp}-4 ${run} 1000000 ns/op 64 B/op 1 allocs/op"
                  done
                done
                echo "PASS"
                """
            ),
        )
        self.write_stub(
            stub_dir,
            "benchstat",
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -euo pipefail
                cat <<'OUT'
                goos: linux
                goarch: amd64
                pkg: github.com/amikos-tech/pure-simdjson
                cpu: Synthetic CPU
                                                   | baseline.bench.txt | head.bench.txt |
                                                   | sec/op             | sec/op vs base |
                Tier1FullParse_twitter_json-4        2.000m +/- 1%       2.200m +/- 1%   +10.00% (p=0.001 n=5)
                OUT
                """
            ),
        )
        return stub_dir

    def run_script(
        self,
        args: list[str],
        *,
        stub_dir: pathlib.Path,
    ) -> subprocess.CompletedProcess[str]:
        env = {**os.environ, "PATH": f"{stub_dir}:{os.environ['PATH']}"}
        return subprocess.run(
            [str(SCRIPT_PATH), *args],
            cwd=REPO_ROOT,
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_with_baseline_produces_full_output_set(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = pathlib.Path(temp_dir)
            stub_dir = self.make_stub_path(root)
            baseline = root / "baseline.bench.txt"
            baseline.write_text("BenchmarkBaseline 1 100 ns/op\n", encoding="utf-8")
            out_dir = root / "out"

            result = self.run_script(
                ["--baseline", str(baseline), "--out-dir", str(out_dir)],
                stub_dir=stub_dir,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                {path.name for path in out_dir.iterdir()},
                {
                    "head.bench.txt",
                    "baseline.bench.txt",
                    "regression.benchstat.txt",
                    "summary.json",
                    "markdown.md",
                },
            )
            payload = json.loads((out_dir / "summary.json").read_text(encoding="utf-8"))
            self.assertTrue(payload["regression"])

    def test_no_baseline_skips_benchstat(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = pathlib.Path(temp_dir)
            stub_dir = self.make_stub_path(root)
            out_dir = root / "out"

            result = self.run_script(
                ["--no-baseline", "--out-dir", str(out_dir)],
                stub_dir=stub_dir,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse((out_dir / "regression.benchstat.txt").exists())
            payload = json.loads((out_dir / "summary.json").read_text(encoding="utf-8"))
            self.assertTrue(payload["bypassed"])

    def test_missing_baseline_path_errors_clearly(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = pathlib.Path(temp_dir)
            stub_dir = self.make_stub_path(root)
            out_dir = root / "out"

            result = self.run_script(
                ["--baseline", str(root / "missing.bench.txt"), "--out-dir", str(out_dir)],
                stub_dir=stub_dir,
            )

            self.assertEqual(result.returncode, 1)
            self.assertIn("baseline benchmark file not found", result.stderr)


if __name__ == "__main__":
    unittest.main()
