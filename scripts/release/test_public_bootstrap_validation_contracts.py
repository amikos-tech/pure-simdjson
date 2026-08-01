#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import unittest

try:
    from scripts.release.workflow_yaml import load_workflow_definition
except ModuleNotFoundError:  # pragma: no cover - direct script invocation
    from workflow_yaml import load_workflow_definition


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "public-bootstrap-validation.yml"
RELEASES_DOC = REPO_ROOT / "docs" / "releases.md"
BOOTSTRAP_DOC = REPO_ROOT / "docs" / "bootstrap.md"
GO_BOOTSTRAP_SMOKE = REPO_ROOT / "tests" / "smoke" / "go_bootstrap_smoke.go"
RUN_PUBLIC_BOOTSTRAP_SMOKE = (
    REPO_ROOT / "scripts" / "release" / "run_public_bootstrap_smoke.sh"
)

def workflow_definition() -> dict[object, object]:
    return load_workflow_definition(WORKFLOW, REPO_ROOT)


def job_step(job: dict[object, object], name: str) -> dict[object, object]:
    for step in job["steps"]:
        if step.get("name") == name:
            return step
    raise AssertionError(f"step {name!r} not found")


def matrix_platform_ids(job: dict[object, object]) -> tuple[str, ...]:
    return tuple(entry["platform_id"] for entry in job["strategy"]["matrix"]["include"])


def is_true(value: object) -> bool:
    return str(value).lower() == "true"


def is_false(value: object) -> bool:
    return str(value).lower() == "false"


