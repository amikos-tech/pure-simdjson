#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "public-bootstrap-validation.yml"
WRAPPER = REPO_ROOT / "scripts" / "release" / "run_public_bootstrap_smoke.sh"


class PublicBootstrapValidationContractTests(unittest.TestCase):
    def test_workflow_exists_and_is_dispatchable(self) -> None:
        workflow_text = WORKFLOW.read_text(encoding="utf-8")

        self.assertTrue(WORKFLOW.exists())
        self.assertIn("workflow_dispatch:", workflow_text)
        self.assertIn("version:", workflow_text)
        self.assertIn("required: true", workflow_text)
        self.assertIn("public-bootstrap-validation-${{ inputs.version }}", workflow_text)

    def test_workflow_pins_actions_and_checks_out_target_src(self) -> None:
        workflow_text = WORKFLOW.read_text(encoding="utf-8")

        self.assertIn(
            "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683",
            workflow_text,
        )
        self.assertIn(
            "actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff",
            workflow_text,
        )
        self.assertIn("path: target-src", workflow_text)
        self.assertIn("refs/tags/v${{ inputs.version }}", workflow_text)

    def test_jobs_force_bash_timeout_and_target_validation(self) -> None:
        workflow_text = WORKFLOW.read_text(encoding="utf-8")
        r2_section = workflow_text.split("  validate-r2:", 1)[1].split("  validate-gh-fallback:", 1)[0]
        fallback_section = workflow_text.split("  validate-gh-fallback:", 1)[1]

        for section in (r2_section, fallback_section):
            self.assertIn("shell: bash", section)
            self.assertIn("timeout-minutes: 20", section)
            self.assertIn("Validate target tag source", section)
            self.assertIn("target-src/go.mod", section)
            self.assertIn("target-src/internal/bootstrap/version.go", section)
            self.assertIn("target-src/tests/smoke/go_bootstrap_smoke.go", section)

    def test_workflow_covers_required_matrices_and_modes(self) -> None:
        workflow_text = WORKFLOW.read_text(encoding="utf-8")

        for target in (
            "linux-amd64",
            "linux-arm64",
            "darwin-amd64",
            "darwin-arm64",
            "windows-amd64",
        ):
            self.assertIn(target, workflow_text)
        self.assertIn("--mode r2", workflow_text)
        self.assertIn("--mode github-fallback", workflow_text)

    def test_wrapper_contract_stays_locked(self) -> None:
        wrapper_text = WRAPPER.read_text(encoding="utf-8")

        self.assertTrue(WRAPPER.exists())
        for snippet in (
            "refusing unsafe cache dir",
            "PURE_SIMDJSON_DISABLE_GH_FALLBACK=1",
            "PURE_SIMDJSON_BINARY_MIRROR=\"https://releases.amikos.tech/pure-simdjson-does-not-exist\"",
            "stat -c '%a'",
            "stat -f '%Lp'",
            "SHA256SUMS",
            "v${version}/${goos}-${goarch}",
        ):
            self.assertIn(snippet, wrapper_text)


if __name__ == "__main__":
    unittest.main()
