#!/usr/bin/env python3

from __future__ import annotations

import json
import pathlib
import subprocess
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "benchmark-capture.yml"
PHASE9_DIR = REPO_ROOT / "testdata" / "benchmark-results" / "v0.1.2"
BASELINE_DIR = REPO_ROOT / "testdata" / "benchmark-results" / "v0.1.1-linux-amd64"
README = REPO_ROOT / "README.md"
CHANGELOG = REPO_ROOT / "CHANGELOG.md"
METHODOLOGY = REPO_ROOT / "docs" / "benchmarks.md"
RESULTS = REPO_ROOT / "docs" / "benchmarks" / "results-v0.1.2.md"
CAPTURE_SCRIPT = REPO_ROOT / "scripts" / "bench" / "capture_release_snapshot.sh"

try:
    import yaml
except ModuleNotFoundError:  # pragma: no cover - exercised via yq fallback when PyYAML is absent
    yaml = None


if yaml is not None:
    class UniqueKeyLoader(yaml.BaseLoader):
        pass


    def _construct_unique_mapping(
        loader: UniqueKeyLoader, node: yaml.nodes.MappingNode, deep: bool = False
    ) -> dict[object, object]:
        mapping: dict[object, object] = {}
        for key_node, value_node in node.value:
            key = loader.construct_object(key_node, deep=deep)
            if key in mapping:
                raise AssertionError(f"duplicate YAML key: {key!r}")
            mapping[key] = loader.construct_object(value_node, deep=deep)
        return mapping


    UniqueKeyLoader.add_constructor(
        yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
        _construct_unique_mapping,
    )