class PublicBootstrapValidationContractTests(unittest.TestCase):
    def test_public_bands_execute_published_tag_abi_1_2_smoke(self) -> None:
        workflow = workflow_definition()
        wrapper_text = RUN_PUBLIC_BOOTSTRAP_SMOKE.read_text(encoding="utf-8")
        smoke_text = GO_BOOTSTRAP_SMOKE.read_text(encoding="utf-8")

        for job_name, smoke_step_name in (
            ("validate-r2", "Run public bootstrap smoke (r2)"),
            (
                "validate-gh-fallback",
                "Run public bootstrap smoke (github-fallback)",
            ),
        ):
            job = workflow["jobs"][job_name]
            validate_step = job_step(job, "Validate target tag source")
            smoke_step = job_step(job, smoke_step_name)
            self.assertIn(
                "target-src/tests/smoke/go_bootstrap_smoke.go",
                validate_step["run"],
            )
            self.assertIn(
                "scripts/release/run_public_bootstrap_smoke.sh",
                smoke_step["run"],
            )

        self.assertIn(
            '(cd "$repo_root" && go run ./tests/smoke/go_bootstrap_smoke.go)',
            wrapper_text,
        )
        for snippet in (
            "WithMaxCapacity",
            "WithMaxDepth",
            "TypeBigInt",
            "GetBigInt",
        ):
            self.assertIn(snippet, smoke_text)

    def test_bootstrap_docs_mark_0_1_7_as_historical(self) -> None:
        bootstrap_text = BOOTSTRAP_DOC.read_text(encoding="utf-8")

        for snippet in (
            "v0.1.7",
            "historical ABI",
            "ErrABIVersionMismatch",
            "Phase `06.1`",
        ):
            self.assertIn(snippet, bootstrap_text)

    def test_workflow_exists_and_has_manual_plus_scheduled_triggers(self) -> None:
        workflow = workflow_definition()

        self.assertTrue(WORKFLOW.exists())
        self.assertIn("workflow_dispatch", workflow["on"])
        self.assertIn("schedule", workflow["on"])

        version_input = workflow["on"]["workflow_dispatch"]["inputs"]["version"]
        self.assertTrue(is_true(version_input["required"]))
        self.assertEqual(version_input["type"], "string")
        self.assertEqual(
            workflow["concurrency"]["group"],
            "public-bootstrap-validation-${{ github.event_name == 'workflow_dispatch' && inputs.version || 'scheduled' }}",
        )
        self.assertTrue(is_false(workflow["concurrency"]["cancel-in-progress"]))

    def test_workflow_keeps_top_level_permissions_read_only(self) -> None:
        workflow = workflow_definition()

        self.assertEqual(workflow["permissions"]["contents"], "read")
        self.assertEqual(workflow["permissions"]["actions"], "read")

    def test_resolve_version_job_fetches_latest_json_and_validates_semver(self) -> None:
        workflow = workflow_definition()
        resolve_job = workflow["jobs"]["resolve-version"]
        resolve_step = job_step(resolve_job, "Resolve target version")
        run_script = resolve_step["run"]

        self.assertEqual(str(resolve_job["timeout-minutes"]), "5")
        self.assertEqual(resolve_job["defaults"]["run"]["shell"], "bash")
        self.assertIn("latest.json", run_script)
        self.assertIn("INPUT_VERSION", resolve_step["env"])
        self.assertIn("^", run_script)
        self.assertIn("[0-9]+\\.[0-9]+\\.[0-9]+", run_script)
        self.assertIn("validation version must match", run_script)

    def test_jobs_pin_actions_and_checkout_target_src(self) -> None:
        workflow = workflow_definition()

        for job_name in ("validate-r2", "validate-gh-fallback"):
            job = workflow["jobs"][job_name]
            current_checkout = job_step(job, "Check out current branch")
            target_checkout = job_step(job, "Check out published tag into target-src")
            setup_go = job_step(job, "Set up Go")

            self.assertEqual(
                current_checkout["uses"],
                "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683",
            )
            self.assertEqual(
                target_checkout["uses"],
                "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683",
            )
            self.assertEqual(
                setup_go["uses"],
                "actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff",
            )
            self.assertEqual(
                target_checkout["with"]["ref"],
                "refs/tags/v${{ needs.resolve-version.outputs.version }}",
            )
            self.assertEqual(target_checkout["with"]["path"], "target-src")

    def test_jobs_force_bash_timeout_and_safe_target_validation(self) -> None:
        workflow = workflow_definition()

        for job_name in ("validate-r2", "validate-gh-fallback"):
            job = workflow["jobs"][job_name]
            validate_step = job_step(job, "Validate target tag source")
            smoke_step_name = (
                "Run public bootstrap smoke (r2)"
                if job_name == "validate-r2"
                else "Run public bootstrap smoke (github-fallback)"
            )
            smoke_step = job_step(job, smoke_step_name)

            self.assertEqual(job["needs"], "resolve-version")
            self.assertEqual(job["defaults"]["run"]["shell"], "bash")
            self.assertEqual(str(job["timeout-minutes"]), "20")
            self.assertEqual(job["env"]["VERSION"], "${{ needs.resolve-version.outputs.version }}")
            self.assertIn('grep -F "const Version = \\"$VERSION\\""', validate_step["run"])
            self.assertNotIn("${{ inputs.version }}", validate_step["run"])
            self.assertIn("target-src/go.mod", validate_step["run"])
            self.assertIn("target-src/internal/bootstrap/version.go", validate_step["run"])
            self.assertIn("target-src/tests/smoke/go_bootstrap_smoke.go", validate_step["run"])
            self.assertIn('--version "$VERSION"', smoke_step["run"])

    def test_validate_r2_job_covers_full_matrix_in_r2_mode(self) -> None:
        workflow = workflow_definition()
        r2_job = workflow["jobs"]["validate-r2"]
        smoke_step = job_step(r2_job, "Run public bootstrap smoke (r2)")

        self.assertEqual(
            matrix_platform_ids(r2_job),
            (
                "linux-amd64",
                "linux-arm64",
                "darwin-amd64",
                "darwin-arm64",
                "windows-amd64",
            ),
        )
        self.assertTrue(is_false(r2_job["strategy"]["fail-fast"]))
        self.assertIn("--mode r2", smoke_step["run"])

    def test_validate_gh_fallback_job_uses_representative_subset(self) -> None:
        workflow = workflow_definition()
        fallback_job = workflow["jobs"]["validate-gh-fallback"]
        smoke_step = job_step(fallback_job, "Run public bootstrap smoke (github-fallback)")

        self.assertEqual(
            matrix_platform_ids(fallback_job),
            ("linux-amd64", "darwin-arm64", "windows-amd64"),
        )
        self.assertTrue(is_false(fallback_job["strategy"]["fail-fast"]))
        self.assertIn("--mode github-fallback", smoke_step["run"])

    def test_scheduled_failure_notification_job_opens_or_updates_issue(self) -> None:
        workflow = workflow_definition()
        notify_job = workflow["jobs"]["notify-scheduled-failure"]
        notify_step = job_step(notify_job, "Open or update scheduled validation issue")

        self.assertEqual(
            notify_job["needs"],
            ["resolve-version", "validate-r2", "validate-gh-fallback"],
        )
        self.assertIn("github.event_name == 'schedule'", notify_job["if"])
        self.assertEqual(notify_job["permissions"]["issues"], "write")
        self.assertIn("gh issue list", notify_step["run"])
        self.assertIn("gh issue comment", notify_step["run"])
        self.assertIn("gh issue create", notify_step["run"])

    def test_docs_define_phase_06_1_boundary_and_entrypoint(self) -> None:
        releases_text = RELEASES_DOC.read_text(encoding="utf-8")
        bootstrap_text = BOOTSTRAP_DOC.read_text(encoding="utf-8")

        self.assertTrue(RELEASES_DOC.exists())
        self.assertTrue(BOOTSTRAP_DOC.exists())
        for snippet in (
            "public-bootstrap-validation.yml",
            "Phase `06.1`",
            "target-src",
            "GitHub fallback",
            "SHA256SUMS",
        ):
            self.assertIn(snippet, releases_text)

        for snippet in (
            "public-bootstrap-validation.yml",
            "Phase `06.1`",
            "target-src",
            "SHA256SUMS",
        ):
            self.assertIn(snippet, bootstrap_text)


if __name__ == "__main__":
    unittest.main()
