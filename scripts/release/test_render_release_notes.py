#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import pathlib
import subprocess
import tempfile
import textwrap
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "release" / "render_release_notes.py"
SPEC = importlib.util.spec_from_file_location("render_release_notes", SCRIPT_PATH)
assert SPEC is not None
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)

SAMPLE_CHANGELOG = textwrap.dedent(
    """\
    # Changelog

    ## [Unreleased]

    ### Changed
    - Future work.

    ## [1.2.0] - 2026-04-22

    ### Added
    - Brand new thing.

    ## [1.1.0] - 2026-04-21

    ### Fixed
    - Existing bug.
    """
)


class RenderReleaseNotesTests(unittest.TestCase):
    def write_sample_changelog(self, tmpdir: str) -> pathlib.Path:
        changelog_path = pathlib.Path(tmpdir) / "CHANGELOG.md"
        changelog_path.write_text(SAMPLE_CHANGELOG, encoding="utf-8")
        return changelog_path

    def test_normalize_version_accepts_tag_prefixes(self) -> None:
        self.assertEqual(MODULE.normalize_version("1.2.0"), "1.2.0")
        self.assertEqual(MODULE.normalize_version("v1.2.0"), "1.2.0")
        self.assertEqual(MODULE.normalize_version("refs/tags/v1.2.0"), "1.2.0")

    def test_extract_release_section_returns_exact_tagged_entry(self) -> None:
        rendered = MODULE.extract_release_section(SAMPLE_CHANGELOG, "1.2.0")

        self.assertEqual(
            rendered,
            "## [1.2.0] - 2026-04-22\n\n### Added\n- Brand new thing.\n",
        )

    def test_cli_writes_requested_changelog_entry(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_sample_changelog(tmpdir)
            output_path = pathlib.Path(tmpdir) / "release-notes.md"

            result = subprocess.run(
                [
                    "python3",
                    str(SCRIPT_PATH),
                    "--changelog",
                    str(changelog_path),
                    "--version",
                    "refs/tags/v1.1.0",
                    "--output",
                    str(output_path),
                ],
                cwd=REPO_ROOT,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0, msg=result.stderr)
            self.assertEqual(
                output_path.read_text(encoding="utf-8"),
                "## [1.1.0] - 2026-04-21\n\n### Fixed\n- Existing bug.\n",
            )

    def test_cli_reports_missing_version(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_sample_changelog(tmpdir)

            result = subprocess.run(
                [
                    "python3",
                    str(SCRIPT_PATH),
                    "--changelog",
                    str(changelog_path),
                    "--version",
                    "v9.9.9",
                ],
                cwd=REPO_ROOT,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("version '9.9.9' not found in changelog", result.stderr)


if __name__ == "__main__":
    unittest.main()
