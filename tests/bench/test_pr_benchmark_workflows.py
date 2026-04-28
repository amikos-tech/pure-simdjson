#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import re
import shutil
import subprocess
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
PR_WORKFLOW = REPO_ROOT / ".github" / "workflows" / "pr-benchmark.yml"
MAIN_WORKFLOW = REPO_ROOT / ".github" / "workflows" / "main-benchmark-baseline.yml"
CHANGELOG = REPO_ROOT / "CHANGELOG.md"

PATHS_IGNORE = [
    "- '**.md'",
    "- 'docs/**'",
    "- 'LICENSE'",
    "- 'NOTICE'",
    "- '.planning/**'",
    "- '.github/workflows/**'",
    "- '.github/actions/**'",
    "- 'testdata/benchmark-results/**'",
]


class PRBenchmarkWorkflowContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.pr_text = PR_WORKFLOW.read_text(encoding="utf-8")
        cls.main_text = MAIN_WORKFLOW.read_text(encoding="utf-8")
        cls.changelog_text = CHANGELOG.read_text(encoding="utf-8")

    def assertContainsAll(self, text: str, snippets: list[str]) -> None:
        for snippet in snippets:
            with self.subTest(snippet=snippet):
                self.assertIn(snippet, text)

    def test_pr_workflow_trigger_permissions_concurrency_and_budget(self) -> None:
        self.assertIn("name: pr benchmark", self.pr_text)
        self.assertIn("pull_request:", self.pr_text)
        self.assertNotIn("pull_request_target", self.pr_text)
        self.assertContainsAll(self.pr_text, PATHS_IGNORE)
        self.assertContainsAll(
            self.pr_text,
            [
                "group: pr-bench-${{ github.event.pull_request.number }}",
                "cancel-in-progress: true",
                "contents: read",
                "pull-requests: write",
                "timeout-minutes: 15",
            ],
        )

    def test_pr_workflow_restores_baseline_and_stays_advisory(self) -> None:
        self.assertContainsAll(
            self.pr_text,
            [
                "actions/cache/restore@0400d5f644dc74513175e3cd8d07132dd4860809",
                "path: baseline.bench.txt",
                "Deliberately misses the exact key so restore-keys selects the newest main baseline.",
                "key: pr-bench-baseline-NEVER-MATCHES",
                "pr-bench-baseline-",
                'REQUIRE_NO_REGRESSION: "false"   # Set to "true" to make this check blocking instead of advisory.',
                "NO_BASELINE: ${{ steps.restore-baseline.outputs.cache-matched-key == '' }}",
                "bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir pr-bench-summary",
                "bash scripts/bench/run_pr_benchmark.sh --baseline baseline.bench.txt --out-dir pr-bench-summary",
            ],
        )
        self.assertNotIn("actions/cache/save", self.pr_text)

    def test_pr_workflow_surfaces_results_without_requiring_comment_success(self) -> None:
        self.assertContainsAll(
            self.pr_text,
            [
                'cat pr-bench-summary/markdown.md >>"$GITHUB_STEP_SUMMARY"',
                "continue-on-error: true",
                "marocchino/sticky-pull-request-comment@0ea0beb66eb9baf113663a64ec522f60e49231c0",
                "header: pr-benchmark-regression",
                "path: pr-bench-summary/markdown.md",
                "actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
                "path: pr-bench-summary/",
                "retention-days: 14",
                "if-no-files-found: warn",
            ],
        )

    def test_main_baseline_workflow_writes_cache_from_main_only(self) -> None:
        self.assertIn("name: main benchmark baseline", self.main_text)
        self.assertContainsAll(
            self.main_text,
            [
                "push:",
                "branches: [main]",
                "workflow_dispatch:",
                "group: main-bench-baseline",
                "cancel-in-progress: false",
                "contents: read",
                "bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir pr-bench-summary",
                "cp pr-bench-summary/head.bench.txt baseline.bench.txt",
                "actions/cache/save@0400d5f644dc74513175e3cd8d07132dd4860809",
                "if: success()",
                "path: baseline.bench.txt",
                "key: pr-bench-baseline-${{ github.sha }}",
                "retention-days: 30",
            ],
        )
        self.assertNotIn("pull-requests: write", self.main_text)
        self.assertContainsAll(self.main_text, PATHS_IGNORE)

    def test_third_party_actions_are_pinned_to_commit_shas(self) -> None:
        combined = f"{self.pr_text}\n{self.main_text}"
        for line in combined.splitlines():
            stripped = line.strip()
            if not stripped.startswith("uses: "):
                continue
            action = stripped.removeprefix("uses: ").split()[0]
            if action.startswith("./"):
                continue
            with self.subTest(action=action):
                self.assertRegex(action, r"@[0-9a-f]{40}$")
                self.assertNotRegex(action, r"@(v\d+|main|master)$")

    def test_workflows_keep_benchmark_selection_inside_orchestrator(self) -> None:
        combined = f"{self.pr_text}\n{self.main_text}"
        self.assertIn("bash scripts/bench/run_pr_benchmark.sh", combined)
        self.assertNotIn("go test -bench", combined)
        self.assertNotIn("BenchmarkTier", combined)
        self.assertNotIn("testdata/benchmark-results/v", combined)
        self.assertNotIn("citm_catalog", combined)
        self.assertNotRegex(combined, re.compile(r"minio-simdjson-go|bytedance-sonic|goccy-go-json"))

    def test_blocking_flip_is_discoverable_from_workflow_and_changelog(self) -> None:
        self.assertIn("REQUIRE_NO_REGRESSION", self.pr_text)
        self.assertIn('Set to "true" to make this check blocking instead of advisory.', self.pr_text)
        self.assertNotIn("Future blocking-flip control knob", self.pr_text)
        self.assertIn("REQUIRE_NO_REGRESSION", self.changelog_text)
        self.assertIn("advisory", self.changelog_text)

    @unittest.skipUnless(shutil.which("yq"), "yq is required for workflow YAML smoke tests")
    def test_workflows_parse_as_yaml(self) -> None:
        for workflow in (PR_WORKFLOW, MAIN_WORKFLOW):
            with self.subTest(workflow=workflow.name):
                result = subprocess.run(
                    ["yq", "eval", ".", str(workflow)],
                    cwd=REPO_ROOT,
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stderr, "")


if __name__ == "__main__":
    unittest.main()