def load_workflow_definition() -> dict[object, object]:
    workflow_text = WORKFLOW.read_text(encoding="utf-8")
    if yaml is not None:
        workflow = yaml.load(workflow_text, Loader=UniqueKeyLoader)
        if not isinstance(workflow, dict):
            raise AssertionError(f"expected workflow mapping, got {type(workflow)!r}")
        return workflow

    result = subprocess.run(
        ["yq", "-o=json", str(WORKFLOW)],
        cwd=REPO_ROOT,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise AssertionError(f"unable to parse workflow YAML: {result.stderr.strip()}")
    workflow = json.loads(result.stdout)
    if not isinstance(workflow, dict):
        raise AssertionError(f"expected workflow mapping, got {type(workflow)!r}")
    return workflow


def is_true(value: object) -> bool:
    return str(value).lower() == "true"


def is_false(value: object) -> bool:
    return str(value).lower() == "false"


def job_step(job: dict[object, object], name: str) -> dict[object, object]:
    for step in job["steps"]:
        if step.get("name") == name:
            return step
    raise AssertionError(f"step {name!r} not found")


def load_json(path: pathlib.Path) -> dict[object, object]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise AssertionError(f"expected JSON object in {path}")
    return payload


class Phase9ValidationContractTests(unittest.TestCase):
    def test_benchmark_capture_workflow_contract(self) -> None:
        workflow = load_workflow_definition()
        workflow_text = WORKFLOW.read_text(encoding="utf-8")

        self.assertTrue(WORKFLOW.exists())
        self.assertIn("workflow_dispatch", workflow["on"])

        snapshot_input = workflow["on"]["workflow_dispatch"]["inputs"]["snapshot"]
        self.assertEqual(snapshot_input["default"], "v0.1.2")
        self.assertEqual(snapshot_input["type"], "string")
        self.assertTrue(is_true(snapshot_input["required"]))

        self.assertEqual(workflow["concurrency"]["group"], "benchmark-capture-${{ inputs.snapshot }}")
        self.assertTrue(is_false(workflow["concurrency"]["cancel-in-progress"]))
        self.assertEqual(workflow["permissions"]["contents"], "read")
        self.assertEqual(workflow["permissions"]["actions"], "read")

        capture_job = workflow["jobs"]["capture"]
        self.assertEqual(capture_job["runs-on"], "ubuntu-latest")
        self.assertEqual(capture_job["defaults"]["run"]["shell"], "bash")

        for step_name in (
            "Check out repository",
            "Set up Go",
            "Install pinned Rust toolchain",
            "Install benchstat",
            "Build native release library",
            "Run comparator and oracle smoke checks",
            "Capture benchmark evidence",
            "Upload benchmark evidence",
        ):
            job_step(capture_job, step_name)

        upload_step = job_step(capture_job, "Upload benchmark evidence")
        self.assertEqual(upload_step["with"]["name"], "benchmark-evidence-${{ inputs.snapshot }}-linux-amd64")
        self.assertEqual(upload_step["with"]["retention-days"], "30")

        for forbidden in ("pages: write", "id-token: write", "git push", "git tag"):
            self.assertNotIn(forbidden, workflow_text)

    def test_phase9_evidence_snapshot_contract(self) -> None:
        required_phase9_files = (
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
            "summary.json",
        )
        required_baseline_files = (
            "phase7.bench.txt",
            "coldwarm.bench.txt",
            "tier1-diagnostics.bench.txt",
            "metadata.json",
        )

        for filename in required_phase9_files:
            path = PHASE9_DIR / filename
            self.assertTrue(path.exists(), filename)
            self.assertGreater(path.stat().st_size, 0, filename)

        for filename in required_baseline_files:
            path = BASELINE_DIR / filename
            self.assertTrue(path.exists(), filename)
            self.assertGreater(path.stat().st_size, 0, filename)

        metadata = load_json(PHASE9_DIR / "metadata.json")
        summary = load_json(PHASE9_DIR / "summary.json")

        self.assertEqual(metadata["snapshot"], "v0.1.2")
        self.assertEqual(metadata["goos"], "linux")
        self.assertEqual(metadata["goarch"], "amd64")
        self.assertIn("commands", metadata)

        target = summary["target"]
        claims = summary["claims"]
        errors = summary["errors"]

        self.assertEqual(summary["snapshot"], "v0.1.2")
        self.assertEqual(target["goos"], "linux")
        self.assertEqual(target["goarch"], "amd64")
        self.assertEqual(errors, [])
        self.assertIn(
            claims["readme_mode"],
            {
                "tier1_headline",
                "tier1_improved_but_tier2_tier3_headline",
                "conservative_current_strengths",
            },
        )
        for key in (
            "tier1_headline_allowed",
            "tier2_headline_allowed",
            "tier3_headline_allowed",
        ):
            self.assertIn(key, claims)

    def test_phase9_docs_contract(self) -> None:
        readme = README.read_text(encoding="utf-8")
        methodology = METHODOLOGY.read_text(encoding="utf-8")
        results = RESULTS.read_text(encoding="utf-8")
        changelog = CHANGELOG.read_text(encoding="utf-8")

        self.assertIn("[benchmark methodology](docs/benchmarks.md)", readme)
        self.assertIn("[v0.1.2 results](docs/benchmarks/results-v0.1.2.md)", readme)
        self.assertIn("linux/amd64", readme)
        self.assertIn("other platforms may differ", readme)
        self.assertIn("Phase 09.1", readme)
        for forbidden in ("minio-simdjson-go", "bytedance-sonic", "goccy-go-json", "git tag", "git push"):
            self.assertNotIn(forbidden, readme)

        for snippet in (
            "testdata/benchmark-results/v0.1.2",
            "results-v0.1.2.md",
            "GitHub Actions artifacts are retention-limited transport",
            "durable source of truth",
            "testdata/benchmark-results/v0.1.1-linux-amd64",
            "Headline numbers come from linux/amd64; other platforms may differ.",
        ):
            self.assertIn(snippet, methodology)

        for snippet in (
            "# Benchmark Results v0.1.2",
            "## Status",
            "## Target and Raw Evidence",
            "## Claim Gate Summary",
            "## Tier 1: Full Parse + Full any Materialization",
            "## Tier 1 Diagnostics",
            "## Tier 2: Typed Extraction",
            "## Tier 3: Selective Traversal on the Current DOM API",
            "## Cold vs Warm Parser Lifecycle",
            "## Comparator Notes",
            "## Release Boundary",
            "summary.json",
            "metadata.json",
            "linux/amd64",
            "Phase 09.1",
            "docs/releases.md",
            "scripts/release/check_readiness.sh --strict --version 0.1.2",
        ):
            self.assertIn(snippet, results)

        self.assertIn("## [Unreleased]", changelog)
        self.assertIn("testdata/benchmark-results/v0.1.2", changelog)
        self.assertIn("docs/benchmarks/results-v0.1.2.md", changelog)
        self.assertIn("benchmark-positioning", changelog)
        for forbidden in ("git tag", "git push", "default-install validation"):
            self.assertNotIn(forbidden, changelog)

    def test_capture_script_preserves_failed_snapshot_separately(self) -> None:
        script = CAPTURE_SCRIPT.read_text(encoding="utf-8")

        self.assertTrue(CAPTURE_SCRIPT.exists())
        self.assertIn(".failed.", script)
        self.assertNotIn("promote_stage\n\texit 1", script)
        self.assertNotIn("promote_stage\n    exit 1", script)


if __name__ == "__main__":
    unittest.main()
